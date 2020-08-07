package models

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubernetes/pkg/api/legacyscheme"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/helm"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
	k8sutil "yunion.io/x/yunion-kube/pkg/k8s/util"
	gotypesutil "yunion.io/x/yunion-kube/pkg/utils/gotypes"
	"yunion.io/x/yunion-kube/pkg/utils/k8serrors"
)

func RegisterSerializable(objs ...gotypes.ISerializable) {
	gotypesutil.RegisterSerializable(objs...)
}

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
		obj := info.Object.(*unstructured.Unstructured)
		objGVK := obj.GroupVersionKind()
		keyword := man.Keyword()
		handler := clusterMan.GetHandler()
		dCli, err := handler.Dynamic(objGVK.GroupKind(), objGVK.Version)
		if err != nil {
			log.Warningf("get %#v dynamic client error: %v", objGVK, err)
			return nil
		}
		getObj, err := dCli.Namespace(obj.GetNamespace()).Get(obj.GetName(), metav1.GetOptions{})
		// getObj, err := handler.DynamicGet(objGVK, obj.GetNamespace(), obj.GetName())
		if err != nil {
			log.Warningf("get resource %#v error: %v", objGVK, err)
			return nil
		}
		modelObj, err := model.NewK8SModelObject(man, clusterMan, getObj)
		if err != nil {
			log.Errorf("%s NewK8sModelObject error: %v", keyword, err)
			return errors.Wrapf(err, "%s NewK8SModelObject", keyword)
		}
		jsonObj, err := model.GetObject(modelObj)
		if err != nil {
			return errors.Wrapf(err, "get %s object", modelObj.Keyword())
		}
		if list, ok := ret[keyword]; ok {
			list = append(list, jsonObj)
			ret[keyword] = list
		} else {
			list = []interface{}{jsonObj}
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

func K8SObjectToJSONObject(obj runtime.Object) jsonutils.JSONObject {
	ov := reflect.ValueOf(obj)
	return ValueToJSONDict(ov)
}

func isJSONObject(input interface{}) (jsonutils.JSONObject, bool) {
	val := reflect.ValueOf(input)
	obj, ok := val.Interface().(jsonutils.JSONObject)
	if !ok {
		return nil, false
	}
	return obj, true
}

func ValueToJSONObject(out reflect.Value) jsonutils.JSONObject {
	if gotypes.IsNil(out.Interface()) {
		return nil
	}

	if obj, ok := isJSONObject(out); ok {
		return obj
	}
	jsonBytes, err := json.Marshal(out.Interface())
	if err != nil {
		panic(fmt.Sprintf("marshal json: %v", err))
	}
	jObj, err := jsonutils.Parse(jsonBytes)
	if err != nil {
		panic(fmt.Sprintf("jsonutils.Parse bytes: %s, error %v", jsonBytes, err))
	}
	return jObj
}

func ValueToJSONDict(out reflect.Value) *jsonutils.JSONDict {
	jsonObj := ValueToJSONObject(out)
	if jsonObj == nil {
		return nil
	}
	return jsonObj.(*jsonutils.JSONDict)
}

func GetSelectorByObjectMeta(meta *metav1.ObjectMeta) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: meta.GetLabels(),
	}
}

func AddObjectMetaDefaultLabel(meta *metav1.ObjectMeta) *metav1.ObjectMeta {
	return AddObjectMetaRunLabel(meta)
}

func AddObjectMetaRunLabel(meta *metav1.ObjectMeta) *metav1.ObjectMeta {
	if len(meta.Labels) == 0 {
		meta.Labels["run"] = meta.GetName()
	}
	return meta
}

func GetServicePortsByMapping(ps []api.PortMapping) []v1.ServicePort {
	ports := []v1.ServicePort{}
	for _, p := range ps {
		ports = append(ports, p.ToServicePort())
	}
	return ports
}

func GetServiceFromOption(objMeta *metav1.ObjectMeta, opt *api.ServiceCreateOption) *v1.Service {
	if opt == nil {
		return nil
	}
	svcType := opt.Type
	if svcType == "" {
		svcType = string(v1.ServiceTypeClusterIP)
	}
	if opt.IsExternal {
		svcType = string(v1.ServiceTypeLoadBalancer)
	}
	selector := opt.Selector
	if len(selector) == 0 {
		selector = GetSelectorByObjectMeta(objMeta).MatchLabels
	}
	svc := &v1.Service{
		ObjectMeta: *objMeta,
		Spec: v1.ServiceSpec{
			Selector: selector,
			Type:     v1.ServiceType(svcType),
			Ports:    GetServicePortsByMapping(opt.PortMappings),
		},
	}
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}
	if opt.LoadBalancerNetwork != "" {
		svc.Annotations[api.YUNION_LB_NETWORK_ANNOTATION] = opt.LoadBalancerNetwork
	}
	if opt.LoadBalancerCluster != "" {
		svc.Annotations[api.YUNION_LB_CLUSTER_ANNOTATION] = opt.LoadBalancerCluster
	}
	return svc
}

