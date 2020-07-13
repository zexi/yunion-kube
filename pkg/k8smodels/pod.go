package k8smodels

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	res "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/k8s/common/getters"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

type SPodManager struct {
	model.SK8SNamespaceResourceBaseManager
	model.SK8SOwnerResourceBaseManager
}

var PodManager *SPodManager

func init() {
	PodManager = &SPodManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			&SPod{},
			"pod",
			"pods"),
	}
	PodManager.SetVirtualObject(PodManager)
}

var (
	_ model.IK8SModel = &SPod{}
)

type SPod struct {
	model.SK8SNamespaceResourceBase
}

func (m SPodManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: api.ResourceNamePod,
		Object:       &v1.Pod{},
		KindName:     api.KindNamePod,
	}
}

func (p SPodManager) ValidateCreateData(
	ctx *model.RequestContext,
	query, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewUnsupportOperationError("kubernetes pod not support create")
}

func (p SPodManager) ListItemFilter(ctx *model.RequestContext, q model.IQuery, query *api.PodListInput) (model.IQuery, error) {
	// q, err := p.SK8SNamespaceResourceBaseManager.ListItemFilter(ctx, q, query.ListInputK8SNamespaceBase)
	// if err != nil {
	// return nil, err
	// }
	q, err := p.SK8SOwnerResourceBaseManager.ListItemFilter(ctx, q, query.ListInputOwner)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (obj *SPod) IsOwnerBy(ownerModel model.IK8SModel) (bool, error) {
	return model.IsPodOwner(ownerModel.(model.IPodOwnerModel), obj.GetRawPod())
}

func (p SPodManager) GetRawPods(cluster model.ICluster, ns string) ([]*v1.Pod, error) {
	return p.GetRawPodsBySelector(cluster, ns, labels.Everything())
}

func (p SPodManager) GetRawPodsBySelector(cluster model.ICluster, ns string, selecotr labels.Selector) ([]*v1.Pod, error) {
	indexer := cluster.GetHandler().GetIndexer()
	return indexer.PodLister().Pods(ns).List(selecotr)
}

func (p SPodManager) GetAllRawPods(cluster model.ICluster) ([]*v1.Pod, error) {
	return p.GetRawPods(cluster, v1.NamespaceAll)
}

func (p *SPodManager) GetAPIPods(cluster model.ICluster, pods []*v1.Pod) ([]*api.Pod, error) {
	ret := make([]*api.Pod, 0)
	err := ConvertRawToAPIObjects(p, cluster, pods, &ret)
	return ret, err
}

func (obj *SPod) GetRawPod() *v1.Pod {
	return obj.GetK8SObject().(*v1.Pod)
}

func (obj *SPod) GetRawConfigMaps() ([]*v1.ConfigMap, error) {
	cfgs, err := ConfigMapManager.GetRawConfigMaps(obj.GetCluster(), obj.GetNamespace())
	if err != nil {
		return nil, err
	}
	cfgs = GetConfigMapsForPod(obj.GetRawPod(), cfgs)
	return cfgs, nil
}

func (obj *SPod) GetAPIConfigMaps() ([]*api.ConfigMap, error) {
	cfgs, err := obj.GetRawConfigMaps()
	if err != nil {
		return nil, err
	}
	return ConfigMapManager.GetAPIConfigMaps(obj.GetCluster(), cfgs)
}

func (obj *SPod) GetRawSecrets() ([]*v1.Secret, error) {
	ss, err := SecretManager.GetRawSecrets(obj.GetCluster(), obj.GetNamespace())
	if err != nil {
		return nil, err
	}
	ss = GetSecretsForPod(obj.GetRawPod(), ss)
	return ss, nil
}

func (obj *SPod) GetAPISecrets() ([]*api.Secret, error) {
	ss, err := obj.GetRawSecrets()
	if err != nil {
		return nil, err
	}
	return SecretManager.GetAPISecrets(obj.GetCluster(), ss)
}

func (obj *SPod) GetAPIObject() (*api.Pod, error) {
	pod := obj.GetRawPod()
	cluster := obj.GetCluster()
	warnings, err := EventManager.GetWarningEventsByPods(cluster, []*v1.Pod{pod})
	secrets, err := SecretManager.GetRawSecrets(cluster, pod.GetNamespace())
	if err != nil {
		return nil, err
	}
	configmaps, err := ConfigMapManager.GetRawConfigMaps(cluster, pod.GetNamespace())
	if err != nil {
		return nil, err
	}
	return &api.Pod{
		ObjectMeta:          api.NewObjectMeta(pod.ObjectMeta, cluster),
		TypeMeta:            api.NewTypeMeta(pod.TypeMeta),
		Warnings:            warnings,
		PodStatus:           PodManager.getPodStatus(pod),
		RestartCount:        PodManager.getRestartCount(pod),
		PodIP:               pod.Status.PodIP,
		QOSClass:            string(pod.Status.QOSClass),
		Containers:          extractContainerInfo(pod.Spec.Containers, pod, configmaps, secrets),
		InitContainers:      extractContainerInfo(pod.Spec.InitContainers, pod, configmaps, secrets),
		ContainerImages:     GetContainerImages(&pod.Spec),
		InitContainerImages: GetInitContainerImages(&pod.Spec),
	}, nil
}

func (obj *SPod) GetPVCs() ([]*api.PersistentVolumeClaim, error) {
	return PVCManager.GetPodAPIPVCs(obj.GetCluster(), obj.GetRawPod())
}

func (obj *SPod) GetEvents() ([]*api.Event, error) {
	return EventManager.GetEventsByObject(obj)
}

func (obj *SPod) getConditions() []*api.Condition {
	var conds []*api.Condition
	pod := obj.GetRawPod()
	for _, cond := range pod.Status.Conditions {
		conds = append(conds, &api.Condition{
			Type:               string(cond.Type),
			Status:             cond.Status,
			LastProbeTime:      cond.LastProbeTime,
			LastTransitionTime: cond.LastTransitionTime,
			Reason:             cond.Reason,
			Message:            cond.Message,
		})
	}
	return SortConditions(conds)
}

func (obj *SPod) GetAPIDetailObject() (*api.PodDetail, error) {
	apiObj, err := obj.GetAPIObject()
	if err != nil {
		return nil, err
	}
	secrets, err := obj.GetAPISecrets()
	if err != nil {
		return nil, err
	}
	cfgs, err := obj.GetAPIConfigMaps()
	if err != nil {
		return nil, err
	}
	events, err := obj.GetEvents()
	if err != nil {
		return nil, err
	}
	pvcs, err := obj.GetPVCs()
	if err != nil {
		return nil, err
	}
	return &api.PodDetail{
		Pod:                    *apiObj,
		Conditions:             obj.getConditions(),
		Events:                 events,
		Persistentvolumeclaims: pvcs,
		ConfigMaps:             cfgs,
		Secrets:                secrets,
	}, nil
}

func (p SPodManager) getPodStatus(pod *v1.Pod) api.PodStatus {
	var states []v1.ContainerState
	for _, containerStatus := range pod.Status.ContainerStatuses {
		states = append(states, containerStatus.State)
	}
	return api.PodStatus{
		PodStatusV2:     *getters.GetPodStatus(pod),
		PodPhase:        pod.Status.Phase,
		ContainerStates: states,
	}
}

func (p SPodManager) getPodConditions(pod *v1.Pod) []api.Condition {
	var conditions []api.Condition
	for _, condition := range pod.Status.Conditions {
		conditions = append(conditions, api.Condition{
			Type:               string(condition.Type),
			Status:             condition.Status,
			LastProbeTime:      condition.LastProbeTime,
			LastTransitionTime: condition.LastTransitionTime,
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}
	return conditions
}

func (p SPodManager) getRestartCount(pod *v1.Pod) int32 {
	var restartCount int32 = 0
	for _, containerStatus := range pod.Status.ContainerStatuses {
		restartCount += containerStatus.RestartCount
	}
	return restartCount
}

// extractContainerResourceValue extracts the value of a resource in an already known container.
func extractContainerResourceValue(fs *v1.ResourceFieldSelector, container *v1.Container) (string,
	error) {
	divisor := res.Quantity{}
	if divisor.Cmp(fs.Divisor) == 0 {
		divisor = res.MustParse("1")
	} else {
		divisor = fs.Divisor
	}

	switch fs.Resource {
	case "limits.cpu":
		return strconv.FormatInt(int64(math.Ceil(float64(container.Resources.Limits.
			Cpu().MilliValue())/float64(divisor.MilliValue()))), 10), nil
	case "limits.memory":
		return strconv.FormatInt(int64(math.Ceil(float64(container.Resources.Limits.
			Memory().Value())/float64(divisor.Value()))), 10), nil
	case "requests.cpu":
		return strconv.FormatInt(int64(math.Ceil(float64(container.Resources.Requests.
			Cpu().MilliValue())/float64(divisor.MilliValue()))), 10), nil
	case "requests.memory":
		return strconv.FormatInt(int64(math.Ceil(float64(container.Resources.Requests.
			Memory().Value())/float64(divisor.Value()))), 10), nil
	}

	return "", fmt.Errorf("Unsupported container resource : %v", fs.Resource)
}

// evalValueFrom evaluates environment value from given source. For more details check:
// https://github.com/kubernetes/kubernetes/blob/d82e51edc5f02bff39661203c9b503d054c3493b/pkg/kubectl/describe.go#L1056
func evalValueFrom(src *v1.EnvVarSource, container *v1.Container, pod *v1.Pod,
	configMaps []*v1.ConfigMap, secrets []*v1.Secret) string {
	switch {
	case src.ConfigMapKeyRef != nil:
		name := src.ConfigMapKeyRef.LocalObjectReference.Name
		for _, configMap := range configMaps {
			if configMap.ObjectMeta.Name == name {
				return configMap.Data[src.ConfigMapKeyRef.Key]
			}
		}
	case src.SecretKeyRef != nil:
		name := src.SecretKeyRef.LocalObjectReference.Name
		for _, secret := range secrets {
			if secret.ObjectMeta.Name == name {
				return base64.StdEncoding.EncodeToString([]byte(
					secret.Data[src.SecretKeyRef.Key]))
			}
		}
	case src.ResourceFieldRef != nil:
		valueFrom, err := extractContainerResourceValue(src.ResourceFieldRef, container)
		if err != nil {
			valueFrom = ""
		}
		resource := src.ResourceFieldRef.Resource
		if valueFrom == "0" && (resource == "limits.cpu" || resource == "limits.memory") {
			valueFrom = "node allocatable"
		}
		return valueFrom
	case src.FieldRef != nil:
		gv, err := schema.ParseGroupVersion(src.FieldRef.APIVersion)
		if err != nil {
			log.V(2).Warningf("%v", err)
			return ""
		}
		gvk := gv.WithKind("Pod")
		internalFieldPath, _, err := runtime.NewScheme().ConvertFieldLabel(gvk, src.FieldRef.FieldPath, "")
		if err != nil {
			log.V(2).Warningf("%v", err)
			return ""
		}
		valueFrom, err := ExtractFieldPathAsString(pod, internalFieldPath)
		if err != nil {
			log.V(2).Warningf("%v", err)
			return ""
		}
		return valueFrom
	}
	return ""
}

// FormatMap formats map[string]string to a string.
func FormatMap(m map[string]string) (fmtStr string) {
	for key, value := range m {
		fmtStr += fmt.Sprintf("%v=%q\n", key, value)
	}
	fmtStr = strings.TrimSuffix(fmtStr, "\n")

	return
}

// ExtractFieldPathAsString extracts the field from the given object
// and returns it as a string.  The object must be a pointer to an
// API type.
func ExtractFieldPathAsString(obj interface{}, fieldPath string) (string, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return "", nil
	}

	switch fieldPath {
	case "metadata.annotations":
		return FormatMap(accessor.GetAnnotations()), nil
	case "metadata.labels":
		return FormatMap(accessor.GetLabels()), nil
	case "metadata.name":
		return accessor.GetName(), nil
	case "metadata.namespace":
		return accessor.GetNamespace(), nil
	}

	return "", fmt.Errorf("unsupported fieldPath: %v", fieldPath)
}

func extractContainerInfo(containerList []v1.Container, pod *v1.Pod, configMaps []*v1.ConfigMap, secrets []*v1.Secret) []api.Container {
	containers := make([]api.Container, 0)
	for _, container := range containerList {
		vars := make([]api.EnvVar, 0)
		for _, envVar := range container.Env {
			variable := api.EnvVar{
				Name:      envVar.Name,
				Value:     envVar.Value,
				ValueFrom: envVar.ValueFrom,
			}
			if variable.ValueFrom != nil {
				variable.Value = evalValueFrom(variable.ValueFrom, &container, pod,
					configMaps, secrets)
			}
			vars = append(vars, variable)
		}
		vars = append(vars, evalEnvFrom(container, configMaps, secrets)...)

		containers = append(containers, api.Container{
			Name:     container.Name,
			Image:    container.Image,
			Env:      vars,
			Commands: container.Command,
			Args:     container.Args,
		})
	}
	return containers
}

func evalEnvFrom(container v1.Container, configMaps []*v1.ConfigMap, secrets []*v1.Secret) []api.EnvVar {
	vars := make([]api.EnvVar, 0)
	for _, envFromVar := range container.EnvFrom {
		switch {
		case envFromVar.ConfigMapRef != nil:
			name := envFromVar.ConfigMapRef.LocalObjectReference.Name
			for _, configMap := range configMaps {
				if configMap.ObjectMeta.Name == name {
					for key, value := range configMap.Data {
						valueFrom := &v1.EnvVarSource{
							ConfigMapKeyRef: &v1.ConfigMapKeySelector{
								LocalObjectReference: v1.LocalObjectReference{
									Name: name,
								},
								Key: key,
							},
						}
						variable := api.EnvVar{
							Name:      envFromVar.Prefix + key,
							Value:     value,
							ValueFrom: valueFrom,
						}
						vars = append(vars, variable)
					}
					break
				}
			}
		case envFromVar.SecretRef != nil:
			name := envFromVar.SecretRef.LocalObjectReference.Name
			for _, secret := range secrets {
				if secret.ObjectMeta.Name == name {
					for key, value := range secret.Data {
						valueFrom := &v1.EnvVarSource{
							SecretKeyRef: &v1.SecretKeySelector{
								LocalObjectReference: v1.LocalObjectReference{
									Name: name,
								},
								Key: key,
							},
						}
						variable := api.EnvVar{
							Name:      envFromVar.Prefix + key,
							Value:     base64.StdEncoding.EncodeToString(value),
							ValueFrom: valueFrom,
						}
						vars = append(vars, variable)
					}
					break
				}
			}
		}
	}
	return vars
}
