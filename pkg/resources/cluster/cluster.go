package cluster

import (
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/namespace"
	"yunion.io/x/yunion-kube/pkg/resources/node"
	pv "yunion.io/x/yunion-kube/pkg/resources/persistentvolume"
	"yunion.io/x/yunion-kube/pkg/resources/rbacroles"
	"yunion.io/x/yunion-kube/pkg/resources/storageclass"
)

var ClusterManager *SClusterManager

type SClusterManager struct {
	*resources.SClusterResourceManager
}

func init() {
	ClusterManager = &SClusterManager{
		SClusterResourceManager: resources.NewClusterResourceManager("k8s_cluster", "k8s_clusters"),
	}
}

// Cluster structure contains all resource lists grouped into the cluster category.
type Cluster struct {
	NamespaceList        namespace.NamespaceList       `json:"namespaceList"`
	NodeList             node.NodeList                 `json:"nodeList"`
	PersistentVolumeList pv.PersistentVolumeList       `json:"persistentVolumeList"`
	RoleList             rbacroles.RbacRoleList        `json:"roleList"`
	StorageClassList     storageclass.StorageClassList `json:"storageClassList"`
}

func (man *SClusterManager) Get(req *common.Request, id string) (interface{}, error) {
	return GetCluster(req.GetK8sClient(), dataselect.DefaultDataSelect)
}

// GetCluster returns a list of all cluster resources in the cluster.
func GetCluster(client kubernetes.Interface, dsQuery *dataselect.DataSelectQuery) (*Cluster, error) {
	log.Infof("Getting cluster category")
	channels := &common.ResourceChannels{
		NamespaceList:        common.GetNamespaceListChannel(client),
		NodeList:             common.GetNodeListChannel(client),
		PersistentVolumeList: common.GetPersistentVolumeListChannel(client),
		RoleList:             common.GetRoleListChannel(client),
		ClusterRoleList:      common.GetClusterRoleListChannel(client),
		StorageClassList:     common.GetStorageClassListChannel(client),
	}

	return GetClusterFromChannels(client, channels, dsQuery)
}

// GetClusterFromChannels returns a list of all cluster in the cluster, from the channel sources.
func GetClusterFromChannels(client kubernetes.Interface, channels *common.ResourceChannels,
	dsQuery *dataselect.DataSelectQuery) (*Cluster, error) {

	numErrs := 5
	errChan := make(chan error, numErrs)
	nsChan := make(chan *namespace.NamespaceList)
	nodeChan := make(chan *node.NodeList)
	pvChan := make(chan *pv.PersistentVolumeList)
	roleChan := make(chan *rbacroles.RbacRoleList)
	storageChan := make(chan *storageclass.StorageClassList)

	go func() {
		items, err := namespace.GetNamespaceListFromChannels(channels, dsQuery)
		errChan <- err
		nsChan <- items
	}()

	go func() {
		items, err := node.GetNodeListFromChannels(client, channels, dsQuery)
		errChan <- err
		nodeChan <- items
	}()

	go func() {
		items, err := pv.GetPersistentVolumeListFromChannels(channels, dsQuery)
		errChan <- err
		pvChan <- items
	}()

	go func() {
		items, err := rbacroles.GetRbacRoleListFromChannels(channels, dsQuery)
		errChan <- err
		roleChan <- items
	}()

	go func() {
		items, err := storageclass.GetStorageClassListFromChannels(channels, dsQuery)
		errChan <- err
		storageChan <- items
	}()

	for i := 0; i < numErrs; i++ {
		err := <-errChan
		if err != nil {
			return nil, err
		}
	}

	cluster := &Cluster{
		NamespaceList:        *(<-nsChan),
		NodeList:             *(<-nodeChan),
		PersistentVolumeList: *(<-pvChan),
		RoleList:             *(<-roleChan),
		StorageClassList:     *(<-storageChan),
	}

	return cluster, nil
}
