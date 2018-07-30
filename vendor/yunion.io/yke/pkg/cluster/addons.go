package cluster

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"yunion.io/yke/pkg/addons"
	"yunion.io/yke/pkg/k8s"

	"yunion.io/yunioncloud/pkg/log"
)

const (
	KubeDNSAddonResourceName      = "yke-kubedns-addon"
	UserAddonResourceName         = "yke-user-addon"
	IngressAddonResourceName      = "yke-ingress-controller"
	UserAddonsIncludeResourceName = "yke-user-includes-addons"
)

type ingressOptions struct {
	RBACConfig     string
	Options        map[string]string
	NodeSelector   map[string]string
	AlpineImage    string
	IngressImage   string
	IngressBackend string
}

func (c *Cluster) deployK8sAddOns(ctx context.Context) error {
	if err := c.deployKubeDNS(ctx); err != nil {
		return err
	}
	return c.deployIngress(ctx)
}

func (c *Cluster) deployUserAddOns(ctx context.Context) error {
	log.Infof("[addons] Setting up user addons")
	if c.Addons != "" {
		if err := c.doAddonDeploy(ctx, c.Addons, UserAddonResourceName); err != nil {
			return err
		}
	}
	if len(c.AddonsInclude) > 0 {
		if err := c.deployAddonsInclude(ctx); err != nil {
			return err
		}
	}
	if c.Addons == "" && len(c.AddonsInclude) == 0 {
		log.Infof("[addons] no user addons defined")
	} else {
		log.Infof("[addons] User addons deployed successfully")
	}
	return nil
}

func (c *Cluster) deployAddonsInclude(ctx context.Context) error {
	var manifests []byte
	log.Infof("[addons] Checking for included user addons")

	if len(c.AddonsInclude) == 0 {
		log.Infof("[addons] No included addon paths or urls..")
		return nil
	}
	for _, addon := range c.AddonsInclude {
		if strings.HasPrefix(addon, "http") {
			addonYAML, err := getAddonFromURL(addon)
			if err != nil {
				return err
			}
			log.Infof("[addons] Adding addon from url %s", addon)
			log.Debugf("URL Yaml: %s", addonYAML)

			if err := validateUserAddonYAML(addonYAML); err != nil {
				return err
			}
			manifests = append(manifests, addonYAML...)
		} else if isFilePath(addon) {
			addonYAML, err := ioutil.ReadFile(addon)
			if err != nil {
				return err
			}
			log.Infof("[addons] Adding addon from %s", addon)
			log.Debugf("FilePath Yaml: %s", string(addonYAML))

			// make sure we properly separated manifests
			addonYAMLStr := string(addonYAML)
			if !strings.HasPrefix(addonYAMLStr, "---") {
				addonYAML = []byte(fmt.Sprintf("%s\n%s", "---", addonYAMLStr))
			}
			if err := validateUserAddonYAML(addonYAML); err != nil {
				return err
			}
			manifests = append(manifests, addonYAML...)
		} else {
			log.Warningf("[addons] Unable to determine if %s is a file path or url, skipping", addon)
		}
	}
	log.Infof("[addons] Deploying %s", UserAddonsIncludeResourceName)
	log.Debugf("[addons] Compiled addons yaml: %s", string(manifests))

	return c.doAddonDeploy(ctx, string(manifests), UserAddonsIncludeResourceName)
}

func validateUserAddonYAML(addon []byte) error {
	yamlContents := make(map[string]interface{})

	return yaml.Unmarshal(addon, &yamlContents)
}

func isFilePath(addonPath string) bool {
	if _, err := os.Stat(addonPath); os.IsNotExist(err) {
		return false
	}
	return true
}

func getAddonFromURL(yamlURL string) ([]byte, error) {
	resp, err := http.Get(yamlURL)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	addonYaml, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return addonYaml, nil

}

func (c *Cluster) deployKubeDNS(ctx context.Context) error {
	if disableDns := ctx.Value("disable-kube-dns"); disableDns != nil && disableDns.(bool) {
		log.Infof("[KubeDNS] disable-kube-dns is specified, skipping deploy it")
		return nil
	}
	log.Infof("[addons] Setting up KubeDNS")
	kubeDNSConfig := map[string]string{
		addons.KubeDNSServer:          c.ClusterDNSServer,
		addons.KubeDNSClusterDomain:   c.ClusterDomain,
		addons.KubeDNSImage:           c.SystemImages.KubeDNS,
		addons.DNSMasqImage:           c.SystemImages.DNSmasq,
		addons.KubeDNSSidecarImage:    c.SystemImages.KubeDNSSidecar,
		addons.KubeDNSAutoScalerImage: c.SystemImages.KubeDNSAutoscaler,
	}
	kubeDNSYaml, err := addons.GetKubeDNSManifest(kubeDNSConfig)
	if err != nil {
		return err
	}
	if err := c.doAddonDeploy(ctx, kubeDNSYaml, KubeDNSAddonResourceName); err != nil {
		return err
	}
	log.Infof("[addons] KubeDNS deployed successfully..")
	return nil
}

