package cluster

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"golang.org/x/sync/errgroup"

	"yunion.io/yke/pkg/docker"
	"yunion.io/yke/pkg/hosts"
	"yunion.io/yke/pkg/templates"
	"yunion.io/yke/pkg/types"
	"yunion.io/yunioncloud/pkg/log"
)

const (
	NetworkPluginResourceName = "yke-network-plugin"

	PortCheckContainer        = "yke-port-checker"
	EtcdPortListenContainer   = "yke-etcd-port-listener"
	CPPortListenContainer     = "yke-cp-port-listener"
	WorkerPortListenContainer = "yke-worker-port-listener"

	KubeAPIPort    = "6443"
	EtcdPort1      = "2379"
	EtcdPort2      = "2380"
	ScedulerPort   = "10251"
	ControllerPort = "10252"
	KubeletPort    = "10250"
	KubeProxyPort  = "10256"

	ProtocolTCP = "TCP"
	ProtocolUDP = "UDP"

	// yunion specified
	YunionNetworkPlugin = "yunion"
	YunionCNIImage      = "yunion_cni_image"
	YunionBridge        = "yunion_bridge"
	YunionAuthURL       = "yunion_auth_url"
	YunionAdminUser     = "yunion_admin_user"
	YunionAdminPasswd   = "yunion_admin_passwd"
	YunionAdminProject  = "yunion_admin_project"
	YunionRegion        = "yunion_region"

	// List of map keys to be used with network templates

	// EtcdEndpoints is the server address for Etcd, used by calico
	EtcdEndpoints = "EtcdEndpoints"
	// APIRoot is the kubernetes API address
	APIRoot = "APIRoot"
	// kubernetes client certificates and kubeconfig paths

	EtcdClientCert     = "EtcdClientCert"
	EtcdClientKey      = "EtcdClientKey"
	EtcdClientCA       = "EtcdClientCA"
	EtcdClientCertPath = "EtcdClientCertPath"
	EtcdClientKeyPath  = "EtcdClientKeyPath"
	EtcdClientCAPath   = "EtcdClientCAPath"

	ClientCertPath = "ClientCertPath"
	ClientKeyPath  = "ClientKeyPath"
	ClientCAPath   = "ClientCAPath"

	KubeCfg = "KubeCfg"

	ClusterCIDR = "ClusterCIDR"
	// Images key names

	Image            = "Image"
	CNIImage         = "CNIImage"
	NodeImage        = "NodeImage"
	ControllersImage = "ControllersImage"

	RBACConfig = "RBACConfig"
)

var EtcdPortList = []string{
	EtcdPort1,
	EtcdPort2,
}

var ControlPlanePortList = []string{
	KubeAPIPort,
}

var WorkerPortList = []string{
	KubeletPort,
}

func (c *Cluster) deployNetworkPlugin(ctx context.Context) error {
	log.Infof("[network] Setting up network plugin: %s", c.Network.Plugin)
	switch c.Network.Plugin {
	case YunionNetworkPlugin:
		return c.doYunionDeploy(ctx)
	default:
		return fmt.Errorf("[network] Unsupported network plugin: %s", c.Network.Plugin)
	}
}

func (c *Cluster) doYunionDeploy(ctx context.Context) error {
	yunionConfig := map[string]string{
		ClusterCIDR:          c.ClusterCIDR,
		RBACConfig:           c.Authorization.Mode,
		CNIImage:             c.SystemImages.YunionCNI,
		"YunionBridge":       c.Network.Options[YunionBridge],
		"YunionAuthURL":      c.Network.Options[YunionAuthURL],
		"YunionAdminUser":    c.Network.Options[YunionAdminUser],
		"YunionAdminPasswd":  c.Network.Options[YunionAdminPasswd],
		"YunionAdminProject": c.Network.Options[YunionAdminProject],
		"YunionRegion":       c.Network.Options[YunionRegion],
	}
	pluginYaml, err := c.getNetworkPluginManifest(yunionConfig)
	if err != nil {
		return err
	}
	err = c.doAddonDeploy(ctx, pluginYaml, NetworkPluginResourceName)
	if err != nil {
		return fmt.Errorf("Deploy yunion cni container: %v", err)
	}
	return nil
}

func (c *Cluster) getNetworkPluginManifest(pluginConfig map[string]string) (string, error) {
	switch c.Network.Plugin {
	case YunionNetworkPlugin:
		return templates.CompileTemplateFromMap(templates.YunionCNITemplate, pluginConfig)
	default:
		return "", fmt.Errorf("[network] Unsupported network plugin: %s", c.Network.Plugin)
	}
}

