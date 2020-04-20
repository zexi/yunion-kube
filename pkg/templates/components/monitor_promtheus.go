package components

import (
	"sigs.k8s.io/yaml"

	"yunion.io/x/jsonutils"
)

type Image struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
}

type PromTLSProxy struct {
	Enabled bool  `json:"enabled"`
	Image   Image `json:"image"`
}

type PromAdmissionWebhooksPatch struct {
	Enabled bool `json:"enabled"`
}

type PromAdmissionWebhooks struct {
	Enabled bool                       `json:"enabled"`
	Patch   PromAdmissionWebhooksPatch `json:"patch"`
}

type Prometheus struct {
	Spec PrometheusSpec `json:"prometheusSpec"`
}

type PrometheusSpec struct {
	// image: quay.io/prometheus/prometheus:v2.15.2
	Image Image `json:"image"`
}

type Alertmanager struct {
	Spec AlertmanagerSpec `json:"alertmanagerSpec"`
}

type AlertmanagerSpec struct {
	// image: quay.io/prometheus/alertmanager:v0.20.0
	Image Image `json:"image"`
}

type PrometheusNodeExporter struct {
	// image: quay.io/prometheus/node-exporter:v0.18.1
	Image Image `json:"image"`
}

type KubeStateMetrics struct {
	// image: quay.io/coreos/kube-state-metrics:v1.9.4
	Image Image `json:"image"`
}

type GrafanaSidecar struct {
	// image: kiwigrid/k8s-sidecar:0.1.99
	Image Image `json:"image"`
}

type Grafana struct {
	Sidecar GrafanaSidecar `json:"sidecar"`
	// image: grafana/grafana:6.7.1
	Image Image `json:"image"`
}

type Loki struct {
	Image Image `json:"image"`
}

type Promtail struct {
	BusyboxImage string `json:"busyboxImage"`
	Image        Image  `json:"image"`
}

type AdmissionWebhooksPatch struct {
	Enabled bool `json:"enabled"`
	// image: jettech/kube-webhook-certgen:v1.0.0
	Image Image `json:"image"`
}

type AdmissionWebhooks struct {
	Enabled bool                   `json:"enabled"`
	Patch   AdmissionWebhooksPatch `json:"patch"`
}

type PrometheusOperator struct {
	AdmissionWebhooks AdmissionWebhooks `json:"admissionWebhooks"`
}

type MonitorStack struct {
	// image: quay.io/coreos/prometheus-operator:v0.37.0
	Image Image `json:"image"`
	// image: quay.io/coreos/prometheus-config-reloader:v0.37.0
	ConfigmapReloadImage Image `json:"configmapReloadImage"`
	// image: quay.io/coreos/configmap-reload:v0.0.1
	PrometheusConfigReloaderImage Image `json:"prometheusConfigReloaderImage"`
	// image: squareup/ghostunnel:v1.5.2
	TLSProxy               PromTLSProxy           `json:"tlsProxy"`
	Prometheus             Prometheus             `json:"prometheus"`
	Alertmanager           Alertmanager           `json:"alertmanager"`
	PrometheusNodeExporter PrometheusNodeExporter `json:"prometheus-node-exporter"`
	KubeStateMetrics       KubeStateMetrics       `json:"kube-state-metrics"`
	Grafana                Grafana                `json:"grafana"`
	Loki                   Loki                   `json:"loki"`
	Promtail               Promtail               `json:"promtail"`
}

func GenerateHelmValues(config interface{}) map[string]interface{} {
	yamlStr := jsonutils.Marshal(config).YAMLString()
	vals := map[string]interface{}{}
	yaml.Unmarshal([]byte(yamlStr), &vals)
	return vals
}
