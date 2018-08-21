package namespace

import (
	"k8s.io/api/core/v1"

	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

func ToNamespace(namespace v1.Namespace) Namespace {
	return Namespace{
		ObjectMeta: api.NewObjectMeta(namespace.ObjectMeta),
		TypeMeta:   api.NewTypeMeta(api.ResourceKindNamespace),
		Phase:      namespace.Status.Phase,
	}
}
