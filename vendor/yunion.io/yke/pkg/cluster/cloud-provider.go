package cluster

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"

	"yunion.io/yke/pkg/docker"
	"yunion.io/yke/pkg/hosts"
	"yunion.io/yke/pkg/types"
	"yunion.io/yunioncloud/pkg/log"
)

const (
	CloudConfigDeployer    = "cloud-config-deployer"
	CloudConfigServiceName = "cloud"
	CloudConfigPath        = "/etc/kubernetes/cloud-config.json"
	CloudConfigEnv         = "YKE_CLOUD_CONFIG"
)

func deployCloudProviderConfig(ctx context.Context, uniqueHosts []*hosts.Host, alpineImage string, prsMap map[string]types.PrivateRegistry, cloudConfig string) error {
	for _, host := range uniqueHosts {
		log.Infof("[%s] Deploying cloud config file to node [%s]", CloudConfigServiceName, host.Address)
		if err := doDeployConfigFile(ctx, host, cloudConfig, alpineImage, prsMap); err != nil {
			return fmt.Errorf("Failed to deploy cloud config file on node [%s]: %v", host.Address, err)
		}
	}
	return nil
}

func doDeployConfigFile(ctx context.Context, host *hosts.Host, cloudConfig, alpineImage string, prsMap map[string]types.PrivateRegistry) error {
	// remove existing container. Only way it's still here is if previous deployment failed
	if err := docker.DoRemoveContainer(ctx, host.DClient, CloudConfigDeployer, host.Address); err != nil {
		return err
	}
	containerEnv := []string{CloudConfigEnv + "=" + cloudConfig}
	imageCfg := &container.Config{
		Image: alpineImage,
		Cmd: []string{
			"sh",
			"-c",
			fmt.Sprintf("if [ ! -f %s ]; then echo -e \"$%s\" > %s;fi", CloudConfigPath, CloudConfigEnv, CloudConfigPath),
		},
		Env: containerEnv,
	}
	hostCfg := &container.HostConfig{
		Binds: []string{
			"/etc/kubernetes:/etc/kubernetes",
		},
		Privileged: true,
	}
	if err := docker.DoRunContainer(ctx, host.DClient, imageCfg, hostCfg, CloudConfigDeployer, host.Address, CloudConfigServiceName, prsMap); err != nil {
		return err
	}
	if err := docker.DoRemoveContainer(ctx, host.DClient, CloudConfigDeployer, host.Address); err != nil {
		return err
	}
	log.Debugf("[%s] Successfully started cloud config deployer container on node [%s]", CloudConfigServiceName, host.Address)
	return nil
}
