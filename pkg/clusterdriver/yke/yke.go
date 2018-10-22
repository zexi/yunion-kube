package yke

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"yunion.io/x/log"
	"yunion.io/x/yke/cmd"
	"yunion.io/x/yke/pkg/hosts"
	"yunion.io/x/yke/pkg/k8s"
	"yunion.io/x/yke/pkg/pki"
	yketypes "yunion.io/x/yke/pkg/types"

	"yunion.io/x/yunion-kube/pkg/clusterdriver/types"
	"yunion.io/x/yunion-kube/pkg/clusterdriver/yke/ykecerts"
	"yunion.io/x/yunion-kube/pkg/types/slice"
	"yunion.io/x/yunion-kube/pkg/utils"
)

const (
	kubeConfigFile = "kube_config_cluster.yml"
	yunionPath     = "/tmp/management-state/yke/"
)

func NewDriver() types.Driver {
	return &Driver{}
}

type WrapTransportFactory func(config *yketypes.KubernetesEngineConfig) k8s.WrapTransport

type Driver struct {
	DockerDialer         hosts.DialerFactory
	LocalDialer          hosts.DialerFactory
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
	//defer d.cleanup(stateDir)

	log.Debugf("-------- create yke config: \n%s, stateDir: %q", yaml, stateDir)

	certsStr := ""
	apiURL, caCrt, clientCert, clientKey, certs, err := clusterUp(ctx, ykeConfig, d.DockerDialer, d.LocalDialer,
		d.wrapTransport(ykeConfig), false, stateDir, false, false)
	if err == nil {
		certsStr, err = ykecerts.ToString(certs)
	}
	if err != nil {
		return nil, err
	}

	return &types.ClusterInfo{
		Endpoint:          apiURL,
		RootCaCertificate: base64.StdEncoding.EncodeToString([]byte(caCrt)),
		ClientCertificate: base64.StdEncoding.EncodeToString([]byte(clientCert)),
		ClientKey:         base64.StdEncoding.EncodeToString([]byte(clientKey)),
		Config:            yaml,
		Certs:             certsStr,
	}, nil
}

// Update the yke cluster
func (d *Driver) Update(ctx context.Context, opts *types.DriverOptions, clusterInfo *types.ClusterInfo) (*types.ClusterInfo, error) {
	yaml, err := getYAML(opts)
	if err != nil {
		return nil, err
	}

	log.Debugf("-------update yke config: \n%s", yaml)

	ykeConfig, err := utils.ConvertToYkeConfig(yaml)
	if err != nil {
		return nil, err
	}

	stateDir, err := d.restore(clusterInfo)
	if err != nil {
		return nil, err
	}
	//defer d.cleanup(stateDir)

	certsStr := ""
	apiURL, caCrt, clientCert, clientKey, certs, err := clusterUp(ctx, ykeConfig, d.DockerDialer, d.LocalDialer,
		d.wrapTransport(ykeConfig), false, stateDir, false, false)
	if err == nil {
		certsStr, err = ykecerts.ToString(certs)
	}
	if err != nil {
		return nil, err
	}

	return &types.ClusterInfo{
		Endpoint:          apiURL,
		RootCaCertificate: base64.StdEncoding.EncodeToString([]byte(caCrt)),
		ClientCertificate: base64.StdEncoding.EncodeToString([]byte(clientCert)),
		ClientKey:         base64.StdEncoding.EncodeToString([]byte(clientKey)),
		Config:            yaml,
		Certs:             certsStr,
	}, nil
}

func (d *Driver) GetK8sRestConfig(info *types.ClusterInfo) (*rest.Config, error) {
	yaml := info.Config

	ykeConfig, err := utils.ConvertToYkeConfig(yaml)
	if err != nil {
		return nil, err
	}

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
		WrapTransport: d.WrapTransportFactory(ykeConfig),
	}
	return config, nil
}

func (d *Driver) getClientset(info *types.ClusterInfo) (*kubernetes.Clientset, error) {
	config, err := d.GetK8sRestConfig(info)
	if err != nil {
		return nil, err
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
	yaml := info.Config
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
	ykeConfig, err := utils.ConvertToYkeConfig(clusterInfo.Config)
	if err != nil {
		return err
	}
	stateDir, _ := d.restore(clusterInfo)
	//defer d.save(nil, stateDir)
	//defer d.cleanup(stateDir)
	return cmd.ClusterRemove(ctx, ykeConfig, d.DockerDialer, d.wrapTransport(ykeConfig), false, stateDir)
}

func getHost(driverOptions *types.DriverOptions) (*hosts.Host, error) {
	hostJsonStr, ok := driverOptions.StringOptions["host"]
	if !ok {
		return nil, fmt.Errorf("No host json string")
	}
	host := hosts.Host{}
	err := json.NewDecoder(strings.NewReader(hostJsonStr)).Decode(&host)
	if err != nil {
		return nil, err
	}
	return &host, nil
}

func (d *Driver) restore(info *types.ClusterInfo) (string, error) {
	os.MkdirAll(yunionPath, 0700)
	dir, err := ioutil.TempDir(yunionPath, "yke-")
	if err != nil {
		return "", err
	}

	if info != nil {
		state := info.KubeConfig
		if state != "" {
			log.Errorf("******** Write file: %q, content: %s", kubeConfig(dir), state)
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
	dockerDialerFactory, localConnDialerFactory hosts.DialerFactory,
	k8sWrapTransport k8s.WrapTransport,
	local bool, configDir string, updateOnly, disablePortCheck bool) (string, string, string, string, map[string]pki.CertificatePKI, error) {
	log.Errorf("=====clusterup configDir: %q", configDir)
	// enable lxcfs init plugin by default
	ctx = context.WithValue(ctx, "enable-lxcfs", true)
	apiURL, caCrt, clientCert, clientKey, certs, err := cmd.ClusterUp(ctx, ykeConfig, dockerDialerFactory, localConnDialerFactory, k8sWrapTransport, local, configDir, updateOnly, disablePortCheck)
	if err != nil {
		log.Warningf("cluster up error: %v", err)
	}
	return apiURL, caCrt, clientCert, clientKey, certs, err
}

func (d *Driver) cleanup(stateDir string) {
	if strings.HasSuffix(stateDir, "/cluster.yml") && !strings.Contains(stateDir, "..") {
		log.Infof("Cleanup statedir: %v", stateDir)
		os.Remove(stateDir)
		os.Remove(kubeConfig(stateDir))
		os.Remove(filepath.Dir(stateDir))
	}
}

//func (d *Driver) save(info *types.ClusterInfo, stateDir string) *types.ClusterInfo {
//if info != nil {
//b, err := ioutil.ReadFile(kubeConfig(stateDir))
//if err == nil {
//if info.Metadata == nil {
//info.Metadata = map[string]string{}
//}
//info.Metadata["state"] = string(b)
//}
//}

//d.cleanup(stateDir)

//return info
//}
