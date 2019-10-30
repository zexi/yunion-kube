package common

import (
	"encoding/json"
	"strings"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"

	api "yunion.io/x/yunion-kube/pkg/apis"
)

func JsonDecode(data jsonutils.JSONObject, obj interface{}) error {
	dataStr, err := data.GetString()
	if err != nil {
		return err
	}
	err = json.NewDecoder(strings.NewReader(dataStr)).Decode(obj)
	return err
}

func GetK8sObjectCreateMetaByRequest(req *Request) (*metav1.ObjectMeta, error) {
	objMeta, err := GetK8sObjectCreateMeta(req.Data)
	if err != nil {
		return nil, err
	}
	ns := req.GetDefaultNamespace()
	objMeta.Namespace = ns
	return objMeta, nil
}

func GetK8sObjectCreateMeta(data jsonutils.JSONObject) (*metav1.ObjectMeta, error) {
	name, err := data.GetString("name")
	if err != nil {
		return nil, httperrors.NewInputParameterError("name not provided")
	}

	labels := make(map[string]string)
	annotations := make(map[string]string)

	data.Unmarshal(&labels, "labels")
	data.Unmarshal(&annotations, "annotations")
	return &metav1.ObjectMeta{
		Name:        name,
		Labels:      labels,
		Annotations: annotations,
	}, nil
}

func GenerateName(base string) string {
	return api.GenerateName(base)
}

func ToConfigMap(configMap *v1.ConfigMap, cluster api.ICluster) api.ConfigMap {
	return api.ConfigMap{
		ObjectMeta: api.NewObjectMeta(configMap.ObjectMeta, cluster),
		TypeMeta:   api.NewTypeMeta(configMap.TypeMeta),
	}
}

func ToConfigMaps(cfgs []*v1.ConfigMap, cluster api.ICluster) []api.ConfigMap {
	ret := make([]api.ConfigMap, 0)
	for _, c := range cfgs {
		ret = append(ret, ToConfigMap(c, cluster))
	}
	return ret
}

func ToSecret(secret *v1.Secret, cluster api.ICluster) *api.Secret {
	return &api.Secret{
		ObjectMeta: api.NewObjectMeta(secret.ObjectMeta, cluster),
		TypeMeta:   api.NewTypeMeta(secret.TypeMeta),
		Type:       secret.Type,
	}
}

func ToSecrets(ss []*v1.Secret, cluster api.ICluster) []api.Secret {
	ret := make([]api.Secret, 0)
	for _, s := range ss {
		ret = append(ret, *ToSecret(s, cluster))
	}
	return ret
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
	for _, cfg := range cfgs {
		for _, vol := range GetPodConfigMapVolumes(pod) {
			if vol.ConfigMap.Name == cfg.GetName() {
				ret = append(ret, cfg)
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
	for _, s := range ss {
		for _, vol := range GetPodSecretVolumes(pod) {
			if vol.Secret.SecretName == s.GetName() {
				ret = append(ret, s)
			}
		}
	}
	return ret
}
