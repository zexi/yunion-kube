package kubeadm

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kubeadmv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	"k8s.io/kubernetes/cmd/kubeadm/app/util"
	"sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

// GetCodecs returns a type that can be used to deserialize most kubeadm
// configuration types.
func GetCodecs() serializer.CodecFactory {
	sb := &scheme.Builder{GroupVersion: kubeadmv1beta1.SchemeGroupVersion}

	sb.Register(&kubeadmv1beta1.JoinConfiguration{}, &kubeadmv1beta1.InitConfiguration{}, &kubeadmv1beta1.ClusterConfiguration{})
	kubeadmScheme, err := sb.Build()
	if err != nil {
		panic(err)
	}
	return serializer.NewCodecFactory(kubeadmScheme)
}

// ConfigurationToYAML converts a kubeadm configuration type to its YAML
// representation.
func ConfigurationToYAML(obj runtime.Object) (string, error) {
	initcfg, err := util.MarshalToYamlForCodecs(obj, kubeadmv1beta1.SchemeGroupVersion, GetCodecs())
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal init configuration")
	}
	return string(initcfg), nil
}
