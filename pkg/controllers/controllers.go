package controllers

import (
	"fmt"
	"sync"

	"k8s.io/client-go/kubernetes"
	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/controllers/auth"
	"yunion.io/x/yunion-kube/pkg/controllers/helm"
	"yunion.io/x/yunion-kube/pkg/models"
)

var Manager *SControllerManager

func init() {
	Manager = newControllerManager()
}

func Start() {
	clusters, err := models.ClusterManager.GetInternalClusters()
	if err != nil {
		log.Fatalf("Get internal clusters: %v", err)
	}
	for _, cluster := range clusters {
		err = Manager.AddController(&cluster)
		if err != nil {
			log.Errorf("Add cluster %q to manager error: %v", cluster.Name, err)
		}
	}
	helm.Start()
}

type SControllerManager struct {
	controllerLock *sync.Mutex
	controllerMap  map[string]*SClusterController
}

func newControllerManager() *SControllerManager {
	return &SControllerManager{
		controllerLock: new(sync.Mutex),
		controllerMap:  make(map[string]*SClusterController),
	}
}

func (m *SControllerManager) GetController(clusterId string) (*SClusterController, error) {
	m.controllerLock.Lock()
	defer m.controllerLock.Unlock()
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

	m.controllerLock.Lock()
	defer m.controllerLock.Unlock()

	controller, err := newClusterController(cluster)
	if err != nil {
		err := fmt.Errorf("Add cluster %q controller error: %v", cluster.Name, err)
		return err
	}
	m.controllerMap[cluster.Id] = controller
	return nil
}

type SClusterController struct {
	clusterId             string
	clusterName           string
	keystoneAuthenticator *auth.KeystoneAuthenticator
}

func newClusterController(cluster *models.SCluster) (*SClusterController, error) {
	ctrl := &SClusterController{
		clusterId:   cluster.Id,
		clusterName: cluster.Name,
	}

	err := ctrl.RunKeystoneAuthenticator()
	if err != nil {
		return nil, err
	}

	return ctrl, nil
}

func (c *SClusterController) RunKeystoneAuthenticator() error {
	k8sCli, err := c.K8sClient()
	if err != nil {
		return err
	}
	kauth, err := auth.NewKeystoneAuthenticator(k8sCli)
	if err != nil {
		return err
	}
	c.keystoneAuthenticator = kauth
	return nil
}

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
