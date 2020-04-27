package k8smodels

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/drivers"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

const (
	IsDefaultStorageClassAnnotation     = "storageclass.kubernetes.io/is-default-class"
	betaIsDefaultStorageClassAnnotation = "storageclass.beta.kubernetes.io/is-default-class"
)

var (
	StorageClassManager *SStorageClassManager
	_                   model.IK8SModel = new(SStorageClass)
)

func init() {
	StorageClassManager = &SStorageClassManager{
		SK8SClusterResourceBaseManager: model.NewK8SClusterResourceBaseManager(
			&SStorageClass{},
			"storageclass",
			"storageclasses"),
		driverManager: drivers.NewDriverManager(""),
	}
	StorageClassManager.SetVirtualObject(StorageClassManager)
}

type SStorageClassManager struct {
	model.SK8SClusterResourceBaseManager
	driverManager *drivers.DriverManager
}

type SStorageClass struct {
	model.SK8SClusterResourceBase
}

type IStorageClassDriver interface {
	ConnectionTest(ctx *model.RequestContext, input *apis.StorageClassCreateInput) (*apis.StorageClassTestResult, error)
	ValidateCreateData(ctx *model.RequestContext, input *apis.StorageClassCreateInput) (*apis.StorageClassCreateInput, error)
	ToStorageClassParams(input *apis.StorageClassCreateInput) (map[string]string, error)
}

func (m *SStorageClassManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNameStorageClass,
		Object:       new(v1.StorageClass),
		KindName:     apis.KindNameStorageClass,
	}
}

func (m *SStorageClassManager) RegisterDriver(provisioner string, driver IStorageClassDriver) {
	if err := m.driverManager.Register(driver, provisioner); err != nil {
		panic(errors.Wrapf(err, "storageclass register driver %s", provisioner))
	}
}

func (m *SStorageClassManager) GetDriver(provisioner string) (IStorageClassDriver, error) {
	drv, err := m.driverManager.Get(provisioner)
	if err != nil {
		if errors.Cause(err) == drivers.ErrDriverNotFound {
			return nil, httperrors.NewNotFoundError("storageclass get %s driver", provisioner)
		}
		return nil, err
	}
	return drv.(IStorageClassDriver), nil
}

func (m *SStorageClassManager) ValidateCreateData(
	ctx *model.RequestContext, query *jsonutils.JSONDict, input *apis.StorageClassCreateInput) (*apis.StorageClassCreateInput, error) {
	if _, err := m.SK8SClusterResourceBaseManager.ValidateCreateData(ctx, query, &input.K8sClusterResourceCreateInput); err != nil {
		return nil, err
	}
	if input.Provisioner == "" {
		return nil, httperrors.NewNotEmptyError("provisioner is empty")
	}
	drv, err := m.GetDriver(input.Provisioner)
	if err != nil {
		return nil, err
	}
	return drv.ValidateCreateData(ctx, input)
}

func (m *SStorageClassManager) NewK8SRawObjectForCreate(
	ctx *model.RequestContext, query *jsonutils.JSONDict, input *apis.StorageClassCreateInput) (
	runtime.Object, error) {
	drv, err := m.GetDriver(input.Provisioner)
	if err != nil {
		return nil, err
	}
	params, err := drv.ToStorageClassParams(input)
	if err != nil {
		return nil, err
	}
	objMeta := input.ToObjectMeta()
	return &v1.StorageClass{
		ObjectMeta:        objMeta,
		Provisioner:       input.Provisioner,
		Parameters:        params,
		ReclaimPolicy:     input.ReclaimPolicy,
		MountOptions:      input.MountOptions,
		VolumeBindingMode: input.VolumeBindingMode,
		AllowedTopologies: input.AllowedTopologies,
	}, nil
}

func (obj *SStorageClassManager) GetRawStorageClasses(cluster model.ICluster) ([]*v1.StorageClass, error) {
	return cluster.GetHandler().GetIndexer().StorageClassLister().List(labels.Everything())
}

func (obj *SStorageClass) GetRawStorageClass() *v1.StorageClass {
	return obj.GetK8SObject().(*v1.StorageClass)
}

func (obj *SStorageClass) GetAPIObject() (*apis.StorageClass, error) {
	sc := obj.GetRawStorageClass()
	isDefault := false
	if _, ok := sc.Annotations[IsDefaultStorageClassAnnotation]; ok {
		isDefault = true
	}
	if _, ok := sc.Annotations[betaIsDefaultStorageClassAnnotation]; ok {
		isDefault = true
	}
	return &apis.StorageClass{
		ObjectMeta:  obj.GetObjectMeta(),
		TypeMeta:    obj.GetTypeMeta(),
		Provisioner: sc.Provisioner,
		Parameters:  sc.Parameters,
		IsDefault:   isDefault,
	}, nil
}

func (obj *SStorageClass) GetPVs() ([]*apis.PersistentVolume, error) {
	pvs, err := PVManager.GetRawPVs(obj.GetCluster())
	if err != nil {
		return nil, err
	}
	ret := make([]*corev1.PersistentVolume, 0)
	for _, pv := range pvs {
		if strings.Compare(pv.Spec.StorageClassName, obj.GetName()) == 0 {
			ret = append(ret, pv)
		}
	}
	return PVManager.GetAPIPVs(obj.GetCluster(), pvs)
}

func (obj *SStorageClass) GetAPIDetailObject() (*apis.StorageClassDetail, error) {
	apiObj, err := obj.GetAPIObject()
	if err != nil {
		return nil, err
	}
	pvs, err := obj.GetPVs()
	if err != nil {
		return nil, err
	}
	return &apis.StorageClassDetail{
		StorageClass:      *apiObj,
		PersistentVolumes: pvs,
	}, nil
}

func (m *SStorageClassManager) PerformClassConnectionTest(
	ctx *model.RequestContext, _ *jsonutils.JSONDict, input *apis.StorageClassCreateInput) (*apis.StorageClassTestResult, error) {
	drv, err := m.GetDriver(input.Provisioner)
	if err != nil {
		return nil, err
	}
	return drv.ConnectionTest(ctx, input)
}

func (obj *SStorageClass) PerformSetDefault(ctx *model.RequestContext, _, _ *jsonutils.JSONDict) (*v1.StorageClass, error) {
	scList, err := StorageClassManager.GetRawStorageClasses(ctx.Cluster())
	if err != nil {
		return nil, err
	}
	var defaultSc *v1.StorageClass
	k8sCli := ctx.Cluster().GetClientset()
	for _, sc := range scList {
		_, hasDefault := sc.Annotations[IsDefaultStorageClassAnnotation]
		_, hasBeta := sc.Annotations[betaIsDefaultStorageClassAnnotation]
		if sc.Annotations == nil {
			sc.Annotations = make(map[string]string)
		}
		if sc.Name == obj.GetName() || hasDefault || hasBeta {
			delete(sc.Annotations, IsDefaultStorageClassAnnotation)
			delete(sc.Annotations, betaIsDefaultStorageClassAnnotation)
			if sc.Name == obj.GetName() {
				sc.Annotations[IsDefaultStorageClassAnnotation] = "true"
				defaultSc = sc
			}
			_, err := k8sCli.StorageV1().StorageClasses().Update(sc)
			if err != nil {
				return nil, err
			}
		}
	}
	return defaultSc, nil
}
