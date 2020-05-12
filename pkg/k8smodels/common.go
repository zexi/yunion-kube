package k8smodels

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"yunion.io/x/log"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"

	"yunion.io/x/yunion-kube/pkg/apis"
)

type PodTemplateResourceBase struct{}

func (_ PodTemplateResourceBase) UpdatePodTemplate(temp *v1.PodTemplateSpec, input apis.PodTemplateUpdateInput) error {
	if len(input.RestartPolicy) != 0 {
		temp.Spec.RestartPolicy = input.RestartPolicy
	}
	if len(input.DNSPolicy) != 0 {
		temp.Spec.DNSPolicy = input.DNSPolicy
	}
	cf := func(container *v1.Container, cs []apis.ContainerUpdateInput) error {
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

type ReplicaResourceBase struct{}

func (_ ReplicaResourceBase) ValidateUpdateData(r *int32) error {
	if r != nil {
		if *r < 0 {
			return httperrors.NewInputParameterError("replica %d is less than 0", *r)
		}
	}
	return nil
}

type UnstructuredResourceBase struct{}

func (_ UnstructuredResourceBase) GetUnstructuredObject(m model.IK8SModel) *unstructured.Unstructured {
	return m.GetK8SObject().(*unstructured.Unstructured)
}

func (res UnstructuredResourceBase) GetRawJSONObject(m model.IK8SModel) (jsonutils.JSONObject, error) {
	rawObj := res.GetUnstructuredObject(m)
	jsonBytes, err := rawObj.MarshalJSON()
	if err != nil {
		return nil, err
	}
	jsonObj, err := jsonutils.Parse(jsonBytes)
	if err != nil {
		return nil, errors.Wrapf(err, "parse json bytes %q", string(jsonBytes))
	}
	return jsonObj, nil
}

type IUnstructuredOutput interface {
	SetObjectMeta(meta apis.ObjectMeta) *apis.ObjectTypeMeta
	SetTypeMeta(meta apis.TypeMeta) *apis.ObjectTypeMeta
}

type IK8SUnstructuredModel interface {
	model.IK8SModel

	FillAPIObjectBySpec(rawObjSpec jsonutils.JSONObject, output IUnstructuredOutput) error
	FillAPIObjectByStatus(rawObjStatus jsonutils.JSONObject, output IUnstructuredOutput) error
}

func (res UnstructuredResourceBase) ConvertToAPIObject(m IK8SUnstructuredModel, output IUnstructuredOutput) error {
	output.SetObjectMeta(m.GetObjectMeta()).SetTypeMeta(m.GetTypeMeta())
	jsonObj, err := res.GetRawJSONObject(m)
	if err != nil {
		return errors.Wrap(err, "get json object")
	}
	specObj, err := jsonObj.Get("spec")
	if err != nil {
		log.Errorf("Get spec object error: %v", err)
	} else {
		if err := m.FillAPIObjectBySpec(specObj, output); err != nil {
			log.Errorf("FillAPIObjectBySpec error: %v", err)
		}
	}
	statusObj, err := jsonObj.Get("status")
	if err != nil {
		log.Errorf("Get status object error: %v", err)
	} else {
		if err := m.FillAPIObjectByStatus(statusObj, output); err != nil {
			log.Errorf("FillAPIObjectByStatus error: %v", err)
		}
	}
	return nil
}
