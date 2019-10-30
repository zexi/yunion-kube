package pod

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"

	"k8s.io/api/core/v1"
	res "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	"yunion.io/x/yunion-kube/pkg/resources/persistentvolumeclaim"
)

func (man *SPodManager) Get(req *common.Request, id string) (interface{}, error) {
	namespace := req.GetNamespaceQuery().ToRequestParam()
	return GetPodDetail(req.GetIndexer(), req.GetCluster(), namespace, id)
}

func GetPodDetail(indexer *client.CacheFactory, cluster api.ICluster, namespace, name string) (*api.PodDetail, error) {
	log.Infof("Getting details of %s pod in %s namespace", name, namespace)
	channels := &common.ResourceChannels{
		ConfigMapList: common.GetConfigMapListChannel(indexer, common.NewSameNamespaceQuery(namespace)),
		SecretList:    common.GetSecretListChannel(indexer, common.NewSameNamespaceQuery(namespace)),
		EventList:     common.GetEventListChannel(indexer, common.NewSameNamespaceQuery(namespace)),
	}

	pod, err := indexer.PodLister().Pods(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	configMapList := <-channels.ConfigMapList.List
	err = <-channels.ConfigMapList.Error
	if err != nil {
		return nil, err
	}

	secretList := <-channels.SecretList.List
	if err != nil {
		return nil, err
	}

	rawEventList := <-channels.EventList.List
	err = <-channels.EventList.Error
	if err != nil {
		return nil, err
	}

	eventList, err := GetEventsForPod(indexer, cluster, dataselect.DefaultDataSelect(), pod.Namespace, pod.Name)
	if err != nil {
		return nil, err
	}

	warnings := event.GetPodsEventWarnings(rawEventList, []*v1.Pod{pod})
	commonPod := ToPod(pod, warnings, configMapList, secretList, cluster)

	persistentVolumeClaimList, err := persistentvolumeclaim.GetPodPersistentVolumeClaims(indexer, cluster, namespace, name, dataselect.DefaultDataSelect())
	if err != nil {
		return nil, err
	}

	configMapList = common.GetConfigMapsForPod(pod, configMapList)
	secretList = common.GetSecretsForPod(pod, secretList)

	podDetail := toPodDetail(
		commonPod, pod, eventList,
		persistentVolumeClaimList.Items,
		common.ToConfigMaps(configMapList, cluster),
		common.ToSecrets(secretList, cluster),
	)
	return &podDetail, nil
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

func toPodDetail(
	commonPod api.Pod,
	pod *v1.Pod,
	events *common.EventList,
	pvcs []api.PersistentVolumeClaim,
	cfgs []api.ConfigMap,
	secrets []api.Secret,
) api.PodDetail {
	return api.PodDetail{
		Pod: commonPod,
		//Controller:                controller,
		//Metrics:                   metrics,
		Conditions:                getPodConditions(*pod),
		Events:                    events.Events,
		PersistentvolumeclaimList: pvcs,
		ConfigMaps:                cfgs,
		Secrets:                   secrets,
	}
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
			log.Errorf("%v", err)
			return ""
		}
		gvk := gv.WithKind("Pod")
		internalFieldPath, _, err := runtime.NewScheme().ConvertFieldLabel(gvk, src.FieldRef.FieldPath, "")
		if err != nil {
			log.Errorf("%v", err)
			return ""
		}
		valueFrom, err := ExtractFieldPathAsString(pod, internalFieldPath)
		if err != nil {
			log.Errorf("%v", err)
			return ""
		}
		return valueFrom
	}
	return ""
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