func (c *Cluster) CheckClusterPorts(ctx context.Context, currentCluster *Cluster) error {
	if currentCluster != nil {
		newEtcdHost := hosts.GetToAddHosts(currentCluster.EtcdHosts, c.EtcdHosts)
		newControlPlanHosts := hosts.GetToAddHosts(currentCluster.ControlPlaneHosts, c.ControlPlaneHosts)
		newWorkerHosts := hosts.GetToAddHosts(currentCluster.WorkerHosts, c.WorkerHosts)

		if len(newEtcdHost) == 0 &&
			len(newWorkerHosts) == 0 &&
			len(newControlPlanHosts) == 0 {
			log.Infof("[network] No hosts added existing cluster, skipping port check")
			return nil
		}
	}
	if err := c.deployTCPPortListeners(ctx, currentCluster); err != nil {
		return err
	}
	if err := c.runServicePortChecks(ctx); err != nil {
		return err
	}
	if c.K8sWrapTransport == nil {
		if err := c.checkKubeAPIPort(ctx); err != nil {
			return err
		}
	}

	return c.removeTCPPortListeners(ctx)
}

func (c *Cluster) checkKubeAPIPort(ctx context.Context) error {
	log.Infof("[network] Checking KubeAPI port Control Plane hosts")
	for _, host := range c.ControlPlaneHosts {
		log.Debugf("[network] Checking KubeAPI port [%s] on host: %s", KubeAPIPort, host.Address)
		address := fmt.Sprintf("%s:%s", host.Address, KubeAPIPort)
		conn, err := net.Dial("tcp", address)
		if err != nil {
			return fmt.Errorf("[network] Can't access KubeAPI port [%s] on Control Plane host: %s", KubeAPIPort, host.Address)
		}
		conn.Close()
	}
	return nil
}

func (c *Cluster) deployTCPPortListeners(ctx context.Context, currentCluster *Cluster) error {
	log.Infof("[network] Deploying port listener containers")

	etcdHosts := []*hosts.Host{}
	cpHosts := []*hosts.Host{}
	workerHosts := []*hosts.Host{}
	if currentCluster != nil {
		etcdHosts = hosts.GetToAddHosts(currentCluster.EtcdHosts, c.EtcdHosts)
		cpHosts = hosts.GetToAddHosts(currentCluster.ControlPlaneHosts, c.ControlPlaneHosts)
		workerHosts = hosts.GetToAddHosts(currentCluster.WorkerHosts, c.WorkerHosts)
	} else {
		etcdHosts = c.EtcdHosts
		cpHosts = c.ControlPlaneHosts
		workerHosts = c.WorkerHosts
	}
	// deploy ectd listeners
	if err := c.deployListenerOnPlane(ctx, EtcdPortList, etcdHosts, EtcdPortListenContainer); err != nil {
		return err
	}

	// deploy controlplane listeners
	if err := c.deployListenerOnPlane(ctx, ControlPlanePortList, cpHosts, CPPortListenContainer); err != nil {
		return err
	}

	// deploy worker listeners
	if err := c.deployListenerOnPlane(ctx, WorkerPortList, workerHosts, WorkerPortListenContainer); err != nil {
		return err
	}
	log.Infof("[network] Port listener containers deployed successfully")
	return nil
}

func (c *Cluster) deployListenerOnPlane(ctx context.Context, portList []string, holstPlane []*hosts.Host, containerName string) error {
	var errgrp errgroup.Group
	for _, host := range holstPlane {
		runHost := host
		errgrp.Go(func() error {
			return c.deployListener(ctx, runHost, portList, containerName)
		})
	}
	return errgrp.Wait()
}

func (c *Cluster) deployListener(ctx context.Context, host *hosts.Host, portList []string, containerName string) error {
	imageCfg := &container.Config{
		Image: c.SystemImages.Alpine,
		Cmd: []string{
			"nc",
			"-kl",
			"-p",
			"1337",
			"-e",
			"echo",
		},
		ExposedPorts: nat.PortSet{
			"1337/tcp": {},
		},
	}
	hostCfg := &container.HostConfig{
		PortBindings: nat.PortMap{
			"1337/tcp": getPortBindings("0.0.0.0", portList),
		},
	}

	log.Debugf("[network] Starting deployListener [%s] on host [%s]", containerName, host.Address)
	if err := docker.DoRunContainer(ctx, host.DClient, imageCfg, hostCfg, containerName, host.Address, "network", c.PrivateRegistriesMap); err != nil {
		if strings.Contains(err.Error(), "bind: address already in use") {
			log.Debugf("[network] Service is already up on host [%s]", host.Address)
			return nil
		}
		return err
	}
	return nil
}

func (c *Cluster) removeTCPPortListeners(ctx context.Context) error {
	log.Infof("[network] Removing port listener containers")

	if err := removeListenerFromPlane(ctx, c.EtcdHosts, EtcdPortListenContainer); err != nil {
		return err
	}
	if err := removeListenerFromPlane(ctx, c.ControlPlaneHosts, CPPortListenContainer); err != nil {
		return err
	}
	if err := removeListenerFromPlane(ctx, c.WorkerHosts, WorkerPortListenContainer); err != nil {
		return err
	}
	log.Infof("[network] Port listener containers removed successfully")
	return nil
}

func removeListenerFromPlane(ctx context.Context, hostPlane []*hosts.Host, containerName string) error {
	var errgrp errgroup.Group
	for _, host := range hostPlane {
		runHost := host
		errgrp.Go(func() error {
			return docker.DoRemoveContainer(ctx, runHost.DClient, containerName, runHost.Address)
		})
	}
	return errgrp.Wait()
}

