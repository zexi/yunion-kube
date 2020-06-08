package k8smodels

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"

	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

func GetSelectorByObjectMeta(meta *metav1.ObjectMeta) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: meta.GetLabels(),
	}
}

func AddObjectMetaDefaultLabel(meta *metav1.ObjectMeta) *metav1.ObjectMeta {
	return AddObjectMetaRunLabel(meta)
}

func AddObjectMetaRunLabel(meta *metav1.ObjectMeta) *metav1.ObjectMeta {
	if len(meta.Labels) == 0 {
		meta.Labels["run"] = meta.GetName()
	}
	return meta
}

func GetServicePortsByMapping(ps []api.PortMapping) []v1.ServicePort {
	ports := []v1.ServicePort{}
	for _, p := range ps {
		ports = append(ports, p.ToServicePort())
	}
	return ports
}

func GetServiceFromOption(objMeta *metav1.ObjectMeta, opt *api.ServiceCreateOption) *v1.Service {
	if opt == nil {
		return nil
	}
	svcType := opt.Type
	if svcType == "" {
		svcType = string(v1.ServiceTypeClusterIP)
	}
	if opt.IsExternal {
		svcType = string(v1.ServiceTypeLoadBalancer)
	}
	selector := opt.Selector
	if len(selector) == 0 {
		selector = GetSelectorByObjectMeta(objMeta).MatchLabels
	}
	svc := &v1.Service{
		ObjectMeta: *objMeta,
		Spec: v1.ServiceSpec{
			Selector: selector,
			Type:     v1.ServiceType(svcType),
			Ports:    GetServicePortsByMapping(opt.PortMappings),
		},
	}
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}
	if opt.LoadBalancerNetwork != "" {
		svc.Annotations[api.YUNION_LB_NETWORK_ANNOTATION] = opt.LoadBalancerNetwork
	}
	if opt.LoadBalancerCluster != "" {
		svc.Annotations[api.YUNION_LB_CLUSTER_ANNOTATION] = opt.LoadBalancerCluster
	}
	return svc
}

func GetRawPodsByController(obj model.IK8SModel) ([]*v1.Pod, error) {
	pods, err := PodManager.GetRawPods(obj.GetCluster(), obj.GetNamespace())
	if err != nil {
		return nil, err
	}
	return FilterPodsByControllerRef(obj.GetK8SObject().(metav1.Object), pods), nil
}

func GetPodsByController(obj model.IK8SModel) ([]*api.Pod, error) {
	pods, err := GetRawPodsByController(obj)
	if err != nil {
		return nil, err
	}
	return PodManager.GetAPIPods(obj.GetCluster(), pods)
}

// FilterPodsByControllerResource returns a subset of pods controlled by given deployment.
func FilterDeploymentPodsByOwnerReference(deployment *apps.Deployment, allRS []*apps.ReplicaSet,
	allPods []*v1.Pod) []*v1.Pod {
	var matchingPods []*v1.Pod
	for _, rs := range allRS {
		if metav1.IsControlledBy(rs, deployment) {
			matchingPods = append(matchingPods, FilterPodsByControllerRef(rs, allPods)...)
		}
	}

	return matchingPods
}

// FilterPodsByControllerRef returns a subset of pods controlled by given controller resource, excluding deployments.
func FilterPodsByControllerRef(owner metav1.Object, allPods []*v1.Pod) []*v1.Pod {
	var matchingPods []*v1.Pod
	for _, pod := range allPods {
		if metav1.IsControlledBy(pod, owner) {
			matchingPods = append(matchingPods, pod)
		}
	}
	return matchingPods
}

func FilterPodsForJob(job *batch.Job, pods []*v1.Pod) []*v1.Pod {
	result := make([]*v1.Pod, 0)
	for _, pod := range pods {
		if pod.Namespace == job.Namespace && pod.Labels["controller-uid"] ==
			job.Spec.Selector.MatchLabels["controller-uid"] {
			result = append(result, pod)
		}
	}

	return result
}

