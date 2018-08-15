package deployment

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"

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
	labels, _ := data.Get("labels")
	if labels == nil {
		labels = jsonutils.Marshal([]Label{
			Label{
				Key:   "run",
				Value: name,
			},
		})
	}
	data.Set("labels", labels)
	return nil
}

func (man *SDeploymentManager) Create(req *common.Request) (jsonutils.JSONObject, error) {
	appSpec := AppDeploymentSpec{}
	err := req.Data.Unmarshal(&appSpec)
	if err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}
	spec, err := DeployApp(&appSpec, req.GetK8sClient())
	if err != nil {
		return nil, err
	}
	log.Infof("========DeployApp spec: %#v", spec)
	return jsonutils.Marshal(spec), nil
}
