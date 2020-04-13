package clientv2

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/discovery"
	cacheddiscovery "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	_ "k8s.io/kubectl/pkg/scheme"

	"yunion.io/x/pkg/errors"
)

type Client struct {
	k8s *K8sClient
}

func NewClient(kubeconfig string) (*Client, error) {
	k8sCli, err := newK8sClient(kubeconfig)
	if err != nil {
		return nil, err
	}
	return &Client{
		k8s: k8sCli,
	}, nil
}

func (c *Client) K8S() *K8sClient {
	return c.k8s
}

var _ genericclioptions.RESTClientGetter = &k8sConfig{}

type k8sConfig struct {
	rawConfig  clientcmd.ClientConfig
	restConfig *rest.Config
	cliSet     *kubernetes.Clientset
}

func newK8sConfig(kubeconfig string) (*k8sConfig, error) {
	apiconfig, err := clientcmd.Load([]byte(kubeconfig))
	if err != nil {
		return nil, err
	}
	rawConfig := clientcmd.NewDefaultClientConfig(*apiconfig, &clientcmd.ConfigOverrides{})
	restConfig, err := rawConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	cliSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return &k8sConfig{
		rawConfig:  rawConfig,
		restConfig: restConfig,
		cliSet:     cliSet,
	}, nil
}

func (c *k8sConfig) ToRESTConfig() (*rest.Config, error) {
	// return c.ToRawKubeConfigLoader().ClientConfig()
	return c.restConfig, nil
}

func (c *k8sConfig) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return c.rawConfig
}

func (c *k8sConfig) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	config, err := c.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	// The more groups you have, the more discovery requests you need to make.
	// given 25 groups (our groups + a few custom resources) with one-ish version each, discovery needs to make 50 requests
	// double it just so we don't end up here again for a while.  This config is only used for discovery.
	config.Burst = 100
	cliSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	// cliSet := c.cliSet
	return cacheddiscovery.NewMemCacheClient(cliSet), nil
}

func (c *k8sConfig) ToRESTMapper() (meta.RESTMapper, error) {
	discoveryClient, err := c.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)
	expander := restmapper.NewShortcutExpander(mapper, discoveryClient)
	return expander, nil
}

type K8sClient struct {
	Factory cmdutil.Factory
}

func newK8sClient(kubeconfig string) (*K8sClient, error) {
	getter, err := newK8sConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	f := cmdutil.NewFactory(getter)

	return &K8sClient{
		Factory: f,
	}, nil
}

func (c *K8sClient) IsReachable() error {
	client, _ := c.Factory.KubernetesClientSet()
	_, err := client.ServerVersion()
	if err != nil {
		return errors.Wrap(err, "Kubernetes cluster unreachable")
	}
	return nil
}

func (c *K8sClient) newBuilder() *resource.Builder {
	return c.Factory.NewBuilder().Flatten()
}

func (c *K8sClient) Get(resourceType string, namespace string, name string) (runtime.Object, error) {
	r := c.newBuilder().Unstructured().
		NamespaceParam(namespace).
		ResourceNames(resourceType, name).
		Latest().
		Do()
	if err := r.Err(); err != nil {
		return nil, err
	}
	infos, err := r.Infos()
	if err != nil {
		return nil, err
	}
	return infos[0].Object, nil
}
