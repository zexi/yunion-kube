package models

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/helm"
	k8sutil "yunion.io/x/yunion-kube/pkg/k8s/util"
)

func RunBatchTask(
	ctx context.Context,
	items []db.IStandaloneModel,
	userCred mcclient.TokenCredential,
	data jsonutils.JSONObject,
	taskName, parentTaskId string,
) error {
	params := data.(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewParallelTask(ctx, taskName, items, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return fmt.Errorf("%s newTask error %s", taskName, err)
	}
	task.ScheduleRun(nil)
	return nil
}

func (m *SClusterManager) GetSystemClusterKubeconfig(apiServer string, cfg *rest.Config) (string, error) {
	cli, err := kubernetes.NewForConfig(cfg)
	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		// return "", errors.Errorf("Not found NAMESPACE in env")
		ns = NamespaceOneCloud
	}
	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		return "", errors.Errorf("Not found HOSTNAME in env")
	}
	selfPod, err := cli.CoreV1().Pods(ns).Get(hostname, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "get pod %s/%s", ns, hostname)
	}
	svcAccount := selfPod.Spec.ServiceAccountName
	if err != nil {
		return "", errors.Wrap(err, "new kubernetes client")
	}
	token := cfg.BearerToken
	caData, err := ioutil.ReadFile(cfg.TLSClientConfig.CAFile)
	if err != nil {
		return "", errors.Wrapf(err, "read ca file %s", cfg.TLSClientConfig.CAFile)
	}

	tmplInput := map[string]string{
		"ClusterName": SystemClusterName,
		"Server":      apiServer,
		"Cert":        base64.StdEncoding.EncodeToString(caData),
		"User":        svcAccount,
		"Token":       token,
	}

	tmpl := `apiVersion: v1
kind: Config
clusters:
- name: "{{.ClusterName}}"
  cluster:
    server: "{{.Server}}"
    certificate-authority-data: "{{.Cert}}"
users:
- name: "{{.User}}"
  user:
    token: "{{.Token}}"
contexts:
- name: "{{.ClusterName}}"
  context:
    user: "{{.User}}"
    cluster: "{{.ClusterName}}"
current-context: "{{.ClusterName}}"
`
	outBuf := &bytes.Buffer{}
	if err := template.Must(template.New("tokenTemplate").Parse(tmpl)).Execute(outBuf, tmplInput); err != nil {
		return "", errors.Wrap(err, "generate kubeconfig")
	}
	return outBuf.String(), nil
}

func NewCheckIdOrNameError(msg string, err error) error {
	if errors.Cause(err) == sql.ErrNoRows {
		return httperrors.NewNotFoundError(msg, err)
	}
	if errors.Cause(err) == sqlchemy.ErrDuplicateEntry {
		return httperrors.NewDuplicateResourceError(msg, err)
	}
	return httperrors.NewGeneralError(err)
}

func NewHelmClient(cluster *SCluster, namespace string) (*helm.Client, error) {
	clusterMan, err := client.GetManagerByCluster(cluster)
	if err != nil {
		return nil, err
	}
	kubeconfigPath, err := clusterMan.GetKubeConfigPath()
	if err != nil {
		return nil, err
	}
	return helm.NewClient(kubeconfigPath, namespace, true)
}

func EnsureNamespace(cluster *SCluster, namespace string) error {
	k8sMan, err := client.GetManagerByCluster(cluster)
	if err != nil {
		return errors.Wrap(err, "get cluster k8s manager")
	}
	lister := k8sMan.GetIndexer().NamespaceLister()
	cli, err := cluster.GetK8sClient()
	if err != nil {
		return errors.Wrap(err, "get cluster k8s client")
	}
	return k8sutil.EnsureNamespace(lister, cli, namespace)
}

