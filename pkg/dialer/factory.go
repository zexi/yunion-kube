package dialer

import (
	"fmt"
	"net"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/yke/pkg/hosts"
	"yunion.io/x/yke/pkg/types"

	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/options"
	"yunion.io/x/yunion-kube/pkg/remotedialer"
	"yunion.io/x/yunion-kube/pkg/tunnelserver"
	"yunion.io/x/yunion-kube/pkg/types/config"
	"yunion.io/x/yunion-kube/pkg/types/config/dialer"
	onecloudcli "yunion.io/x/yunion-kube/pkg/utils/onecloud/client"
)

type Factory struct {
	Tunnelserver     *remotedialer.Server
	TunnelAuthorizer *tunnelserver.Authorizer
	ClusterManager   *models.SClusterManager
	NodeManager      *models.SNodeManager
}

func NewFactory(apiCtx *config.ScaledContext) (dialer.Factory, error) {
	authorizer := tunnelserver.NewAuthorizer(apiCtx)
	tunneler := tunnelserver.NewTunnelServer(apiCtx, authorizer)
	return &Factory{
		Tunnelserver:     tunneler,
		TunnelAuthorizer: authorizer,
		ClusterManager:   apiCtx.ClusterManager,
		NodeManager:      apiCtx.NodeManager,
	}, nil
}

func (f *Factory) sshDialer(node *models.SNode, privateKey string) (dialer.Dialer, error) {
	dial, err := hosts.SSHFactory(&hosts.Host{
		ConfigNode: types.ConfigNode{
			Address: node.Address,
			Port:    "22",
			User:    "root",
			SSHKey:  privateKey,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Get ssh Factory: %v", err)
	}
	return dial, nil
}

func (f *Factory) DockerDialer(clusterId, nodeId string) (dialer.Dialer, error) {
	node, err := f.NodeManager.FetchClusterNode(clusterId, nodeId)
	if err != nil {
		return nil, err
	}
	if options.Options.YkeUseSshDialer {
		log.Errorf("Use ssh docker dialer")
		session, err := models.GetAdminSession()
		if err != nil {
			return nil, fmt.Errorf("Get admin session: %v", err)
		}
		privateKey, err := onecloudcli.GetCloudSSHPrivateKey(session)
		if err != nil {
			return nil, fmt.Errorf("Get cloud ssh privateKey: %v", err)
		}
		return f.sshDialer(node, privateKey)
	}
	machineName := node.Name
	if f.Tunnelserver.HasSession(machineName) {
		d := f.Tunnelserver.Dialer(machineName, 15*time.Second)
		return func(string, string) (net.Conn, error) {
			return d("unix", "/var/run/docker.sock")
		}, nil
	}
	log.Errorf("Not found session: %s", machineName)

	return nil, fmt.Errorf("can not build docker dialer to %s:%s", clusterId, machineName)
}

func (f *Factory) NodeDialer(clusterId, nodeId string) (dialer.Dialer, error) {
	return func(network, address string) (net.Conn, error) {
		d, err := f.nodeDialer(clusterId, nodeId)
		if err != nil {
			return nil, err
		}
		return d(network, address)
	}, nil
}

func (f *Factory) nodeDialer(clusterId, nodeId string) (dialer.Dialer, error) {
	node, err := f.NodeManager.FetchClusterNode(clusterId, nodeId)
	if err != nil {
		return nil, err
	}
	machineName := node.Name

	if f.Tunnelserver.HasSession(machineName) {
		d := f.Tunnelserver.Dialer(machineName, 15*time.Second)
		return d, nil
	}

	return nil, fmt.Errorf("can not build node dialer to %s:%s", clusterId, machineName)
}
