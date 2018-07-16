package yke

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"yunion.io/yke/cmd"
	"yunion.io/yke/pkg/k8s"
	"yunion.io/yke/pkg/pki"
	"yunion.io/yke/pkg/tunnel"
	yketypes "yunion.io/yke/pkg/types"
	"yunion.io/yunioncloud/pkg/log"

	"yunion.io/yunion-kube/pkg/clusterdriver/types"
	"yunion.io/yunion-kube/pkg/clusterdriver/yke/ykecerts"
	"yunion.io/yunion-kube/pkg/types/slice"
	"yunion.io/yunion-kube/pkg/utils"
)

const (
	kubeConfigFile = "kube_config_cluster.yml"
	yunionPath     = "./management-state/yke/"
)

func NewDriver() types.Driver {
	return &Driver{}
}

type WrapTransportFactory func(config *yketypes.KubernetesEngineConfig) k8s.WrapTransport

type Driver struct {
	DockerDialer         tunnel.DialerFactory
	LocalDialer          tunnel.DialerFactory
	WrapTransportFactory WrapTransportFactory
}

func (d *Driver) wrapTransport(config *yketypes.KubernetesEngineConfig) k8s.WrapTransport {
	if d.WrapTransportFactory == nil {
		return nil
	}

	return k8s.WrapTransport(func(rt http.RoundTripper) http.RoundTripper {
		fn := d.WrapTransportFactory(config)
		if fn == nil {
			return rt
		}
		return fn(rt)
	})
}

func getYAML(driverOptions *types.DriverOptions) (string, error) {
	// first look up the file path then lookup raw ykeconfig
	if path, ok := driverOptions.StringOptions["config-file-path"]; ok {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return driverOptions.StringOptions["ykeConfig"], nil
}

func (d *Driver) Create(ctx context.Context, opts *types.DriverOptions, info *types.ClusterInfo) (*types.ClusterInfo, error) {
	yaml, err := getYAML(opts)
	if err != nil {
		return nil, err
	}

	ykeConfig, err := utils.ConvertToYkeConfig(yaml)
	if err != nil {
		return nil, err
	}

	stateDir, err := d.restore(info)
	if err != nil {
		return nil, err
	}
	defer d.cleanup(stateDir)

	log.Warningf("=== Start clusterup ykeconfig: %#v", ykeConfig)
	certsStr := ""
	apiURL, caCrt, clientCert, clientKey, certs, err := clusterUp(ctx, &ykeConfig, d.DockerDialer, d.LocalDialer,
		d.wrapTransport(&ykeConfig), false, stateDir, false, false)
	if err == nil {
		certsStr, err = ykecerts.ToString(certs)
	}
	if err != nil {
		return d.save(&types.ClusterInfo{
			Metadata: map[string]string{
				"Config": yaml,
			},
		}, stateDir), err
	}

	return d.save(&types.ClusterInfo{
		Metadata: map[string]string{
			"Endpoint":   apiURL,
			"RootCA":     base64.StdEncoding.EncodeToString([]byte(caCrt)),
			"ClientCert": base64.StdEncoding.EncodeToString([]byte(clientCert)),
			"ClientKey":  base64.StdEncoding.EncodeToString([]byte(clientKey)),
			"Config":     yaml,
			"Certs":      certsStr,
		},
	}, stateDir), nil
}

// Update the yke cluster
func (d *Driver) Update(ctx context.Context, opts *types.DriverOptions, clusterInfo *types.ClusterInfo) (*types.ClusterInfo, error) {
	yaml, err := getYAML(opts)
	if err != nil {
		return nil, err
	}

	ykeConfig, err := utils.ConvertToYkeConfig(yaml)
	if err != nil {
		return nil, err
	}

	stateDir, err := d.restore(clusterInfo)
	if err != nil {
		return nil, err
	}
	defer d.cleanup(stateDir)

	certStr := ""
	apiURL, caCrt, clientCert, clientKey, certs, err := cmd.ClusterUp(ctx, &ykeConfig, d.DockerDialer, d.LocalDialer,
		d.wrapTransport(&ykeConfig), false, stateDir, false, false)
	if err == nil {
		certStr, err = ykecerts.ToString(certs)
	}
	if err != nil {
		return d.save(&types.ClusterInfo{
			Metadata: map[string]string{
				"Config": yaml,
			},
		}, stateDir), err
	}

	return d.save(&types.ClusterInfo{
		Metadata: map[string]string{
			"Endpoint":   apiURL,
			"RootCA":     base64.StdEncoding.EncodeToString([]byte(caCrt)),
			"ClientCert": base64.StdEncoding.EncodeToString([]byte(clientCert)),
			"ClientKey":  base64.StdEncoding.EncodeToString([]byte(clientKey)),
			"Config":     yaml,
			"Certs":      certStr,
		},
	}, stateDir), nil
}

func (d *Driver) getClientset(info *types.ClusterInfo) (*kubernetes.Clientset, error) {
	yaml := info.Metadata["Config"]

	ykeConfig, err := utils.ConvertToYkeConfig(yaml)
	if err != nil {
		return nil, err
	}

	info.Endpoint = info.Metadata["Endpoint"]
	info.ClientCertificate = info.Metadata["ClientCert"]
	info.ClientKey = info.Metadata["ClientKey"]
	info.RootCaCertificate = info.Metadata["RootCA"]

	certBytes, err := base64.StdEncoding.DecodeString(info.ClientCertificate)
	if err != nil {
		return nil, err
	}
	keyBytes, err := base64.StdEncoding.DecodeString(info.ClientKey)
	if err != nil {
		return nil, err
	}
	rootBytes, err := base64.StdEncoding.DecodeString(info.RootCaCertificate)
	if err != nil {
		return nil, err
	}

	host := info.Endpoint
	if !strings.HasPrefix(host, "https://") {
		host = fmt.Sprintf("https://%s", host)
	}
	config := &rest.Config{
		Host: host,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   rootBytes,
			CertData: certBytes,
			KeyData:  keyBytes,
		},
		WrapTransport: d.WrapTransportFactory(&ykeConfig),
	}

	return kubernetes.NewForConfig(config)
}