// GetContainerImages returns container image strings from the given pod spec.
func GetContainerImages(podTemplate *v1.PodSpec) []api.ContainerImage {
	containerImages := []api.ContainerImage{}
	for _, container := range podTemplate.Containers {
		containerImages = append(containerImages, api.ContainerImage{
			Name:  container.Name,
			Image: container.Image,
		})
	}
	return containerImages
}

// GetInitContainerImages returns init container image strings from the given pod spec.
func GetInitContainerImages(podTemplate *v1.PodSpec) []api.ContainerImage {
	initContainerImages := []api.ContainerImage{}
	for _, initContainer := range podTemplate.InitContainers {
		initContainerImages = append(initContainerImages, api.ContainerImage{
			Name:  initContainer.Name,
			Image: initContainer.Image})
	}
	return initContainerImages
}

// GetContainerNames returns the container image name without the version number from the given pod spec.
func GetContainerNames(podTemplate *v1.PodSpec) []string {
	var containerNames []string
	for _, container := range podTemplate.Containers {
		containerNames = append(containerNames, container.Name)
	}
	return containerNames
}

// GetInitContainerNames returns the init container image name without the version number from the given pod spec.
func GetInitContainerNames(podTemplate *v1.PodSpec) []string {
	var initContainerNames []string
	for _, initContainer := range podTemplate.InitContainers {
		initContainerNames = append(initContainerNames, initContainer.Name)
	}
	return initContainerNames
}

// EqualIgnoreHash returns true if two given podTemplateSpec are equal, ignoring the diff in value of Labels[pod-template-hash]
// We ignore pod-template-hash because the hash result would be different upon podTemplateSpec API changes
// (e.g. the addition of a new field will cause the hash code to change)
// Note that we assume input podTemplateSpecs contain non-empty labels
func EqualIgnoreHash(template1, template2 v1.PodTemplateSpec) bool {
	// First, compare template.Labels (ignoring hash)
	labels1, labels2 := template1.Labels, template2.Labels
	if len(labels1) > len(labels2) {
		labels1, labels2 = labels2, labels1
	}
	// We make sure len(labels2) >= len(labels1)
	for k, v := range labels2 {
		if labels1[k] != v && k != apps.DefaultDeploymentUniqueLabelKey {
			return false
		}
	}
	// Then, compare the templates without comparing their labels
	template1.Labels, template2.Labels = nil, nil
	return equality.Semantic.DeepEqual(template1, template2)
}

func GetPodInfo(obj model.IK8SModel, current int32, desired *int32, pods []*v1.Pod) (*api.PodInfo, error) {
	podInfo := getPodInfo(current, desired, pods)
	warnEvents, err := EventManager.GetWarningEventsByPods(obj.GetCluster(), pods)
	if err != nil {
		return nil, err
	}
	ws := make([]api.Event, len(warnEvents))
	for i := range warnEvents {
		ws[i] = *warnEvents[i]
	}
	podInfo.Warnings = ws
	return &podInfo, nil
}

// GetPodInfo returns aggregate information about a group of pods.
func getPodInfo(current int32, desired *int32, pods []*v1.Pod) api.PodInfo {
	result := api.PodInfo{
		Current:  current,
		Desired:  desired,
		Warnings: make([]api.Event, 0),
	}

	for _, pod := range pods {
		switch pod.Status.Phase {
		case v1.PodRunning:
			result.Running++
		case v1.PodPending:
			result.Pending++
		case v1.PodFailed:
			result.Failed++
		case v1.PodSucceeded:
			result.Succeeded++
		}
	}

	return result
}

// Methods below are taken from kubernetes repo:
// https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/deployment/util/deployment_util.go

// FindNewReplicaSet returns the new RS this given deployment targets (the one with the same pod template).
func FindNewReplicaSet(deployment *apps.Deployment, rsList []*apps.ReplicaSet) (*apps.ReplicaSet, error) {
	newRSTemplate := GetNewReplicaSetTemplate(deployment)
	for i := range rsList {
		if EqualIgnoreHash(rsList[i].Spec.Template, newRSTemplate) {
			// This is the new ReplicaSet.
			return rsList[i], nil
		}
	}
	// new ReplicaSet does not exist.
	return nil, nil
}

