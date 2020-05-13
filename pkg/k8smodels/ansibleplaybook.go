package k8smodels

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"yunion.io/x/jsonutils"
	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	AnsiblePlaybookManager *SAnsiblePlaybookManager
)

func init() {
	AnsiblePlaybookManager = &SAnsiblePlaybookManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(new(SAnsiblePlaybook), "k8s_ansibleplaybook", "k8s_ansibleplaybooks"),
	}
	AnsiblePlaybookManager.SetVirtualObject(AnsiblePlaybookManager)
}

type SAnsiblePlaybookManager struct {
	model.SK8SNamespaceResourceBaseManager
}

type SAnsiblePlaybook struct {
	model.SK8SNamespaceResourceBase
	UnstructuredResourceBase
}

func (m *SAnsiblePlaybookManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNameAnsiblePlaybook,
		KindName:     apis.KindNameAnsiblePlaybook,
		Object:       &unstructured.Unstructured{},
	}
}

func (obj *SAnsiblePlaybook) GetAPIObject() (*apis.AnsiblePlaybook, error) {
	out := new(apis.AnsiblePlaybook)
	if err := obj.ConvertToAPIObject(obj, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (obj *SAnsiblePlaybook) FillAPIObjectBySpec(specObj jsonutils.JSONObject, out IUnstructuredOutput) error {
	ret := out.(*apis.AnsiblePlaybook)
	if tmplateName, err := specObj.GetString("playbookTemplateRef", "name"); err == nil {
		ret.PlaybookTemplateRef = &apis.LocalObjectReference{
			Name: tmplateName,
		}
	}
	if maxRetryTime, _ := specObj.Int("maxRetryTimes"); maxRetryTime > 0 {
		mrt := int32(maxRetryTime)
		ret.MaxRetryTime = &mrt
	}
	return nil
}

func (obj *SAnsiblePlaybook) FillAPIObjectByStatus(statusObj jsonutils.JSONObject, out IUnstructuredOutput) error {
	ret := out.(*apis.AnsiblePlaybook)
	phase, _ := statusObj.GetString("phase")
	ret.AnsiblePlaybookStatus.Status = phase

	tryTimes, _ := statusObj.Int("tryTimes")
	ret.TryTimes = int32(tryTimes)

	extInfo := &ret.AnsiblePlaybookStatus.ExternalInfo
	if extraInfoObj, err := statusObj.Get("externalInfo"); err == nil {
		extraInfoObj.Unmarshal(extInfo)
	}
	return nil
}