func CreateServiceIfNotExist(cli *client.ClusterManager, objMeta *metav1.ObjectMeta, opt *api.ServiceCreateOption) (*v1.Service, error) {
	svc, err := cli.GetHandler().GetIndexer().ServiceLister().Services(objMeta.GetNamespace()).Get(objMeta.GetName())
	if err != nil {
		if kerrors.IsNotFound(err) {
			return CreateServiceByOption(cli, objMeta, opt)
		}
		return nil, err
	}
	return svc, nil
}

func CreateServiceByOption(cli *client.ClusterManager, objMeta *metav1.ObjectMeta, opt *api.ServiceCreateOption) (*v1.Service, error) {
	svc := GetServiceFromOption(objMeta, opt)
	if svc == nil {
		return nil, nil
	}
	return CreateService(cli, svc)
}

func CreateService(cliMan *client.ClusterManager, svc *v1.Service) (*v1.Service, error) {
	cli := cliMan.GetClientset()
	return cli.CoreV1().Services(svc.GetNamespace()).Create(svc)
}

// GetContainerImages returns container image strings from the given pod spec.
func GetContainerImages(podTemplate *v1.PodSpec) []api.ContainerImage {
	containerImages := []api.ContainerImage{}
	for _, container := range podTemplate.Containers {
		containerImages = append(containerImages, api.ContainerImage{
			Name:  container.Name,
			Image: container.Image,
		})
	}
	return containerImages
}

// GetInitContainerImages returns init container image strings from the given pod spec.
func GetInitContainerImages(podTemplate *v1.PodSpec) []api.ContainerImage {
	initContainerImages := []api.ContainerImage{}
	for _, initContainer := range podTemplate.InitContainers {
		initContainerImages = append(initContainerImages, api.ContainerImage{
			Name:  initContainer.Name,
			Image: initContainer.Image})
	}
	return initContainerImages
}

type condtionSorter []*api.Condition

func (s condtionSorter) Len() int {
	return len(s)
}

func (s condtionSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s condtionSorter) Less(i, j int) bool {
	c1 := s[i]
	c2 := s[j]
	return c1.LastTransitionTime.Before(&c2.LastTransitionTime)
}

func SortConditions(conds []*api.Condition) []*api.Condition {
	sort.Sort(condtionSorter(conds))
	return conds
}

// FilterPodsByControllerResource returns a subset of pods controlled by given deployment.
func FilterDeploymentPodsByOwnerReference(deployment *apps.Deployment, allRS []*apps.ReplicaSet,
	allPods []*v1.Pod) []*v1.Pod {
	var matchingPods []*v1.Pod
	for _, rs := range allRS {
		if metav1.IsControlledBy(rs, deployment) {
			matchingPods = append(matchingPods, FilterPodsByControllerRef(rs, allPods)...)
		}
	}

	return matchingPods
}

// FilterPodsByControllerRef returns a subset of pods controlled by given controller resource, excluding deployments.
func FilterPodsByControllerRef(owner metav1.Object, allPods []*v1.Pod) []*v1.Pod {
	var matchingPods []*v1.Pod
	for _, pod := range allPods {
		if metav1.IsControlledBy(pod, owner) {
			matchingPods = append(matchingPods, pod)
		}
	}
	return matchingPods
}

func GetRawPodsByController(cli *client.ClusterManager, obj metav1.Object) ([]*v1.Pod, error) {
	pods, err := PodManager.GetRawPods(cli, obj.GetNamespace())
	if err != nil {
		return nil, err
	}
	return FilterPodsByControllerRef(obj, pods), nil
}

// getPodInfo returns aggregate information about a group of pods.
func getPodInfo(current int32, desired *int32, pods []*v1.Pod) api.PodInfo {
	result := api.PodInfo{
		Current:  current,
		Desired:  desired,
		Warnings: make([]api.Event, 0),
	}

	for _, pod := range pods {
		switch pod.Status.Phase {
		case v1.PodRunning:
			result.Running++
		case v1.PodPending:
			result.Pending++
		case v1.PodFailed:
			result.Failed++
		case v1.PodSucceeded:
			result.Succeeded++
		}
	}

	return result
}

func GetPodInfo(current int32, desired *int32, pods []*v1.Pod) (*api.PodInfo, error) {
	podInfo := getPodInfo(current, desired, pods)
	// TODO: fill warnEvents
	// warnEvents, err := EventManager.GetWarningEventsByPods(obj.GetCluster(), pods)
	return &podInfo, nil
}

