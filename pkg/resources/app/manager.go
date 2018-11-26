package app

import (
	"k8s.io/api/core/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/yunion-kube/pkg/resources"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/types/apis"
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

func ValidateCreateData(req *common.Request) error {
	data := req.Data
	controllerType, _ := data.GetString("controllerType")
	if !AppControllerTypes.Has(controllerType) {
		return httperrors.NewInputParameterError("Invalid app controller type: %s", controllerType)
	}
	err := common.ValidateK8sResourceCreateData(req, controllerType, true)
	if err != nil {
		return err
	}
	replica, _ := data.Int("replicas")
	if replica == 0 {
		data.Set("replicas", jsonutils.NewInt(1))
	}

	restartPolicy, _ := data.GetString("restartPolicy")
	if restartPolicy == "" {
		policy := v1.RestartPolicyAlways
		if sets.NewString(apis.ResourceKindCronJob, apis.ResourceKindJob).Has(controllerType) {
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

func Create(req *common.Request, createFunc CreateResourceFunc) (interface{}, error) {
	appSpec, err := NewAppCreateData(req.Data)
	if err != nil {
		return nil, err
	}

	spec, err := DeployApp(appSpec, req.GetK8sClient(), createFunc)
	if err != nil {
		return nil, err
	}
	return spec, nil
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
	_, err = DeployAppFromFile(req.K8sConfig, &deploymentSpec)
	if err != nil {
		return nil, err
	}
	return &AppDeploymentFromFileResponse{
		Name:    deploymentSpec.Name,
		Content: deploymentSpec.Content,
	}, nil
}
