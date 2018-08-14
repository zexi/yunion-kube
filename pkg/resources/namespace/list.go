package namespace

import (
	"k8s.io/api/core/v1"
	client "k8s.io/client-go/kubernetes"
	"yunion.io/x/jsonutils"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type Namespace struct {
	api.ObjectMeta
	api.TypeMeta

	Phase v1.NamespacePhase
}

type NamespaceList struct {
	*dataselect.ListMeta
	namespaces []Namespace
}

func (n Namespace) ToListItem() jsonutils.JSONObject {
	return jsonutils.Marshal(n)
}

func (man *SNamespaceManager) AllowListItems(req *common.Request) bool {
	return req.UserCred.IsSystemAdmin()
}

func (man *SNamespaceManager) List(client client.Interface, req *common.Request) (common.ListResource, error) {
	return man.GetNamespaceList(client, req.ToQuery())
}

func (man *SNamespaceManager) GetNamespaceList(client client.Interface, dsQuery *dataselect.DataSelectQuery) (*NamespaceList, error) {
	channels := &common.ResourceChannels{
		NamespaceList: common.GetNamespaceListChannel(client),
	}
	return GetNamespaceListFromChannels(channels, dsQuery)
}

func (l *NamespaceList) Append(obj interface{}) {
	l.namespaces = append(l.namespaces, ToNamespace(obj.(v1.Namespace)))
}

func (l *NamespaceList) GetResponseData() interface{} {
	return l.namespaces
}

func GetNamespaceListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery) (*NamespaceList, error) {
	namespaces := <-channels.NamespaceList.List
	err := <-channels.NamespaceList.Error
	if err != nil {
		return nil, err
	}
	namespaceList := &NamespaceList{
		ListMeta:   dataselect.NewListMeta(),
		namespaces: make([]Namespace, 0),
	}
	err = dataselect.ToResourceList(
		namespaceList,
		namespaces.Items,
		dataselect.NewNamespaceDataCell,
		dsQuery,
	)
	return namespaceList, err
}
