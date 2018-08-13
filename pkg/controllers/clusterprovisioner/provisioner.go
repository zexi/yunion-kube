package clusterprovisioner

import (
	"fmt"
	"time"

	"k8s.io/client-go/util/flowcontrol"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/clusterdriver"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/types/apis"
	"yunion.io/x/yunion-kube/pkg/types/config"
	"yunion.io/x/yunion-kube/pkg/ykedialerfactory"
)

type Provisioner struct {
	ClusterManager *models.SClusterManager
	NodeManager    *models.SNodeManager
	backoff        *flowcontrol.Backoff
}

func Register(scaledCtx *config.ScaledContext) {
	local := &ykedialerfactory.YKEDialerFactory{
		Factory: scaledCtx.Dialer,
	}
	docker := &ykedialerfactory.YKEDialerFactory{
		Factory: scaledCtx.Dialer,
		Docker:  true,
	}
	driver := clusterdriver.Drivers["yke"]
	ykeDriver := driver.(*yke.Driver)
	ykeDriver.DockerDialer = docker.Build
	ykeDriver.LocalDialer = local.Build
	ykeDriver.WrapTransportFactory = docker.WrapTransport
}

func (p *Provisioner) Remove(cluster *apis.Cluster) (*apis.Cluster, error) {
	log.Infof("Deleting cluster [%s]", cluster.Name)
	for i := 0; i < 5; i++ {
		err := p.driverRemove(cluster)
		if err == nil {
			break
		}
		if i == 3 {
			return cluster, fmt.Errorf("failed to remove the cluster[%s]: %v", cluster.Name, err)
		}
		time.Sleep(1 * time.Second)
	}
	log.Infof("Deleted cluster [%s]", cluster.Name)
	return nil, nil
}

func (p *Provisioner) Create(cluster *apis.Cluster) (*apis.Cluster, error) {
	var err error
	cls := p.ClusterManager.FetchCluster(cluster.Name)
	if cls == nil {
		return nil, fmt.Errorf("Cluster %q not found", cluster.Name)
	}
}
