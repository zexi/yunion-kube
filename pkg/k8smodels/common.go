package k8smodels

import (
	"k8s.io/api/core/v1"

	"yunion.io/x/onecloud/pkg/httperrors"
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
