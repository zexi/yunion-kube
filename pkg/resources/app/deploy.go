package app

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/yaml"
	dynamicclient "k8s.io/client-go/deprecated-dynamic"
	"k8s.io/client-go/discovery"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/yunion-kube/pkg/types/apis"
)

var (
	AppControllerTypes = sets.NewString(
		apis.ResourceKindDeployment,
		apis.ResourceKindStatefulSet,
		apis.ResourceKindDaemonSet,
		apis.ResourceKindCronJob,
		apis.ResourceKindJob,
	)
)

func NewAppCreateData(data jsonutils.JSONObject) (*AppDeploymentSpec, error) {
	appSpec := AppDeploymentSpec{}
	dataStr, err := data.GetString()
	if err != nil {
		return nil, err
	}
	err = json.NewDecoder(strings.NewReader(dataStr)).Decode(&appSpec)
	if err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}

	// check labels
	if len(appSpec.Labels) == 0 {
		// set default label run=<name>
		appSpec.Labels = append(appSpec.Labels, Label{
			Key:   "run",
			Value: appSpec.Name,
		})
	}

	if appSpec.NetworkConfig != nil {
		if addr := appSpec.NetworkConfig.Address; addr != "" {
			if !regutils.MatchIP4Addr(addr) {
				return nil, httperrors.NewInputParameterError("Invalid network ip address format: %q", addr)
			}
		}
	}

	return &appSpec, nil
}

type CreateResourceFunc func(client.Interface, metaV1.ObjectMeta, map[string]string, api.PodTemplateSpec, *AppDeploymentSpec) error

// DeployApp deploys an app based on the given configuration. The app is deployed using the given
// client. App deployment consists of a deployment and an optional service. Both of them
// share common labels.
func DeployApp(spec *AppDeploymentSpec, cli client.Interface, createFunc CreateResourceFunc) (*AppDeploymentSpec, error) {
	controllerType := spec.ControllerType
	log.Infof("Deploying %q application into %q namespace, type %q", spec.Name, spec.Namespace, controllerType)

	// parse annotations
	annotations := make(map[string]string)
	if spec.NetworkConfig != nil {
		annotations = spec.NetworkConfig.ToPodAnnotation()
	}
	if spec.Description != nil {
		annotations[DescriptionAnnotationKey] = *spec.Description
	}

	// parse labels
	labels := getLabelsMap(spec.Labels)
	objectMeta := metaV1.ObjectMeta{
		Annotations: annotations,
		Name:        spec.Name,
		Labels:      labels,
	}

	// get podTemplate
	podTemplate := api.PodTemplateSpec{
		ObjectMeta: objectMeta,
		Spec:       getPodSpec(spec),
	}

	var err error

	if len(spec.PortMappings) > 0 {
		// create service
		err = createAppService(cli, objectMeta, labels, spec)
		if err != nil {
			return nil, err
		}
	}

	err = createFunc(cli, objectMeta, labels, podTemplate, spec)
	if err != nil {
		// TODO: Roll back created resources in case of error.
		return nil, err
	}

	return spec, err
}

func createAppService(
	cli client.Interface,
	objectMeta metaV1.ObjectMeta,
	labels map[string]string,
	spec *AppDeploymentSpec,
) error {
	service := &api.Service{
		ObjectMeta: objectMeta,
		Spec: api.ServiceSpec{
			Selector: labels,
		},
	}
	if spec.IsExternal {
		service.Spec.Type = api.ServiceTypeLoadBalancer
	} else {
		service.Spec.Type = api.ServiceTypeClusterIP
	}

	for _, portMapping := range spec.PortMappings {
		servicePort := api.ServicePort{
			Protocol: portMapping.Protocol,
			Port:     portMapping.Port,
			Name:     generatePortMappingName(portMapping),
			TargetPort: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: portMapping.TargetPort,
			},
		}
		service.Spec.Ports = append(service.Spec.Ports, servicePort)
	}
	_, err := cli.CoreV1().Services(spec.Namespace).Create(service)
	return err
}

