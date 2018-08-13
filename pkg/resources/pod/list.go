package pod

import (
	"yunion.io/x/log"

	"yunion.io/x/jsonutils"
	//"yunion.io/x/log"
	"k8s.io/api/core/v1"
	//metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/yunion-kube/pkg/resources"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type PodList struct {
	total  int
	limit  int
	offset int
	pods   []v1.Pod
}

func (l *PodList) Total() int {
	return l.total
}

func (l *PodList) Limit() int {
	return l.limit
}

func (l *PodList) Offset() int {
	return l.limit
}

func (l *PodList) Data() []jsonutils.JSONObject {
	ret := make([]jsonutils.JSONObject, len(l.pods))
	for i, item := range l.pods {
		ret[i] = jsonutils.Marshal(item)
	}
	return ret
}

func (man *SPodManager) AllowListItems(req *resources.Request) bool {
	return true
}

func (man *SPodManager) List(k8sCli kubernetes.Interface, req *resources.Request) (resources.ListResource, error) {
	log.Infof("Getting list of all pods in the cluster")
	return man.GetPodList(k8sCli, req)
}

func (man *SPodManager) GetPodList(k8sCli kubernetes.Interface, req *resources.Request) (*PodList, error) {
	//namespace := req.GetNamespace()
	list, err := k8sCli.CoreV1().Pods("").List(api.ListEverything)
	if err != nil {
		return nil, err
	}
	return &PodList{pods: list.Items, total: len(list.Items)}, nil
}
