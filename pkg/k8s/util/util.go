package util

import (
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	yerrors "yunion.io/x/pkg/util/errors"
	"yunion.io/x/pkg/util/workqueue"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

var (
	ParallelizeWorks = 4
)

func IsK8sResourceExist(checkF func() (interface{}, error)) (bool, error) {
	_, err := checkF()
	if errors.IsNotFound(err) {
		return false, nil
	}
	if errors.IsAlreadyExists(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func IsNamespaceExist(cli kubernetes.Interface, name string) (bool, error) {
	return IsK8sResourceExist(func() (interface{}, error) {
		return cli.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
	})
}

func CreateNamespace(cli kubernetes.Interface, name string) error {
	opt := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	_, err := cli.CoreV1().Namespaces().Create(opt)
	return err
}

func EnsureResourceFunc(
	existsF func() (bool, error),
	createF func() error,
) error {
	exists, err := existsF()
	if err != nil {
		return err
	}
	if !exists {
		err = createF()
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}
	return nil
}

func EnsureNamespace(cli kubernetes.Interface, name string) error {
	return EnsureResourceFunc(
		func() (bool, error) {
			return IsNamespaceExist(cli, name)
		},
		func() error {
			return CreateNamespace(cli, name)
		})
}

func EnsureNamespaces(cli kubernetes.Interface, names ...string) error {
	return Parallelize(func(name string) error {
		return EnsureNamespace(cli, name)
	}, names...)
}

func Parallelize(execF func(string) error, args ...string) error {
	errsChannel := make(chan error, len(args))
	workqueue.Parallelize(ParallelizeWorks, len(args), func(i int) {
		err := execF(args[i])
		if err != nil {
			errsChannel <- err
			return
		}
	})
	if len(errsChannel) > 0 {
		errs := make([]error, 0)
		length := len(errsChannel)
		for ; length > 0; length-- {
			errs = append(errs, <-errsChannel)
		}
		return yerrors.NewAggregate(errs)
	}
	return nil
}

func GetPVCList(cli kubernetes.Interface, namespace string) (*apiv1.PersistentVolumeClaimList, error) {
	if namespace == "" {
		namespace = apiv1.NamespaceAll
	}
	list, err := cli.CoreV1().PersistentVolumeClaims(namespace).List(api.ListEverything)
	return list, err
}
