package cluster

import (
	"yunion.io/x/yunion-kube/pkg/client"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/namespace"
	"yunion.io/x/yunion-kube/pkg/resources/node"
	pv "yunion.io/x/yunion-kube/pkg/resources/persistentvolume"
	"yunion.io/x/yunion-kube/pkg/resources/rbacroles"
	"yunion.io/x/yunion-kube/pkg/resources/storageclass"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
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
	return GetCluster(req.GetIndexer(), req.GetCluster(), dataselect.DefaultDataSelect())
}

// GetCluster returns a list of all cluster resources in the cluster.
func GetCluster(
	indexer *client.CacheFactory,
	cluster api.ICluster,
	dsQuery *dataselect.DataSelectQuery,
) (*Cluster, error) {
	log.Infof("Getting cluster category")
	channels := &common.ResourceChannels{
		NamespaceList:        common.GetNamespaceListChannel(indexer),
		NodeList:             common.GetNodeListChannel(indexer),
		PersistentVolumeList: common.GetPersistentVolumeListChannel(indexer),
		RoleList:             common.GetRoleListChannel(indexer),
		ClusterRoleList:      common.GetClusterRoleListChannel(indexer),
		StorageClassList:     common.GetStorageClassListChannel(indexer),
	}

	return GetClusterFromChannels(indexer, cluster, channels, dsQuery)
}

// GetClusterFromChannels returns a list of all cluster in the cluster, from the channel sources.
func GetClusterFromChannels(
	indexer *client.CacheFactory,
	cluster api.ICluster,
	channels *common.ResourceChannels,
	dsQuery *dataselect.DataSelectQuery,
) (*Cluster, error) {

	numErrs := 5
	errChan := make(chan error, numErrs)
	nsChan := make(chan *namespace.NamespaceList)
	nodeChan := make(chan *node.NodeList)
	pvChan := make(chan *pv.PersistentVolumeList)
	roleChan := make(chan *rbacroles.RbacRoleList)
	storageChan := make(chan *storageclass.StorageClassList)

	go func() {
		items, err := namespace.GetNamespaceListFromChannels(channels, dsQuery, cluster)
		errChan <- err
		nsChan <- items
	}()

	go func() {
		items, err := node.GetNodeListFromChannels(indexer, channels, dsQuery, cluster)
		errChan <- err
		nodeChan <- items
	}()

	go func() {
		items, err := pv.GetPersistentVolumeListFromChannels(channels, dsQuery, cluster)
		errChan <- err
		pvChan <- items
	}()

	go func() {
		items, err := rbacroles.GetRbacRoleListFromChannels(channels, dsQuery, cluster)
		errChan <- err
		roleChan <- items
	}()

	go func() {
		items, err := storageclass.GetStorageClassListFromChannels(channels, dsQuery, cluster)
		errChan <- err
		storageChan <- items
	}()

	for i := 0; i < numErrs; i++ {
		err := <-errChan
		if err != nil {
			return nil, err
		}
	}

	clusterObj := &Cluster{
		NamespaceList:        *(<-nsChan),
		NodeList:             *(<-nodeChan),
		PersistentVolumeList: *(<-pvChan),
		RoleList:             *(<-roleChan),
		StorageClassList:     *(<-storageChan),
	}

	return clusterObj, nil
}
