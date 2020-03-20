package storageclass

import (
	v1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/drivers"
	"yunion.io/x/yunion-kube/pkg/resources"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

const (
	IsDefaultStorageClassAnnotation     = "storageclass.kubernetes.io/is-default-class"
	betaIsDefaultStorageClassAnnotation = "storageclass.beta.kubernetes.io/is-default-class"
)

var StorageClassManager *SStorageClassManager

type SStorageClassManager struct {
	*resources.SClusterResourceManager
	driverManager *drivers.DriverManager
}

func init() {
	StorageClassManager = &SStorageClassManager{
		SClusterResourceManager: resources.NewClusterResourceManager("storageclass", "storageclasses"),
		driverManager:           drivers.NewDriverManager(""),
	}
}

type IStorageClassDriver interface {
	ConnectionTest(req *common.Request, input *apis.StorageClassCreateInput) (*apis.StorageClassTestResult, error)
	ValidateCreateData(req *common.Request, input *apis.StorageClassCreateInput) error
	ToStorageClassParams(input *apis.StorageClassCreateInput) (map[string]string, error)
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

func (man *SStorageClassManager) AllowPerformSetDefault(req *common.Request, id string) bool {
	return man.SClusterResourceManager.AllowUpdateItem(req, id)
}

func (man *SStorageClassManager) PerformSetDefault(req *common.Request, id string) (*v1.StorageClass, error) {
	lister := req.GetIndexer().StorageClassLister()
	scList, err := lister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	k8sCli := req.GetK8sClient()
	var defaultSc *v1.StorageClass
	for _, sc := range scList {
		_, hasDefault := sc.Annotations[IsDefaultStorageClassAnnotation]
		_, hasBeta := sc.Annotations[betaIsDefaultStorageClassAnnotation]
		if sc.Annotations == nil {
			sc.Annotations = make(map[string]string)
		}
		if sc.Name == id || hasDefault || hasBeta {
			delete(sc.Annotations, IsDefaultStorageClassAnnotation)
			delete(sc.Annotations, betaIsDefaultStorageClassAnnotation)
			if sc.Name == id {
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