// PostCheck does post action
func (d *Driver) PostCheck(ctx context.Context, info *types.ClusterInfo) (*types.ClusterInfo, error) {
	clientSet, err := d.getClientset(info)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for i := 0; i < 3; i++ {
		serverVersion, err := clientSet.DiscoveryClient.ServerVersion()
		if err != nil {
			lastErr = fmt.Errorf("failed to get Kubernetes server version: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		info.Version = serverVersion.GitVersion
		info.NodeCount, err = nodeCount(info)
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}

		return info, err
	}

	return nil, lastErr
}

func nodeCount(info *types.ClusterInfo) (int64, error) {
	yaml, ok := info.Metadata["Config"]
	if !ok {
		return 0, nil
	}

	ykeConfig, err := utils.ConvertToYkeConfig(yaml)
	if err != nil {
		return 0, err
	}

	count := int64(0)
	for _, node := range ykeConfig.Nodes {
		if slice.ContainsString(node.Role, "worker") {
			count++
		}
	}

	return count, nil
}

// Remove the cluster
func (d *Driver) Remove(ctx context.Context, clusterInfo *types.ClusterInfo) error {
	ykeConfig, err := utils.ConvertToYkeConfig(clusterInfo.Metadata["Config"])
	if err != nil {
		return err
	}
	stateDir, _ := d.restore(clusterInfo)
	defer d.save(nil, stateDir)
	return cmd.ClusterRemove(ctx, &ykeConfig, d.DockerDialer, d.wrapTransport(&ykeConfig), false, stateDir)
}

func (d *Driver) restore(info *types.ClusterInfo) (string, error) {
	os.MkdirAll(yunionPath, 0700)
	dir, err := ioutil.TempDir(yunionPath, "yke-")
	if err != nil {
		return "", err
	}

	if info != nil {
		state := info.Metadata["state"]
		if state != "" {
			ioutil.WriteFile(kubeConfig(dir), []byte(state), 0600)
		}
	}

	return filepath.Join(dir, "cluster.yml"), nil
}

func kubeConfig(stateDir string) string {
	if strings.HasSuffix(stateDir, "/cluster.yml") {
		return filepath.Join(filepath.Dir(stateDir), kubeConfigFile)
	}
	return filepath.Join(stateDir, kubeConfigFile)
}

func clusterUp(
	ctx context.Context,
	ykeConfig *yketypes.KubernetesEngineConfig,
	dockerDialerFactory, localConnDialerFactory tunnel.DialerFactory,
	k8sWrapTransport k8s.WrapTransport,
	local bool, configDir string, updateOnly, disablePortCheck bool) (string, string, string, string, map[string]pki.CertificatePKI, error) {
	apiURL, caCrt, clientCert, clientKey, certs, err := cmd.ClusterUp(ctx, ykeConfig, dockerDialerFactory, localConnDialerFactory, k8sWrapTransport, local, configDir, updateOnly, disablePortCheck)
	if err != nil {
		log.Warningf("cluster up error: %v", err)
	}
	return apiURL, caCrt, clientCert, clientKey, certs, err
}

func (d *Driver) cleanup(stateDir string) {
	if strings.HasSuffix(stateDir, "/cluster.yml") && !strings.Contains(stateDir, "..") {
		log.Infof("===cleanup statedir: %v", stateDir)
		os.Remove(stateDir)
		os.Remove(kubeConfig(stateDir))
		os.Remove(filepath.Dir(stateDir))
	}
}

func (d *Driver) save(info *types.ClusterInfo, stateDir string) *types.ClusterInfo {
	if info != nil {
		b, err := ioutil.ReadFile(kubeConfig(stateDir))
		if err == nil {
			if info.Metadata == nil {
				info.Metadata = map[string]string{}
			}
			info.Metadata["state"] = string(b)
		}
	}

	d.cleanup(stateDir)

	return info
}
