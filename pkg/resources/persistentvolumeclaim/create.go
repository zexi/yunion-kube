package persistentvolumeclaim

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SPersistentVolumeClaimManager) ValidateCreateData(req *common.Request) error {
	return man.SNamespaceResourceManager.ValidateCreateData(req)
}

func (man *SPersistentVolumeClaimManager) Create(req *common.Request) (interface{}, error) {
	objMeta, err := common.GetK8sObjectCreateMeta(req.Data)
	if err != nil {
		return nil, err
	}

	size, err := req.Data.GetString("size")
	if err != nil {
		return nil, err
	}
	storageSize, err := resource.ParseQuantity(size)
	if err != nil {
		return nil, err
	}
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: *objMeta,
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					"storage": storageSize,
				},
			},
		},
	}

	storageClass, _ := req.Data.GetString("storageClass")
	if storageClass != "" {
		pvc.Spec.StorageClassName = &storageClass
	}

	ns := req.GetDefaultNamespace()
	obj, err := req.GetK8sClient().CoreV1().PersistentVolumeClaims(ns).Create(pvc)
	return obj, err
}
