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
	"path/filepath"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/runtime"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/cli-runtime/pkg/resource"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/helm"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
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

func NewCheckIdOrNameError(res, resName string, err error) error {
	if errors.Cause(err) == sql.ErrNoRows {
		return httperrors.NewNotFoundError(fmt.Sprintf("resource %s/%s not found: %v", res, resName, err))
	}
	if errors.Cause(err) == sqlchemy.ErrDuplicateEntry {
		return httperrors.NewDuplicateResourceError(fmt.Sprintf("resource %s/%s duplicate: %v", res, resName, err))
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
func GetReleaseResources(
	cli *helm.Client, rel *release.Release,
	clusterMan model.ICluster,
) (map[string][]interface{}, error) {
	cfg := cli.GetConfig()
	ress, err := cfg.KubeClient.Build(bytes.NewBufferString(rel.Manifest), true)
	if err != nil {
		return nil, err
	}
	ret := make(map[string][]interface{})
	ress.Visit(func(info *resource.Info, err error) error {
		gvk := info.Object.GetObjectKind().GroupVersionKind()
		man := model.GetK8SModelManagerByKind(gvk.Kind)
		if man == nil {
			log.Warningf("not fond %s manager", gvk.Kind)
			return nil
		}
		keyword := man.Keyword()
		unstructObj := info.Object.(*unstructured.Unstructured)
		newObj := man.GetK8SResourceInfo().Object.DeepCopyObject()
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructObj.Object, newObj); err != nil {
			return err
		}
		namespace := info.Namespace
		metaObj := newObj.(metav1.Object)
		modelObj, err := model.NewK8SModelObjectByName(man, clusterMan, namespace, metaObj.GetName())
		if err != nil {
			return err
		}
		obj, err := model.GetObject(modelObj)
		if err != nil {
			return err
		}
		if list, ok := ret[keyword]; ok {
			list = append(list, obj)
		} else {
			list = []interface{}{obj}
			ret[keyword] = list
		}
		return nil
	})
	return ret, nil
}

func GetChartRawFiles(chObj *chart.Chart) []*chart.File {
	files := make([]*chart.File, len(chObj.Raw))
	for idx, rf := range chObj.Raw {
		files[idx] = &chart.File{
			Name: filepath.Join(chObj.Name(), rf.Name),
			Data: rf.Data,
		}
	}
	return files
}

func GetK8SObjectTypeMeta(kObj runtime.Object) metav1.TypeMeta {
	v := reflect.ValueOf(kObj)
	f := reflect.Indirect(v).FieldByName("TypeMeta")
	if !f.IsValid() {
		panic(fmt.Sprintf("get invalid object meta %#v", kObj))
	}
	return f.Interface().(metav1.TypeMeta)
}

