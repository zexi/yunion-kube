package pod

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"

	"k8s.io/api/core/v1"
	res "k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	client "k8s.io/client-go/kubernetes"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	//"yunion.io/x/yunion-kube/pkg/resources/persistentvolumeclaim"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type PodDetail struct {
	api.ObjectMeta
	api.TypeMeta
	PodPhase       v1.PodPhase        `json:"podPhase"`
	PodIP          string             `json:"podIP"`
	NodeName       string             `json:"nodeName"`
	RestartCount   int32              `json:"restartCount"`
	QOSClass       string             `json:"qosClass"`
	Containers     []Container        `json:"containers"`
	InitContainers []Container        `json:"initContainers"`
	Conditions     []common.Condition `json:"conditions"`
	EventList      common.EventList   `json:"eventList"`
	//PersistentvolumeclaimList persistentvolumeclaim.PersistentVolumeClaimList `json:"persistentVolumeClaimList"`
}

// Container represents a docker/rkt/etc. container that lives in a pod.
type Container struct {
	// Name of the container.
	Name string `json:"name"`

	// Image URI of the container.
	Image string `json:"image"`

	// List of environment variables.
	Env []EnvVar `json:"env"`

	// Commands of the container
	Commands []string `json:"commands"`

	// Command arguments
	Args []string `json:"args"`
}

// EnvVar represents an environment variable of a container.
type EnvVar struct {
	// Name of the variable.
	Name string `json:"name"`

	// Value of the variable. May be empty if value from is defined.
	Value string `json:"value"`

	// Defined for derived variables. If non-null, the value is get from the reference.
	// Note that this is an API struct. This is intentional, as EnvVarSources are plain struct
	// references.
	ValueFrom *v1.EnvVarSource `json:"valueFrom"`
}

func (man *SPodManager) Get(req *common.Request, id string) (jsonutils.JSONObject, error) {
	namespace := req.GetNamespaceQuery().ToRequestParam()
	detail, err := GetPodDetail(req.GetK8sClient(), namespace, id)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(detail), nil
}

func GetPodDetail(client client.Interface, namespace, name string) (*PodDetail, error) {
	log.Infof("Getting details of %s pod in %s namespace", name, namespace)
	channels := &common.ResourceChannels{
		ConfigMapList: common.GetConfigMapListChannel(client, common.NewSameNamespaceQuery(namespace)),
		SecretList:    common.GetSecretListChannel(client, common.NewSameNamespaceQuery(namespace)),
	}

	pod, err := client.CoreV1().Pods(namespace).Get(name, metaV1.GetOptions{})
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

	eventList, err := GetEventsForPod(client, dataselect.DefaultDataSelect, pod.Namespace, pod.Name)
	if err != nil {
		return nil, err
	}

	podDetail := toPodDetail(pod, configMapList, secretList, eventList)
	return &podDetail, nil
}

func extractContainerInfo(containerList []v1.Container, pod *v1.Pod, configMaps *v1.ConfigMapList, secrets *v1.SecretList) []Container {
	containers := make([]Container, 0)
	for _, container := range containerList {
		vars := make([]EnvVar, 0)
		for _, envVar := range container.Env {
			variable := EnvVar{
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

		containers = append(containers, Container{
			Name:     container.Name,
			Image:    container.Image,
			Env:      vars,
			Commands: container.Command,
			Args:     container.Args,
		})
	}
	return containers
}

func toPodDetail(pod *v1.Pod, configMaps *v1.ConfigMapList, secrets *v1.SecretList, events *common.EventList) PodDetail {
	return PodDetail{
		ObjectMeta:   api.NewObjectMeta(pod.ObjectMeta),
		TypeMeta:     api.NewTypeMeta(api.ResourceKindPod),
		PodPhase:     pod.Status.Phase,
		PodIP:        pod.Status.PodIP,
		RestartCount: getRestartCount(*pod),
		QOSClass:     string(pod.Status.QOSClass),
		NodeName:     pod.Spec.NodeName,
		//Controller:                controller,
		Containers:     extractContainerInfo(pod.Spec.Containers, pod, configMaps, secrets),
		InitContainers: extractContainerInfo(pod.Spec.InitContainers, pod, configMaps, secrets),
		//Metrics:                   metrics,
		Conditions: getPodConditions(*pod),
		EventList:  *events,
		//PersistentvolumeclaimList: *persistentVolumeClaimList,
	}
}

func evalEnvFrom(container v1.Container, configMaps *v1.ConfigMapList, secrets *v1.SecretList) []EnvVar {
	vars := make([]EnvVar, 0)
	for _, envFromVar := range container.EnvFrom {
		switch {
		case envFromVar.ConfigMapRef != nil:
			name := envFromVar.ConfigMapRef.LocalObjectReference.Name
			for _, configMap := range configMaps.Items {
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
						variable := EnvVar{
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
			for _, secret := range secrets.Items {
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
						variable := EnvVar{
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
	configMaps *v1.ConfigMapList, secrets *v1.SecretList) string {
	switch {
	case src.ConfigMapKeyRef != nil:
		name := src.ConfigMapKeyRef.LocalObjectReference.Name
		for _, configMap := range configMaps.Items {
			if configMap.ObjectMeta.Name == name {
				return configMap.Data[src.ConfigMapKeyRef.Key]
			}
		}
	case src.SecretKeyRef != nil:
		name := src.SecretKeyRef.LocalObjectReference.Name
		for _, secret := range secrets.Items {
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
		internalFieldPath, _, err := runtime.NewScheme().ConvertFieldLabel(src.FieldRef.APIVersion,
			"Pod", src.FieldRef.FieldPath, "")
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
