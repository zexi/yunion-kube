package client

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/kube"

	"yunion.io/x/log"
)

type HelmTunnelClient struct {
	*helm.Client
	tillerTunnel *kube.Tunnel
	k8sClient    kubernetes.Interface
	k8sConfig    *rest.Config
}

func NewHelmTunnelClient(client kubernetes.Interface, config *rest.Config) (*HelmTunnelClient, error) {
	cli := &HelmTunnelClient{
		k8sClient: client,
		k8sConfig: config,
	}
	err := cli.tunnel()
	return cli, err
}

func (c *HelmTunnelClient) tunnel() error {
	log.Debugf("Create helm kubernetes tunnel...")
	tillerTunnel, err := portforwarder.New("kube-system", c.k8sClient, c.k8sConfig)
	if err != nil {
		return fmt.Errorf("create tunnel failed: %v", err)
	}
	tillerTunnelAddress := fmt.Sprintf("localhost:%d", tillerTunnel.Local)
	log.Debugf("Created kubernetes tunnel on address: %s", tillerTunnelAddress)
	helmClient := helm.NewClient(helm.Host(tillerTunnelAddress))
	c.Client = helmClient
	c.tillerTunnel = tillerTunnel
	return nil
}

func ensureTunnelClosed(tunnel *kube.Tunnel) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Ensure kubernetes tunnel closed: %v", err)
		}
	}()
	tunnel.Close()
}

// Close must be called when tunnel connected
func (c *HelmTunnelClient) Close() {
	ensureTunnelClosed(c.tillerTunnel)
}