// GetInternalEndpoint returns internal endpoint name for the given service properties, e.g.,
// "my-service.namespace 80/TCP" or "my-service 53/TCP,53/UDP".
func GetInternalEndpoint(serviceName, namespace string, ports []v1.ServicePort) api.Endpoint {
	name := serviceName

	if namespace != v1.NamespaceDefault && len(namespace) > 0 && len(serviceName) > 0 {
		bufferName := bytes.NewBufferString(name)
		bufferName.WriteString(".")
		bufferName.WriteString(namespace)
		name = bufferName.String()
	}

	return api.Endpoint{
		Host:  name,
		Ports: GetServicePorts(ports),
	}
}

// Returns external endpoint name for the given service properties.
func getExternalEndpoint(ingress v1.LoadBalancerIngress, ports []v1.ServicePort) api.Endpoint {
	var host string
	if ingress.Hostname != "" {
		host = ingress.Hostname
	} else {
		host = ingress.IP
	}
	return api.Endpoint{
		Host:  host,
		Ports: GetServicePorts(ports),
	}
}

// GetServicePorts returns human readable name for the given service ports list.
func GetServicePorts(apiPorts []v1.ServicePort) []api.ServicePort {
	var ports []api.ServicePort
	for _, port := range apiPorts {
		ports = append(ports, api.ServicePort{port.Port, port.Protocol, port.NodePort})
	}
	return ports
}

// GetExternalEndpoints returns endpoints that are externally reachable for a service.
func GetExternalEndpoints(service *v1.Service) []api.Endpoint {
	var externalEndpoints []api.Endpoint
	if service.Spec.Type == v1.ServiceTypeLoadBalancer {
		for _, ingress := range service.Status.LoadBalancer.Ingress {
			externalEndpoints = append(externalEndpoints, getExternalEndpoint(ingress, service.Spec.Ports))
		}
	}

	for _, ip := range service.Spec.ExternalIPs {
		externalEndpoints = append(externalEndpoints, api.Endpoint{
			Host:  ip,
			Ports: GetServicePorts(service.Spec.Ports),
		})
	}

	return externalEndpoints
}

func getPodResourceVolumes(pod *v1.Pod, predicateF func(v1.Volume) bool) []v1.Volume {
	var cfgs []v1.Volume
	vols := pod.Spec.Volumes
	for _, vol := range vols {
		if predicateF(vol) {
			cfgs = append(cfgs, vol)
		}
	}
	return cfgs
}

func GetPodSecretVolumes(pod *v1.Pod) []v1.Volume {
	return getPodResourceVolumes(pod, func(vol v1.Volume) bool {
		return vol.VolumeSource.Secret != nil
	})
}

func GetPodConfigMapVolumes(pod *v1.Pod) []v1.Volume {
	return getPodResourceVolumes(pod, func(vol v1.Volume) bool {
		return vol.VolumeSource.ConfigMap != nil
	})
}

func GetConfigMapsForPod(pod *v1.Pod, cfgs []*v1.ConfigMap) []*v1.ConfigMap {
	if len(cfgs) == 0 {
		return nil
	}
	ret := make([]*v1.ConfigMap, 0)
	uniqM := make(map[string]bool, 0)
	for _, cfg := range cfgs {
		for _, vol := range GetPodConfigMapVolumes(pod) {
			if vol.ConfigMap.Name == cfg.GetName() {
				if _, ok := uniqM[cfg.GetName()]; !ok {
					uniqM[cfg.GetName()] = true
					ret = append(ret, cfg)
				}
			}
		}
	}
	return ret
}

func GetSecretsForPod(pod *v1.Pod, ss []*v1.Secret) []*v1.Secret {
	if len(ss) == 0 {
		return nil
	}
	ret := make([]*v1.Secret, 0)
	uniqM := make(map[string]bool, 0)
	for _, s := range ss {
		for _, vol := range GetPodSecretVolumes(pod) {
			if vol.Secret.SecretName == s.GetName() {
				if _, ok := uniqM[s.GetName()]; !ok {
					uniqM[s.GetName()] = true
					ret = append(ret, s)
				}
			}
		}
	}
	return ret
}

func ValidateK8sObject(versionObj runtime.Object, internalObj interface{}, validateFunc func(internalObj interface{}) field.ErrorList) error {
	if err := legacyscheme.Scheme.Convert(versionObj, internalObj, nil); err != nil {
		return k8serrors.NewGeneralError(err)
	}
	if err := validateFunc(internalObj).ToAggregate(); err != nil {
		return httperrors.NewInputParameterError("%s", err)
	}
	return nil
}
