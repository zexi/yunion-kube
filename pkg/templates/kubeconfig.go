package templates

import "encoding/base64"

const KubeConfigProxyClientT = `
apiVersion: v1
kind: Config
clusters:
- cluster:
    api-version: v1
    insecure-skip-tls-verify: true
    server: {{.KubernetesURL}}
  name: {{.ClusterName}}
contexts:
- context:
    cluster: {{.ClusterName}}
    user: {{.ComponentName}}
  name: "Default"
current-context: "Default"
users:
- name: {{.ComponentName}}
  user:
    client-certificate-data: {{.Crt}}
    client-key-data: {{.Key}}
`

const KubeConfigClientT = `
apiVersion: v1
kind: Config
clusters:
- cluster:
    api-version: v1
    certificate-authority-data: {{.Cacrt}}
    server: {{.KubernetesURL}}
  name: {{.ClusterName}}
contexts:
- context:
    cluster: {{.ClusterName}}
    user: {{.ComponentName}}
  name: "Default"
current-context: "Default"
users:
- name: {{.ComponentName}}
  user:
    client-certificate-data: {{.Crt}}
    client-key-data: {{.Key}}
`

func newKubeConfigMap(kubernetesURL, clusterName, componentName, cacrt, crt, key string) map[string]string {
	return map[string]string{
		"KubernetesURL": kubernetesURL,
		"ClusterName":   clusterName,
		"ComponentName": componentName,
		"Cacrt":         base64.StdEncoding.EncodeToString([]byte(cacrt)),
		"Crt":           base64.StdEncoding.EncodeToString([]byte(crt)),
		"Key":           base64.StdEncoding.EncodeToString([]byte(key)),
	}
}

func GetKubeConfigByProxy(kubernetesURL, clusterName, componentName, cacrt, crt, key string) (string, error) {
	return CompileTemplateFromMap(KubeConfigProxyClientT, newKubeConfigMap(kubernetesURL, clusterName, componentName, cacrt, crt, key))
}

func GetKubeConfig(kubernetesURL, clusterName, componentName, cacrt, crt, key string) (string, error) {
	return CompileTemplateFromMap(KubeConfigClientT, newKubeConfigMap(kubernetesURL, clusterName, componentName, cacrt, crt, key))
}
