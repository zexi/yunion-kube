package clusters

import (
	"context"
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

type sBaseDriver struct{}

func newBaseDriver() *sBaseDriver {
	return &sBaseDriver{}
}

func (d *sBaseDriver) ValidateCreateData(userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
	return nil
}

func (d *sBaseDriver) ValidateDeleteCondition() error {
	return nil
}

func (d *sBaseDriver) CreateClusterResource(man *clusters.SClusterManager, data *types.CreateClusterData) error {
	// do nothing
	return nil
}

func (d *sBaseDriver) ValidateAddMachine(man *clusters.SClusterManager, machine *types.CreateMachineData) error {
	return nil
}

func (d *sBaseDriver) GetAddonsManifest(cluster *clusters.SCluster) (string, error) {
	return "", nil
}

func (d *sBaseDriver) UseClusterAPI() bool {
	return false
}

func (d *sBaseDriver) RequestDeleteCluster(c *clusters.SCluster) error {
	return fmt.Errorf("Not supported")
}

func (d *sBaseDriver) ValidateAddMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, data []*types.CreateMachineData) error {
	return nil
}

func (d *sBaseDriver) StartSyncStatus(cluster *clusters.SCluster, ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterSyncstatusTask", cluster, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

type sClusterAPIDriver struct {
	*sBaseDriver
}

func newClusterAPIDriver() *sClusterAPIDriver {
	return &sClusterAPIDriver{
		sBaseDriver: newBaseDriver(),
	}
}

func (d *sClusterAPIDriver) UseClusterAPI() bool {
	return true
}

func (d *sClusterAPIDriver) EnsureNamespace(cli *kubernetes.Clientset, namespace string) error {
	ns := apiv1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: namespace,
		},
	}
	_, err := cli.CoreV1().Namespaces().Create(&ns)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (d *sClusterAPIDriver) DeleteNamespace(cli *kubernetes.Clientset, namespace string) error {
	if namespace == apiv1.NamespaceDefault {
		return nil
	}

	err := cli.CoreV1().Namespaces().Delete(namespace, &v1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (d *sClusterAPIDriver) ValidateCreateData(userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
	ok, err := clusters.ClusterManager.IsSystemClusterReady()
	if err != nil {
		return err
	}
	if !ok {
		return httperrors.NewNotAcceptableError("System k8s cluster default not running")
	}
	return nil
}