// GetNewReplicaSetTemplate returns the desired PodTemplateSpec for the new ReplicaSet corresponding to the given ReplicaSet.
// Callers of this helper need to set the DefaultDeploymentUniqueLabelKey k/v pair.
func GetNewReplicaSetTemplate(deployment *apps.Deployment) v1.PodTemplateSpec {
	// newRS will have the same template as in deployment spec.
	return v1.PodTemplateSpec{
		ObjectMeta: deployment.Spec.Template.ObjectMeta,
		Spec:       deployment.Spec.Template.Spec,
	}
}

// GetExternalEndpoints returns endpoints that are externally reachable for a service.
func GetExternalEndpoints(service *v1.Service) []api.Endpoint {
	var externalEndpoints []api.Endpoint
	if service.Spec.Type == v1.ServiceTypeLoadBalancer {
		for _, ingress := range service.Status.LoadBalancer.Ingress {
			externalEndpoints = append(externalEndpoints, getExternalEndpoint(ingress, service.Spec.Ports))
		}
	}

	for _, ip := range service.Spec.ExternalIPs {
		externalEndpoints = append(externalEndpoints, api.Endpoint{
			Host:  ip,
			Ports: GetServicePorts(service.Spec.Ports),
		})
	}

	return externalEndpoints
}

// GetInternalEndpoint returns internal endpoint name for the given service properties, e.g.,
// "my-service.namespace 80/TCP" or "my-service 53/TCP,53/UDP".
func GetInternalEndpoint(serviceName, namespace string, ports []v1.ServicePort) api.Endpoint {
	name := serviceName

	if namespace != v1.NamespaceDefault && len(namespace) > 0 && len(serviceName) > 0 {
		bufferName := bytes.NewBufferString(name)
		bufferName.WriteString(".")
		bufferName.WriteString(namespace)
		name = bufferName.String()
	}

	return api.Endpoint{
		Host:  name,
		Ports: GetServicePorts(ports),
	}
}

// Returns external endpoint name for the given service properties.
func getExternalEndpoint(ingress v1.LoadBalancerIngress, ports []v1.ServicePort) api.Endpoint {
	var host string
	if ingress.Hostname != "" {
		host = ingress.Hostname
	} else {
		host = ingress.IP
	}
	return api.Endpoint{
		Host:  host,
		Ports: GetServicePorts(ports),
	}
}

// GetServicePorts returns human readable name for the given service ports list.
func GetServicePorts(apiPorts []v1.ServicePort) []api.ServicePort {
	var ports []api.ServicePort
	for _, port := range apiPorts {
		ports = append(ports, api.ServicePort{port.Port, port.Protocol, port.NodePort})
	}
	return ports
}

func CreateServiceIfNotExist(ctx *model.RequestContext, objMeta *metav1.ObjectMeta, opt *api.ServiceCreateOption) (*v1.Service, error) {
	svc, err := ctx.Cluster().GetHandler().GetIndexer().ServiceLister().Services(objMeta.GetNamespace()).Get(objMeta.GetName())
	if err != nil {
		if errors.IsNotFound(err) {
			return CreateServiceByOption(ctx, objMeta, opt)
		}
		return nil, err
	}
	return svc, nil
}

func CreateServiceByOption(ctx *model.RequestContext, objMeta *metav1.ObjectMeta, opt *api.ServiceCreateOption) (*v1.Service, error) {
	svc := GetServiceFromOption(objMeta, opt)
	if svc == nil {
		return nil, nil
	}
	return CreateService(ctx, svc)
}

func CreateService(ctx *model.RequestContext, svc *v1.Service) (*v1.Service, error) {
	cli := ctx.Cluster().GetClientset()
	return cli.CoreV1().Services(svc.GetNamespace()).Create(svc)
}

func getPodResourceVolumes(pod *v1.Pod, predicateF func(v1.Volume) bool) []v1.Volume {
	var cfgs []v1.Volume
	vols := pod.Spec.Volumes
	for _, vol := range vols {
		if predicateF(vol) {
			cfgs = append(cfgs, vol)
		}
	}
	return cfgs
}

