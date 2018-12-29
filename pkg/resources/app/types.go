package app

import (
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/service"
)

const (
	// DescriptionAnnotationKey is annotation key for a description.
	DescriptionAnnotationKey = "description"
)

// AppDeploymentSpec is a specification for an app deployment.
type AppDeploymentSpec struct {
	// Name of the application.
	Name string `json:"name"`

	// Docker image path for the application.
	ContainerImage string `json:"containerImage"`

	// The name of an image pull secret in case of a private docker repository.
	ImagePullSecret *string `json:"imagePullSecret"`

	// Command that is executed instead of container entrypoint, if specified.
	ContainerCommand *string `json:"containerCommand"`

	// Arguments for the specified container command or container entrypoint (if command is not
	// specified here).
	ContainerCommandArgs *string `json:"containerCommandArgs"`

	// Number of replicas of the image to maintain.
	Replicas int32 `json:"replicas"`

	// Port mappings for the service that is created. The service is created if there is at least
	// one port mapping.
	PortMappings []service.PortMapping `json:"portMappings"`

	// List of user-defined environment variables.
	Variables []EnvironmentVariable `json:"variables"`

	// Whether the created service is external.
	IsExternal bool `json:"isExternal"`

	// LoadBalancerNetworkId
	LoadBalancerNetworkId string `json:"loadBalancerNetwork"`

	// Description of the deployment.
	Description *string `json:"description"`

	// Target namespace of the application.
	Namespace string `json:"namespace"`

	// Optional memory requirement for the container.
	MemoryRequirement *resource.Quantity `json:"memoryRequirement"`

	// Optional CPU requirement for the container.
	CpuRequirement *resource.Quantity `json:"cpuRequirement"`

	// Labels that will be defined on Pods/RCs/Services
	Labels []Label `json:"labels"`

	// Whether to run the container as privileged user (essentially equivalent to root on the host).
	RunAsPrivileged bool `json:"runAsPrivileged"`

	// Network config
	NetworkConfig *common.NetworkConfig `json:"networkConfig"`

	// Controller type. e.g. deployment, statefulset, daemonset, job
	ControllerType string `json:"controllerType"`

	RestartPolicy string `json:"restartPolicy"`

	// Pod volumes
	Volumes []api.Volume `json:"volumes"`

	// Container volume mounts
	VolumeMounts []api.VolumeMount `json:"volumeMounts"`

	VolumeClaimTemplates []PersistentVolumeClaim `json:"volumeClaimTemplates"`
}

type PersistentVolumeClaim struct {
	Name         string `json:"name"`
	StorageClass string `json:"storageClass"`
	Size         string `json:"size"`
	//AccessModes []string `json:"accessModes"`
}

func (spec AppDeploymentSpec) GetTemplatePVCs() ([]api.PersistentVolumeClaim, error) {
	pvcs := []api.PersistentVolumeClaim{}
	for _, pt := range spec.VolumeClaimTemplates {
		storageSize, err := resource.ParseQuantity(pt.Size)
		if err != nil {
			return nil, err
		}
		pvc := api.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pt.Name,
				Namespace: spec.Namespace,
			},
			Spec: api.PersistentVolumeClaimSpec{
				AccessModes: []api.PersistentVolumeAccessMode{api.ReadWriteOnce},
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{
						"storage": storageSize,
					},
				},
			},
		}
		pvcs = append(pvcs, pvc)
	}
	return pvcs, nil
}

// AppDeploymentFromFileSpec is a specification for deployment from file
type AppDeploymentFromFileSpec struct {
	// Name of the file
	Name string `json:"name"`

	// Namespace that object should be deployed in
	Namespace string `json:"namespace"`

	// File content
	Content string `json:"content"`

	// Whether validate content before creation or not
	Validate bool `json:"validate"`
}

// AppDeploymentFromFileResponse is a specification for deployment from file
type AppDeploymentFromFileResponse struct {
	// Name of the file
	Name string `json:"name"`

	// File content
	Content string `json:"content"`

	// Error after create resource
	//Error string `json:"error"`
}

// EnvironmentVariable represents a named variable accessible for containers.
type EnvironmentVariable struct {
	// Name of the variable. Must be a C_IDENTIFIER.
	Name string `json:"name"`

	// Value of the variable, as defined in Kubernetes core API.
	Value string `json:"value"`
}

// Label is a structure representing label assignable to Pod/RC/Service
type Label struct {
	// Label key
	Key string `json:"key"`

	// Label value
	Value string `json:"value"`
}

// Protocols is a structure representing supported protocol types for a service
type Protocols struct {
	// Array containing supported protocol types e.g., ["TCP", "UDP"]
	Protocols []api.Protocol `json:"protocols"`
}
