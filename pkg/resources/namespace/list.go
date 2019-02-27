package namespace

import (
	"k8s.io/api/core/v1"
	client "k8s.io/client-go/kubernetes"
	"yunion.io/x/jsonutils"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// NamespaceList contains a list of namespaces in the cluster.
type NamespaceList struct {
	*common.BaseList

	// Unordered list of Namespaces.
	Namespaces []Namespace `json:"namespaces"`
}

func (l *NamespaceList) GetNamespaceListFromChannels() interface{} {
	return l.Namespaces
}

// Namespace is a presentation layer view of Kubernetes namespaces. This means it is namespace plus
// additional augmented data we can get from other sources.
type Namespace struct {
	api.ObjectMeta
	api.TypeMeta

	Phase v1.NamespacePhase `json:"status"`
}

func (n Namespace) ToListItem() jsonutils.JSONObject {
	return jsonutils.Marshal(n)
}

func (man *SNamespaceManager) List(req *common.Request) (common.ListResource, error) {
	return man.GetNamespaceList(req.GetK8sClient(), req.GetCluster(), req.ToQuery())
}

func (man *SNamespaceManager) GetNamespaceList(client client.Interface, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery) (*NamespaceList, error) {
	log.Infof("Getting list of all namespaces in the cluster")
	channels := &common.ResourceChannels{
		NamespaceList: common.GetNamespaceListChannel(client),
	}
	return GetNamespaceListFromChannels(channels, dsQuery, cluster)
}

func (l *NamespaceList) Append(obj interface{}) {
	l.Namespaces = append(l.Namespaces, toNamespace(obj.(v1.Namespace), l.GetCluster()))
}

func toNamespace(namespace v1.Namespace, cluster api.ICluster) Namespace {
	return Namespace{
		ObjectMeta: api.NewObjectMetaV2(namespace.ObjectMeta, cluster),
		TypeMeta:   api.NewTypeMeta(api.ResourceKindNamespace),
		Phase:      namespace.Status.Phase,
	}
}

func (l *NamespaceList) GetResponseData() interface{} {
	return l.Namespaces
}

func GetNamespaceListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*NamespaceList, error) {
	namespaces := <-channels.NamespaceList.List
	err := <-channels.NamespaceList.Error
	if err != nil {
		return nil, err
	}
	namespaceList := &NamespaceList{
		BaseList:   common.NewBaseList(cluster),
		Namespaces: make([]Namespace, 0),
	}
	err = dataselect.ToResourceList(
		namespaceList,
		namespaces.Items,
		dataselect.NewNamespaceDataCell,
		dsQuery,
	)
	return namespaceList, err
}