func GetPodSecretVolumes(pod *v1.Pod) []v1.Volume {
	return getPodResourceVolumes(pod, func(vol v1.Volume) bool {
		return vol.VolumeSource.Secret != nil
	})
}

func GetPodConfigMapVolumes(pod *v1.Pod) []v1.Volume {
	return getPodResourceVolumes(pod, func(vol v1.Volume) bool {
		return vol.VolumeSource.ConfigMap != nil
	})
}

func GetConfigMapsForPod(pod *v1.Pod, cfgs []*v1.ConfigMap) []*v1.ConfigMap {
	if len(cfgs) == 0 {
		return nil
	}
	ret := make([]*v1.ConfigMap, 0)
	uniqM := make(map[string]bool, 0)
	for _, cfg := range cfgs {
		for _, vol := range GetPodConfigMapVolumes(pod) {
			if vol.ConfigMap.Name == cfg.GetName() {
				if _, ok := uniqM[cfg.GetName()]; !ok {
					uniqM[cfg.GetName()] = true
					ret = append(ret, cfg)
				}
			}
		}
	}
	return ret
}

func GetSecretsForPod(pod *v1.Pod, ss []*v1.Secret) []*v1.Secret {
	if len(ss) == 0 {
		return nil
	}
	ret := make([]*v1.Secret, 0)
	uniqM := make(map[string]bool, 0)
	for _, s := range ss {
		for _, vol := range GetPodSecretVolumes(pod) {
			if vol.Secret.SecretName == s.GetName() {
				if _, ok := uniqM[s.GetName()]; !ok {
					uniqM[s.GetName()] = true
					ret = append(ret, s)
				}
			}
		}
	}
	return ret
}

// objs is: []*objs, e.g.: []*v1.Pod{}
// targets is: the pointer of []*v, e.g.: &[]*api.Pod{}
func ConvertRawToAPIObjects(
	man model.IK8SModelManager,
	cluster model.ICluster,
	objs interface{},
	targets interface{}) error {
	objsVal := reflect.ValueOf(objs)
	// get targets slice value
	targetsValue := reflect.Indirect(reflect.ValueOf(targets))
	for i := 0; i < objsVal.Len(); i++ {
		objVal := objsVal.Index(i)
		// get targetType *v, the pointer of targetType
		targetPtrType := targetsValue.Type().Elem()
		// get targetType v
		targetType := targetPtrType.Elem()
		// target is the *v instance
		target := reflect.New(targetType).Interface()
		if err := ConvertRawToAPIObject(man, cluster, objVal.Interface().(runtime.Object), target); err != nil {
			return err
		}
		newTargets := reflect.Append(targetsValue, reflect.ValueOf(target))
		targetsValue.Set(newTargets)
	}
	return nil
}

func ConvertRawToAPIObject(
	man model.IK8SModelManager,
	cluster model.ICluster,
	obj runtime.Object, target interface{}) error {
	mObj, err := model.NewK8SModelObject(man, cluster, obj)
	if err != nil {
		return err
	}
	mv := reflect.ValueOf(mObj)
	funcVal, err := model.FindFunc(mv, model.DMethodGetAPIObject)
	if err != nil {
		return err
	}
	ret := funcVal.Call(nil)
	if len(ret) != 2 {
		return fmt.Errorf("invalidate %s %s return value number", man.Keyword(), model.DMethodGetAPIObject)
	}
	if err := model.ValueToError(ret[1]); err != nil {
		return err
	}
	targetVal := reflect.ValueOf(target)
	targetVal.Elem().Set(ret[0].Elem())
	return nil
}

type condtionSorter []*api.Condition

func (s condtionSorter) Len() int {
	return len(s)
}

func (s condtionSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s condtionSorter) Less(i, j int) bool {
	c1 := s[i]
	c2 := s[j]
	return c1.LastTransitionTime.Before(&c2.LastTransitionTime)
}

func SortConditions(conds []*api.Condition) []*api.Condition {
	sort.Sort(condtionSorter(conds))
	return conds
}