func getPodSpec(spec *AppDeploymentSpec) api.PodSpec {
	// parse container spec
	containerSpec := api.Container{
		Name:  spec.Name,
		Image: spec.ContainerImage,
		SecurityContext: &api.SecurityContext{
			Privileged: &spec.RunAsPrivileged,
		},
		Resources: api.ResourceRequirements{
			Requests: make(map[api.ResourceName]resource.Quantity),
		},
		Env: convertEnvVarsSpec(spec.Variables),
	}

	if spec.ContainerCommand != nil {
		containerSpec.Command = []string{*spec.ContainerCommand}
	}
	if spec.ContainerCommandArgs != nil {
		containerSpec.Args = []string{*spec.ContainerCommandArgs}
	}

	if spec.CpuRequirement != nil {
		containerSpec.Resources.Requests[api.ResourceCPU] = *spec.CpuRequirement
	}
	if spec.MemoryRequirement != nil {
		containerSpec.Resources.Requests[api.ResourceMemory] = *spec.MemoryRequirement
	}
	podSpec := api.PodSpec{
		Containers: []api.Container{containerSpec},
	}

	podSpec.RestartPolicy = api.RestartPolicy(spec.RestartPolicy)

	if spec.ImagePullSecret != nil {
		podSpec.ImagePullSecrets = []api.LocalObjectReference{{Name: *spec.ImagePullSecret}}
	}
	return podSpec
}

// Converts array of labels to map[string]string
func getLabelsMap(labels []Label) map[string]string {
	result := make(map[string]string)

	for _, label := range labels {
		result[label.Key] = label.Value
	}

	return result
}

// GetAvailableProtocols returns list of available protocols. Currently it is TCP and UDP.
func GetAvailableProtocols() *Protocols {
	return &Protocols{Protocols: []api.Protocol{api.ProtocolTCP, api.ProtocolUDP}}
}

func convertEnvVarsSpec(variables []EnvironmentVariable) []api.EnvVar {
	var result []api.EnvVar
	for _, variable := range variables {
		result = append(result, api.EnvVar{Name: variable.Name, Value: variable.Value})
	}
	return result
}

func generatePortMappingName(portMapping PortMapping) string {
	return generateName(fmt.Sprintf("%s-%d-%d-", strings.ToLower(string(portMapping.Protocol)),
		portMapping.Port, portMapping.TargetPort))
}

func generateName(base string) string {
	maxNameLength := 63
	randomLength := 5
	maxGeneratedNameLength := maxNameLength - randomLength
	if len(base) > maxGeneratedNameLength {
		base = base[:maxGeneratedNameLength]
	}
	return fmt.Sprintf("%s%s", base, rand.String(randomLength))
}

// DeployAppFromFile deploys an app based on the given yaml or json file.
func DeployAppFromFile(cfg *rest.Config, spec *AppDeploymentFromFileSpec) (bool, error) {
	reader := strings.NewReader(spec.Content)
	log.Infof("Namespace for deploy from file: %s", spec.Namespace)
	d := yaml.NewYAMLOrJSONDecoder(reader, 4096)
	for {
		data := unstructured.Unstructured{}
		if err := d.Decode(&data); err != nil {
			if err == io.EOF {
				return true, nil
			}
			return false, err
		}

		version := data.GetAPIVersion()
		kind := data.GetKind()

		gv, err := schema.ParseGroupVersion(version)
		if err != nil {
			gv = schema.GroupVersion{Version: version}
		}

		groupVersionKind := schema.GroupVersionKind{Group: gv.Group, Version: gv.Version, Kind: kind}

		discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
		if err != nil {
			return false, err
		}

		apiResourceList, err := discoveryClient.ServerResourcesForGroupVersion(version)
		if err != nil {
			return false, err
		}
		apiResources := apiResourceList.APIResources
		var resource *metaV1.APIResource
		for _, apiResource := range apiResources {
			if apiResource.Kind == kind && !strings.Contains(apiResource.Name, "/") {
				resource = &apiResource
				break
			}
		}
		if resource == nil {
			return false, fmt.Errorf("Unknown resource kind: %s", kind)
		}

		dynamicClientPool := dynamicclient.NewDynamicClientPool(cfg)

		dynamicClient, err := dynamicClientPool.ClientForGroupVersionKind(groupVersionKind)

		if err != nil {
			return false, err
		}

		// FIXME: _all is invalid
		if strings.Compare(spec.Namespace, "_all") == 0 {
			_, err = dynamicClient.Resource(resource, data.GetNamespace()).Create(&data)
		} else {
			_, err = dynamicClient.Resource(resource, spec.Namespace).Create(&data)
		}

		if err != nil {
			return false, err
		}
	}
	return true, nil
}
