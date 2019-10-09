package controllers

import (
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"
	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/controllers/auth"
	"yunion.io/x/yunion-kube/pkg/controllers/helm"
	//lxcfscontroller "yunion.io/x/yunion-kube/pkg/controllers/lxcfs"
	synccontroller "yunion.io/x/yunion-kube/pkg/controllers/sync"
	"yunion.io/x/yunion-kube/pkg/models"
)

var Manager *SControllerManager

func init() {
	Manager = newControllerManager()
}

func Start() {
	clusters, err := models.ClusterManager.GetInternalClusters()
	if err != nil {
		log.Errorf("Get internal clusters: %v", err)
	}
	for _, cluster := range clusters {
		if cluster.ApiEndpoint == "" || cluster.Status == models.CLUSTER_STATUS_INIT {
			log.Warningf("Empty cluster %q, skip it", cluster.Name)
			continue
		}
		err = Manager.AddController(&cluster)
		if err != nil {
			log.Errorf("Add cluster %q to manager error: %v", cluster.Name, err)
		}
	}
	helm.Start()
}

type SControllerManager struct {
	controllerMap map[string]*SClusterController
}

func newControllerManager() *SControllerManager {
	return &SControllerManager{
		controllerMap: make(map[string]*SClusterController),
	}
}

func (m *SControllerManager) GetController(clusterId string) (*SClusterController, error) {
	ctrl, ok := m.controllerMap[clusterId]
	if !ok {
		return nil, fmt.Errorf("Cluster controller %q not found", clusterId)
	}
	return ctrl, nil
}

func (m *SControllerManager) AddController(cluster *models.SCluster) error {
	controller, _ := m.GetController(cluster.Id)
	if controller != nil {
		return nil
	}

	controller, err := newClusterController(cluster)
	if err != nil {
		err := fmt.Errorf("Add cluster %q controller error: %v", cluster.Name, err)
		return err
	}
	m.controllerMap[cluster.Id] = controller
	return nil
}

func (m *SControllerManager) RemoveController(cluster *models.SCluster) error {
	controller, err := m.GetController(cluster.Id)
	if err != nil {
		return fmt.Errorf("Not found controller by id: %q, cluster name: %q", cluster.Id, cluster.Name)
	}
	close(controller.stopCh)
	delete(m.controllerMap, cluster.Id)
	return nil
}

type SClusterController struct {
	clusterId             string
	clusterName           string
	keystoneAuthenticator *auth.KeystoneAuthenticator
	syncController        *synccontroller.SyncController
	//lxcfsController       *lxcfscontroller.LxcfsInitializeController
	stopCh chan struct{}
}

func newClusterController(cluster *models.SCluster) (*SClusterController, error) {
	ctrl := &SClusterController{
		clusterId:   cluster.Id,
		clusterName: cluster.Name,
		stopCh:      make(chan struct{}),
	}

	k8sCli, err := cluster.GetK8sClient()
	if err != nil {
		return nil, err
	}

	ctrl.RunKeystoneAuthenticator(k8sCli, ctrl.stopCh)

	controllerFuncs := []func(*kubernetes.Clientset, chan struct{}){
		ctrl.RunSyncController,
		//ctrl.RunLxcfsController,
	}

	for i := range controllerFuncs {
		idx := i
		go func() {
			controllerFuncs[idx](k8sCli, ctrl.stopCh)
		}()
	}

	return ctrl, nil
}

func (c *SClusterController) RunKeystoneAuthenticator(k8sCli *kubernetes.Clientset, stopCh chan struct{}) {
	c.keystoneAuthenticator = auth.NewKeystoneAuthenticator(k8sCli, stopCh)
}

func (c *SClusterController) RunSyncController(k8sCli *kubernetes.Clientset, stopCh chan struct{}) {
	c.syncController = synccontroller.NewSyncController(k8sCli, synccontroller.SyncOptions{
		ResyncPeriod: time.Duration(5 * time.Minute),
		StopCh:       stopCh,
	})
	c.syncController.Run()
}

//func (c *SClusterController) RunLxcfsController(k8sCli *kubernetes.Clientset, stopCh chan struct{}) {
//c.lxcfsController = lxcfscontroller.NewLxcfsInitializeController(k8sCli, stopCh)
//c.lxcfsController.Run()
//}

func (c *SClusterController) GetKeystoneAuthenticator() *auth.KeystoneAuthenticator {
	return c.keystoneAuthenticator
}

func (c *SClusterController) GetCluster() (*models.SCluster, error) {
	return models.ClusterManager.FetchClusterById(c.clusterId)
}

func (c *SClusterController) K8sClient() (*kubernetes.Clientset, error) {
	cluster, err := c.GetCluster()
	if err != nil {
		return nil, err
	}
	restConfig, err := cluster.GetK8sRestConfig()
	if err != nil {
		return nil, fmt.Errorf("Get cluster k8s rest config error: %v", err)
	}
	return kubernetes.NewForConfig(restConfig)
}
