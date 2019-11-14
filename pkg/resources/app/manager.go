package app

import (
	"k8s.io/api/core/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/util/sets"

	api "yunion.io/x/yunion-kube/pkg/apis"
	clientapi "yunion.io/x/yunion-kube/pkg/client/api"
	"yunion.io/x/yunion-kube/pkg/k8s"
	"yunion.io/x/yunion-kube/pkg/resources"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

var (
	AppFromFileManager *SAppFromFileManager
)

type SAppFromFileManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	AppFromFileManager = &SAppFromFileManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("appfromfile", "appfromfiles"),
	}
}

func ValidateCreateData(req *common.Request, man k8s.IK8sResourceManager) error {
	controllerType := clientapi.TranslateKindPlural(man.KeywordPlural())
	if !AppControllerTypes.Has(controllerType) {
		return httperrors.NewInputParameterError("Invalid app controller type: %s", controllerType)
	}
	err := common.ValidateK8sResourceCreateData(req, controllerType, true)
	if err != nil {
		return err
	}
	data := req.Data
	replica, _ := data.Int("replicas")
	if replica == 0 {
		data.Set("replicas", jsonutils.NewInt(1))
	}

	restartPolicy, _ := data.GetString("restartPolicy")
	if restartPolicy == "" {
		policy := v1.RestartPolicyAlways
		if sets.NewString(clientapi.ResourceNameCronJob, clientapi.ResourceNameJob).Has(controllerType) {
			policy = v1.RestartPolicyNever
		}
		data.Set("restartPolicy", jsonutils.NewString(string(policy)))
	} else if !sets.NewString(
		string(v1.RestartPolicyAlways),
		string(v1.RestartPolicyNever),
		string(v1.RestartPolicyOnFailure),
	).Has(restartPolicy) {
		return httperrors.NewInputParameterError("Invalid restartPolicy %s", restartPolicy)
	}

	return nil
}

func (man *SAppFromFileManager) ValidateCreateData(req *common.Request) error {
	return nil
}

func (man *SAppFromFileManager) Create(req *common.Request) (interface{}, error) {
	deploymentSpec := AppDeploymentFromFileSpec{}
	err := req.Data.Unmarshal(&deploymentSpec)
	if err != nil {
		return nil, err
	}
	if deploymentSpec.Namespace == "" {
		deploymentSpec.Namespace = req.GetDefaultNamespace()
	}
	_, err = DeployAppFromFile(req.K8sConfig, &deploymentSpec)
	if err != nil {
		return nil, err
	}
	return &AppDeploymentFromFileResponse{
		Name:    deploymentSpec.Name,
		Content: deploymentSpec.Content,
	}, nil
}

func UpdatePodTemplate(temp *v1.PodTemplateSpec, input api.PodUpdateInput) error {
	if len(input.RestartPolicy) != 0 {
		temp.Spec.RestartPolicy = input.RestartPolicy
	}
	if len(input.DNSPolicy) != 0 {
		temp.Spec.DNSPolicy = input.DNSPolicy
	}
	cf := func(container *v1.Container, cs []api.ContainerUpdateInput) error {
		if len(cs) == 0 {
			return nil
		}
		for _, c := range cs {
			if container.Name == c.Name {
				container.Image = c.Image
				return nil
			}
		}
		return httperrors.NewNotFoundError("Not found container %s in input", container.Name)
	}
	for _, c := range temp.Spec.InitContainers {
		if err := cf(&c, input.InitContainers); err != nil {
			return err
		}
	}
	for i, c := range temp.Spec.Containers {
		if err := cf(&c, input.Containers); err != nil {
			return err
		}
		temp.Spec.Containers[i] = c
	}
	return nil
}
