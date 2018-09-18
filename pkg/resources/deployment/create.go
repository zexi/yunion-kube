package deployment

import (
	"encoding/json"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SDeploymentManager) ValidateCreateData(req *common.Request) error {
	data := req.Data
	name, _ := data.GetString("name")
	if name == "" {
		return httperrors.NewInputParameterError("Name must provided")
	}
	replica, _ := data.Int("replicas")
	if replica == 0 {
		data.Set("replicas", jsonutils.NewInt(1))
	}
	namespace, _ := req.GetNamespaceByData()
	if namespace == "" {
		namespace = req.GetDefaultNamespace()
		data.Set("namespace", jsonutils.NewString(namespace))
	}
	return nil
}

func (man *SDeploymentManager) Create(req *common.Request) (interface{}, error) {
	appSpec := AppDeploymentSpec{}
	dataStr, err := req.Data.GetString()
	if err != nil {
		return nil, err
	}
	log.Errorf("====Get string: %s", dataStr)
	err = json.NewDecoder(strings.NewReader(dataStr)).Decode(&appSpec)
	//err := req.Data.Unmarshal(&appSpec)
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

	spec, err := DeployApp(&appSpec, req.GetK8sClient())
	if err != nil {
		return nil, err
	}
	return spec, nil
}

func (man *SDeployFromFileManager) ValidateCreateData(req *common.Request) error {
	return nil
}

func (man *SDeployFromFileManager) Create(req *common.Request) (interface{}, error) {
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
