package controllers

import (
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"

	"yunion.io/x/yunion-kube/pkg/controllers/auth"
	"yunion.io/x/yunion-kube/pkg/controllers/helm"
	//lxcfscontroller "yunion.io/x/yunion-kube/pkg/controllers/lxcfs"
	synccontroller "yunion.io/x/yunion-kube/pkg/controllers/sync"
)

var Manager *SControllerManager

func init() {
	Manager = newControllerManager()
}

func Start() {
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

type SClusterController struct {
	clusterId             string
	clusterName           string
	keystoneAuthenticator *auth.KeystoneAuthenticator
	syncController        *synccontroller.SyncController
	//lxcfsController       *lxcfscontroller.LxcfsInitializeController
	stopCh chan struct{}
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
