/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package userdata

import "github.com/pkg/errors"

const (
	controlPlaneBashScript = `{{.Header}}

{{.DockerScript}}

{{.OnecloudConfig}}

mkdir -p /etc/kubernetes/pki/etcd

echo '{{.CACert}}' > /etc/kubernetes/pki/ca.crt
echo '{{.CAKey}}' > /etc/kubernetes/pki/ca.key
echo '{{.EtcdCACert}}' > /etc/kubernetes/pki/etcd/ca.crt
echo '{{.EtcdCAKey}}' > /etc/kubernetes/pki/etcd/ca.key
echo '{{.FrontProxyCACert}}' > /etc/kubernetes/pki/front-proxy-ca.crt
echo '{{.FrontProxyCAKey}}' > /etc/kubernetes/pki/front-proxy-ca.key
echo '{{.SaCert}}' > /etc/kubernetes/pki/sa.pub
echo '{{.SaKey}}' > /etc/kubernetes/pki/sa.key

cat >/tmp/kubeadm.yaml <<EOF
---
apiVersion: kubeadm.k8s.io/v1beta1
kind: ClusterConfiguration
apiServer:
  certSANs:
    - "{{.PrivateIP}}"
    - "{{.ELBAddress}}"
  extraArgs:
    cloud-provider: external
    feature-gates: "CSIPersistentVolume=true,MountPropagation=true"
    runtime-config: "storage.k8s.io/v1alpha1=true,admissionregistration.k8s.io/v1alpha1=true,settings.k8s.io/v1alpha1=true"
controllerManager:
  extraArgs:
    cloud-provider: external
    feature-gates: "CSIPersistentVolume=true,MountPropagation=true"
scheduler:
  extraArgs:
    feature-gates: "CSIPersistentVolume=true,MountPropagation=true"
controlPlaneEndpoint: "{{.ELBAddress}}:6443"
imageRepository: "registry.cn-beijing.aliyuncs.com/yunionio"
clusterName: "{{.ClusterName}}"
networking:
  dnsDomain: "{{.ServiceDomain}}"
  podSubnet: "{{.PodSubnet}}"
  serviceSubnet: "{{.ServiceSubnet}}"
kubernetesVersion: "{{.KubernetesVersion}}"
---
apiVersion: kubeadm.k8s.io/v1beta1
kind: InitConfiguration
nodeRegistration:
  name: $(hostname)
  #criSocket: /var/run/containerd/containerd.sock
  kubeletExtraArgs:
    cloud-provider: external
    read-only-port: "10255"
    pod-infra-container-image: registry.cn-beijing.aliyuncs.com/yunionio/pause-amd64:3.1
    feature-gates: "CSIPersistentVolume=true,MountPropagation=true,KubeletPluginsWatcher=true,VolumeScheduling=true"
    eviction-hard: "memory.available<100Mi,nodefs.available<2Gi,nodefs.inodesFree<5%"
---
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
mode: ipvs
clusterCIDR: {{.ServiceSubnet}}
EOF

kubeadm init --config /tmp/kubeadm.yaml
systemctl enable kubelet
`

	controlPlaneJoinBashScript = `{{.Header}}

{{.DockerScript}}

{{.OnecloudConfig}}

mkdir -p /etc/kubernetes/pki/etcd

echo '{{.CACert}}' > /etc/kubernetes/pki/ca.crt
echo '{{.CAKey}}' > /etc/kubernetes/pki/ca.key
echo '{{.EtcdCACert}}' > /etc/kubernetes/pki/etcd/ca.crt
echo '{{.EtcdCAKey}}' > /etc/kubernetes/pki/etcd/ca.key
echo '{{.FrontProxyCACert}}' > /etc/kubernetes/pki/front-proxy-ca.crt
echo '{{.FrontProxyCAKey}}' > /etc/kubernetes/pki/front-proxy-ca.key
echo '{{.SaCert}}' > /etc/kubernetes/pki/sa.pub
echo '{{.SaKey}}' > /etc/kubernetes/pki/sa.key

#echo 'KubeConfig' > /etc/kubernetes/admin.conf

cat >/tmp/kubeadm-controlplane-join-config.yaml <<EOF
---
apiVersion: kubeadm.k8s.io/v1beta1
kind: JoinConfiguration
discovery:
  bootstrapToken:
    token: "{{.BootstrapToken}}"
    apiServerEndpoint: "{{.ELBAddress}}:6443"
    caCertHashes:
      - "{{.CACertHash}}"
nodeRegistration:
  name: "$(hostname)"
  kubeletExtraArgs:
    cloud-provider: external
controlPlane:
  localAPIEndpoint:
    advertiseAddress: "{{.PrivateIP}}"
    bindPort: 6443
EOF

kubeadm join --config /tmp/kubeadm-controlplane-join-config.yaml --v 10
systemctl enable kubelet
`
)

// ControlPlaneInput defines the context to generate a controlplane instance user data.
type ControlPlaneInput struct {
	*baseUserData

	BaseConfigure

	CACert            string
	CAKey             string
	EtcdCACert        string
	EtcdCAKey         string
	FrontProxyCACert  string
	FrontProxyCAKey   string
	SaCert            string
	SaKey             string
	ELBAddress        string
	ClusterName       string
	PodSubnet         string
	ServiceDomain     string
	ServiceSubnet     string
	KubernetesVersion string
	Hostname          string
	PrivateIP         string
}

// ContolPlaneJoinInput defines context to generate controlplane instance user data for controlplane node join.
type ControlPlaneJoinInput struct {
	*baseUserData

	BaseConfigure

	CACertHash       string
	CACert           string
	CAKey            string
	EtcdCACert       string
	EtcdCAKey        string
	FrontProxyCACert string
	FrontProxyCAKey  string
	SaCert           string
	SaKey            string
	BootstrapToken   string
	ELBAddress       string
	PrivateIP        string
}

// NewControlPlane returns the user data string to be used on a controlplane instance.
func NewControlPlane(input *ControlPlaneInput) (string, error) {
	var err error
	input.baseUserData, err = newBaseUserData(input.BaseConfigure)
	if err != nil {
		return "", err
	}
	userData, err := generate("controlplane", controlPlaneBashScript, input)
	if err != nil {
		return "", errors.Wrapf(err, "failed to generate user data for new control plane machine")
	}
	return userData, err
}

// JoinControlPlane returns the user data string to be used on a new contrplplane instance.
func JoinControlPlane(input *ControlPlaneJoinInput) (string, error) {
	var err error
	input.baseUserData, err = newBaseUserData(input.BaseConfigure)
	if err != nil {
		return "", err
	}

	userData, err := generate("controlplane", controlPlaneJoinBashScript, input)
	if err != nil {
		return "", errors.Wrapf(err, "failed to generate user data for machine joining control plane")
	}
	return userData, err
}