func (c *Cluster) runServicePortChecks(ctx context.Context) error {
	var errgrp errgroup.Group
	// check etcd <-> etcd
	// one etcd host is a pass
	if len(c.EtcdHosts) > 1 {
		log.Infof("[network] Running etcd <-> etcd port checks")
		for _, host := range c.EtcdHosts {
			runHost := host
			errgrp.Go(func() error {
				return checkPlaneTCPPortsFromHost(ctx, runHost, EtcdPortList, c.EtcdHosts, c.SystemImages.Alpine, c.PrivateRegistriesMap)
			})
		}
		if err := errgrp.Wait(); err != nil {
			return err
		}
	}
	// check all -> etcd connectivity
	log.Infof("[network] Running control plane -> etcd port checks")
	for _, host := range c.ControlPlaneHosts {
		runHost := host
		errgrp.Go(func() error {
			return checkPlaneTCPPortsFromHost(ctx, runHost, EtcdPortList, c.EtcdHosts, c.SystemImages.Alpine, c.PrivateRegistriesMap)
		})
	}
	if err := errgrp.Wait(); err != nil {
		return err
	}
	// check controle plane -> Workers
	log.Infof("[network] Running control plane -> worker port checks")
	for _, host := range c.ControlPlaneHosts {
		runHost := host
		errgrp.Go(func() error {
			return checkPlaneTCPPortsFromHost(ctx, runHost, WorkerPortList, c.WorkerHosts, c.SystemImages.Alpine, c.PrivateRegistriesMap)
		})
	}
	if err := errgrp.Wait(); err != nil {
		return err
	}
	// check workers -> control plane
	log.Infof("[network] Running workers -> control plane port checks")
	for _, host := range c.WorkerHosts {
		runHost := host
		errgrp.Go(func() error {
			return checkPlaneTCPPortsFromHost(ctx, runHost, ControlPlanePortList, c.ControlPlaneHosts, c.SystemImages.Alpine, c.PrivateRegistriesMap)
		})
	}
	return errgrp.Wait()
}

func checkPlaneTCPPortsFromHost(ctx context.Context, host *hosts.Host, portList []string, planeHosts []*hosts.Host, image string, prsMap map[string]types.PrivateRegistry) error {
	hosts := []string{}
	for _, host := range planeHosts {
		hosts = append(hosts, host.InternalAddress)
	}
	imageCfg := &container.Config{
		Image: image,
		Tty:   true,
		Env: []string{
			fmt.Sprintf("HOSTS=%s", strings.Join(hosts, " ")),
			fmt.Sprintf("PORTS=%s", strings.Join(portList, " ")),
		},
		Cmd: []string{
			"sh",
			"-c",
			"for host in $HOSTS; do for port in $PORTS ; do nc -z $host $port > /dev/null || echo $host $port ; done; done",
		},
	}
	hostCfg := &container.HostConfig{
		NetworkMode: "host",
	}
	if err := docker.DoRemoveContainer(ctx, host.DClient, PortCheckContainer, host.Address); err != nil {
		return err
	}
	if err := docker.DoRunContainer(ctx, host.DClient, imageCfg, hostCfg, PortCheckContainer, host.Address, "network", prsMap); err != nil {
		return err
	}
	if err := docker.WaitForContainer(ctx, host.DClient, host.Address, PortCheckContainer); err != nil {
		return err
	}
	logs, err := docker.ReadContainerLogs(ctx, host.DClient, PortCheckContainer)
	if err != nil {
		return err
	}
	defer logs.Close()
	if err := docker.RemoveContainer(ctx, host.DClient, host.Address, PortCheckContainer); err != nil {
		return err
	}
	portCheckLogs, err := getPortCheckLogs(logs)
	if err != nil {
		return err
	}
	if len(portCheckLogs) > 0 {
		return fmt.Errorf("[network] Port check for ports: [%s] failed on host: [%s]", strings.Join(portCheckLogs, ", "), host.Address)
	}
	return nil
}

func getPortCheckLogs(reader io.ReadCloser) ([]string, error) {
	logLines := bufio.NewScanner(reader)
	hostPortLines := []string{}
	for logLines.Scan() {
		logLine := strings.Split(logLines.Text(), " ")
		hostPortLines = append(hostPortLines, fmt.Sprintf("%s:%s", logLine[0], logLine[1]))
	}
	if err := logLines.Err(); err != nil {
		return nil, err
	}
	return hostPortLines, nil
}

func getPortBindings(hostAddress string, portList []string) []nat.PortBinding {
	portBindingList := []nat.PortBinding{}
	for _, portNumber := range portList {
		rawPort := fmt.Sprintf("%s:%s:1337/tcp", hostAddress, portNumber)
		portMapping, _ := nat.ParsePortSpec(rawPort)
		portBindingList = append(portBindingList, portMapping[0].Binding)
	}
	return portBindingList
}