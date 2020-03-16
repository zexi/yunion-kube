package storageclass

import (
	v1 "k8s.io/api/storage/v1"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SStorageClassManager) ValidateCreateData(req *common.Request) error {
	if err := man.SClusterResourceManager.ValidateCreateData(req); err != nil {
		return err
	}
	input := new(apis.StorageClassCreateInput)
	if err := req.DataUnmarshal(input); err != nil {
		return err
	}
	if input.Provisioner == "" {
		return httperrors.NewNotEmptyError("provisioner is empty")
	}
	drv, err := man.GetDriver(input.Provisioner)
	if err != nil {
		return err
	}
	return drv.ValidateCreateData(req, input)
}

func (man *SStorageClassManager) Create(req *common.Request) (interface{}, error) {
	input := new(apis.StorageClassCreateInput)
	if err := req.DataUnmarshal(input); err != nil {
		return nil, err
	}
	drv, err := man.GetDriver(input.Provisioner)
	if err != nil {
		return nil, err
	}
	params, err := drv.ToStorageClassParams(input)
	if err != nil {
		return nil, err
	}
	objMeta, err := common.GetK8sObjectCreateMeta(req.Data)
	if err != nil {
		return nil, err
	}
	storageClass := &v1.StorageClass{
		ObjectMeta: *objMeta,
		Provisioner: input.Provisioner,
		ReclaimPolicy: input.ReclaimPolicy,
		AllowVolumeExpansion: input.AllowVolumeExpansion,
		MountOptions: input.MountOptions,
		VolumeBindingMode: input.VolumeBindingMode,
		AllowedTopologies: input.AllowedTopologies,
		Parameters: params,
	}
	cli := req.GetK8sClient()
	obj, err := cli.StorageV1().StorageClasses().Create(storageClass)
	return obj, err
}

func (man *SStorageClassManager) ConnectionTest(req *common.Request) (interface{}, error) {
	input := new(apis.StorageClassCreateInput)
	if err := req.DataUnmarshal(input); err != nil {
		return nil, err
	}
	drv, err := man.GetDriver(input.Provisioner)
	if err != nil {
		return nil, err
	}
	ret, err := drv.ConnectionTest(input)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
