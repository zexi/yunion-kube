package namespace

import (
	"k8s.io/api/core/v1"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

// NamespaceList contains a list of namespaces in the cluster.
type NamespaceList struct {
	*common.BaseList

	// Unordered list of Namespaces.
	Namespaces []api.Namespace `json:"namespaces"`
}

func (l *NamespaceList) GetResponseData() interface{} {
	return l.Namespaces
}

func (l *NamespaceList) GetNamespaceListFromChannels() interface{} {
	return l.Namespaces
}

func (man *SNamespaceManager) List(req *common.Request) (common.ListResource, error) {
	return man.GetNamespaceList(req.GetIndexer(), req.GetCluster(), req.ToQuery())
}

func (man *SNamespaceManager) GetNamespaceList(
	indexer *client.CacheFactory,
	cluster api.ICluster,
	dsQuery *dataselect.DataSelectQuery,
) (*NamespaceList, error) {
	log.Infof("Getting list of all namespaces in the cluster")
	channels := &common.ResourceChannels{
		NamespaceList: common.GetNamespaceListChannel(indexer),
	}
	return GetNamespaceListFromChannels(channels, dsQuery, cluster)
}

func (l *NamespaceList) Append(obj interface{}) {
	l.Namespaces = append(l.Namespaces, toNamespace(obj.(*v1.Namespace), l.GetCluster()))
}

func toNamespace(namespace *v1.Namespace, cluster api.ICluster) api.Namespace {
	return api.Namespace{
		ObjectMeta: api.NewObjectMeta(namespace.ObjectMeta, cluster),
		TypeMeta:   api.NewTypeMeta(namespace.TypeMeta),
		Phase:      namespace.Status.Phase,
	}
}

func GetNamespaceListFromChannels(
	channels *common.ResourceChannels,
	dsQuery *dataselect.DataSelectQuery,
	cluster api.ICluster,
	//projectNamespaces *common.ProjectNamespaces,
) (*NamespaceList, error) {
	namespaces := <-channels.NamespaceList.List
	err := <-channels.NamespaceList.Error
	if err != nil {
		return nil, err
	}
	items := make([]*v1.Namespace, 0)
	allNs := namespaces
	//if !projectNamespaces.HasAllNamespacePrivelege() {
	//for _, ns := range allNs {
	//if projectNamespaces.Sets().Has(ns.GetName()) {
	//items = append(items, ns)
	//}
	//}
	//} else {
	//items = allNs
	//}
	items = allNs
	namespaceList := &NamespaceList{
		BaseList:   common.NewBaseList(cluster),
		Namespaces: make([]api.Namespace, 0),
	}
	err = dataselect.ToResourceList(
		namespaceList,
		items,
		dataselect.NewNamespaceDataCell,
		dsQuery,
	)
	return namespaceList, err
}
