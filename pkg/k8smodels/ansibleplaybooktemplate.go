package k8smodels

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"yunion.io/x/jsonutils"
	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	AnsiblePlaybookTemplateManager *SAnsiblePlaybookTemplateManager
)

func init() {
	AnsiblePlaybookTemplateManager = &SAnsiblePlaybookTemplateManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(new(SAnsiblePlaybook), "k8s_ansibleplaybooktemplate", "k8s_ansibleplaybooktemplates"),
	}
	AnsiblePlaybookTemplateManager.SetVirtualObject(AnsiblePlaybookTemplateManager)
}

type SAnsiblePlaybookTemplateManager struct {
	model.SK8SNamespaceResourceBaseManager
}

type SAnsiblePlaybookTemplate struct {
	model.SK8SNamespaceResourceBase
	UnstructuredResourceBase
}

func (m *SAnsiblePlaybookTemplateManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNameAnsiblePlaybookTemplate,
		KindName:     apis.KindNameAnsiblePlaybookTemplate,
		Object:       &unstructured.Unstructured{},
	}
}

func (obj *SAnsiblePlaybookTemplate) GetAPIObject() (*apis.AnsiblePlaybookTemplate, error) {
	out := new(apis.AnsiblePlaybookTemplate)
	if err := obj.ConvertToAPIObject(obj, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (obj *SAnsiblePlaybookTemplate) FillAPIObjectBySpec(specObj jsonutils.JSONObject, out IUnstructuredOutput) error {
	ret := out.(*apis.AnsiblePlaybookTemplate)
	if err := specObj.Unmarshal(&ret.AnsiblePlaybookTemplateSpec); err != nil {
		return err
	}
	return nil
}

func (obj *SAnsiblePlaybookTemplate) FillAPIObjectByStatus(statusObj jsonutils.JSONObject, out IUnstructuredOutput) error {
	return nil
}