func (c *Cluster) deployWithKubectl(ctx context.Context, addonYaml string) error {
	buf := bytes.NewBufferString(addonYaml)
	cmd := exec.Command("kubectl", "--kubeconfig", c.LocalKubeConfigPath, "apply", "-f", "-")
	cmd.Stdin = buf
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *Cluster) doAddonDeploy(ctx context.Context, addonYaml, resourceName string) error {
	if c.UseKubectlDeploy {
		return c.deployWithKubectl(ctx, addonYaml)
	}

	err := c.StoreAddonConfigMap(ctx, addonYaml, resourceName)
	if err != nil {
		return fmt.Errorf("Failed to save addon ConfigMap: %v", err)
	}

	log.Infof("[addons] Executing deploy job [%s] ...", resourceName)
	k8sClient, err := k8s.NewClient(c.LocalKubeConfigPath, c.K8sWrapTransport)
	if err != nil {
		return err
	}
	node, err := k8s.GetNode(k8sClient, c.ControlPlaneHosts[0].HostnameOverride)
	if err != nil {
		return fmt.Errorf("Failed to get Node [%s]: %v", c.ControlPlaneHosts[0].HostnameOverride, err)
	}
	addonJob, err := addons.GetAddonsExcuteJob(resourceName, node.Name, c.Services.KubeAPI.Image)

	if err != nil {
		return fmt.Errorf("Failed to deploy addon execute job: %v", err)
	}
	err = c.ApplySystemAddonExcuteJob(addonJob)
	if err != nil {
		return fmt.Errorf("Failed to deploy addon execute job: %v", err)
	}
	return nil
}

func (c *Cluster) StoreAddonConfigMap(ctx context.Context, addonYaml string, addonName string) error {
	log.Infof("[addons] Saving addon ConfigMap to Kubernetes")
	kubeClient, err := k8s.NewClient(c.LocalKubeConfigPath, c.K8sWrapTransport)
	if err != nil {
		return err
	}
	timeout := make(chan bool, 1)
	go func() {
		for {
			err := k8s.UpdateConfigMap(kubeClient, []byte(addonYaml), addonName)
			if err != nil {
				time.Sleep(time.Second * 5)
				fmt.Println(err)
				continue
			}
			log.Infof("[addons] Successfully Saved addon to Kubernetes ConfigMap: %s", addonName)
			timeout <- true
			break
		}
	}()
	select {
	case <-timeout:
		return nil
	case <-time.After(time.Second * UpdateStateTimeout):
		return fmt.Errorf("[addons] Timeout waiting for kubernetes to be ready")
	}
}

func (c *Cluster) ApplySystemAddonExcuteJob(addonJob string) error {
	if err := k8s.ApplyK8sSystemJob(addonJob, c.LocalKubeConfigPath, c.K8sWrapTransport); err != nil {
		log.Errorf("%v", err)
		return err
	}
	return nil
}

func (c *Cluster) deployIngress(ctx context.Context) error {
	if disableIngress := ctx.Value("disable-ingress-controller"); disableIngress != nil && disableIngress.(bool) {
		log.Infof("[ingress] disable-ingress-controller is specified, skipping deploy")
		return nil
	}
	if c.Ingress.Provider == "none" {
		log.Infof("[ingress] ingress controller is not defined, skipping ingress controller")
		return nil
	}
	log.Infof("[ingress] Setting up %s ingress controller", c.Ingress.Provider)
	ingressConfig := ingressOptions{
		RBACConfig:     c.Authorization.Mode,
		Options:        c.Ingress.Options,
		NodeSelector:   c.Ingress.NodeSelector,
		AlpineImage:    c.SystemImages.Alpine,
		IngressImage:   c.SystemImages.Ingress,
		IngressBackend: c.SystemImages.IngressBackend,
	}
	// Currently only deploying nginx ingress controller
	ingressYaml, err := addons.GetNginxIngressManifest(ingressConfig)
	if err != nil {
		return err
	}
	if err := c.doAddonDeploy(ctx, ingressYaml, IngressAddonResourceName); err != nil {
		return err
	}
	log.Infof("[ingress] ingress controller %s is successfully deployed", c.Ingress.Provider)
	return nil
}
