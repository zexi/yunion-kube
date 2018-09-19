package sync

import (
	"fmt"
	"time"

	api "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"

	"yunion.io/x/yunion-kube/pkg/models"
)

var NamespaceAll = api.NamespaceAll

type SyncOptions struct {
	ResyncPeriod time.Duration
	Selector     labels.Selector
}

type SyncController struct {
	client   *kubernetes.Clientset
	selector labels.Selector

	podController cache.Controller
	podLister     cache.Indexer
	stopCh        chan struct{}
}

func NewSyncController(k8sCli *kubernetes.Clientset, opts SyncOptions) *SyncController {
	c := &SyncController{
		client:   k8sCli,
		selector: opts.Selector,
		stopCh:   make(chan struct{}),
	}

	c.podLister, c.podController = cache.NewIndexerInformer(
		&cache.ListWatch{
			ListFunc:  podListFunc(c.client, NamespaceAll, c.selector),
			WatchFunc: podWatchFunc(c.client, NamespaceAll, c.selector),
		},
		&api.Pod{},
		opts.ResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.podAdd,
			UpdateFunc: c.podUpdate,
			DeleteFunc: c.podDelete,
		},
		cache.Indexers{},
	)
	return c
}

func podListFunc(c *kubernetes.Clientset, ns string, s labels.Selector) func(meta.ListOptions) (runtime.Object, error) {
	return func(opts meta.ListOptions) (runtime.Object, error) {
		if s != nil {
			opts.LabelSelector = s.String()
		}
		pods, err := c.Core().Pods(ns).List(opts)
		if err != nil {
			return nil, err
		}
		return pods, err
	}
}

func podWatchFunc(c *kubernetes.Clientset, ns string, s labels.Selector) func(meta.ListOptions) (watch.Interface, error) {
	return func(opts meta.ListOptions) (watch.Interface, error) {
		if s != nil {
			opts.LabelSelector = s.String()
		}
		w, err := c.CoreV1().Pods(ns).Watch(opts)
		if err != nil {
			return nil, err
		}
		return w, err
	}
}

func (c *SyncController) podAdd(obj interface{}) {
	c.sendPodUpdates(nil, obj.(*api.Pod))
}

func (c *SyncController) podUpdate(oldObj, newObj interface{}) {
	c.sendPodUpdates(oldObj.(*api.Pod), newObj.(*api.Pod))
}

func (c *SyncController) podDelete(obj interface{}) {
	pod := obj.(*api.Pod)
	log.Infof("Pod %q was deleted, namespace: %q", pod.GetName(), pod.GetNamespace())
	//c.sendPodUpdates(obj.(*api.Pod), nil)
}

func (c *SyncController) sendPodUpdates(oldPod, newPod *api.Pod) {
	//log.Infof("sendPodUpdates, oldPod: %#v, newPod: %#v", oldPod, newPod)
	if oldPod != nil && newPod != nil && (oldPod.GetResourceVersion() == newPod.GetResourceVersion()) {
		log.Debugf("pod %s/%s metadata not change, skip update", oldPod.GetNamespace(), oldPod.GetName())
		return
	}
	pod := newPod
	if pod == nil {
		pod = oldPod
	}
	//c.ensurePodLimitRange(pod)
	err := c.updateCloudGuest(pod)
	if err != nil {
		log.Errorf("Update cloud guest error: %v", err)
	}
}

// Run starts the controller
func (c *SyncController) Run() {
	if c.podController != nil {
		go c.podController.Run(c.stopCh)
	}
	<-c.stopCh
}

func (c *SyncController) Stop() {
	close(c.stopCh)
}

func (c *SyncController) ensurePodLimitRange(pod *api.Pod) error {
	//_, err := c.client.CoreV1().Pods(pod.Namespace).Update(&api.Pod{
	//LimitRange:
	//})
	//if err != nil {
	//return err
	//}
	return nil
}

type Resource struct {
	MilliCPU int64
	Memory   int64
	//EphemeralStorage int64
}

func (r Resource) ToGuestUpdateParams() *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	//params.Add(jsonutils.NewInt(), "vcpu_count")
	params.Add(jsonutils.NewString(fmt.Sprintf("%dM", r.Memory/1024/1024)), "vmem_size")
	return params
}

const (
	DefaultMilliCPURequest int64 = 100              // 0.1 core
	DefaultMemoryRequest   int64 = 64 * 1024 * 1024 // 64 MB
)

// GetNonZeroRequests returns the default resource request if none is found or what is provided on the request
func GetNonZeroRequests(requests *api.ResourceList) (int64, int64) {
	var outMilliCPU, outMemory int64
	if _, found := (*requests)[api.ResourceCPU]; !found {
		outMilliCPU = DefaultMilliCPURequest
	} else {
		outMilliCPU = requests.Cpu().MilliValue()
	}
	if _, found := (*requests)[api.ResourceMemory]; !found {
		outMemory = DefaultMemoryRequest
	} else {
		outMemory = requests.Memory().Value()
	}
	return outMilliCPU, outMemory
}

func GetPodNonZeroRequests(pod *api.Pod) *Resource {
	result := &Resource{}
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		cpu, memory := GetNonZeroRequests(&container.Resources.Requests)
		result.MilliCPU += cpu
		result.Memory += memory
	}
	return result
}

func (c *SyncController) updateCloudGuest(pod *api.Pod) error {
	resource := GetPodNonZeroRequests(pod)
	session, err := models.GetAdminSession()
	if err != nil {
		return err
	}
	obj, err := cloudmod.Servers.Update(session, pod.Name, resource.ToGuestUpdateParams())
	if err != nil {
		log.Errorf("Update guest %s/%s error: %v", pod.Namespace, pod.Name, err)
		return err
	}
	log.Debugf("Update guest: %s", obj)
	return nil
}
