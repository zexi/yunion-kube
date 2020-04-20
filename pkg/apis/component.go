package apis

import "yunion.io/x/onecloud/pkg/apis"

const (
	ClusterComponentCephCSI   = "cephCSI"
	ClusterComponentMonitor   = "monitor"
	ClusterComponentFluentBit = "fluentbit"
)

const (
	ComponentStatusDeploying  = "deploying"
	ComponentStatusDeployFail = "deploy_fail"
	ComponentStatusDeployed   = "deployed"
	ComponentStatusDeleting   = "deleting"
	ComponentStatusDeleteFail = "delete_fail"
	ComponentStatusUpdating   = "updating"
	ComponentStatusUpdateFail = "update_fail"
	ComponentStatusInit       = "init"
)

type ComponentCreateInput struct {
	apis.Meta

	Name string `json:"name"`
	Type string `json:"type"`

	ComponentSettings
}

type ComponentSettings struct {
	Namespace string                     `json:"namespace"`
	CephCSI   *ComponentSettingCephCSI   `json:"cephCSI"`
	Monitor   *ComponentSettingMonitor   `json:"monitor"`
	FluentBit *ComponentSettingFluentBit `json:"fluentbit"`
}

type ComponentCephCSIConfigCluster struct {
	ClsuterId string   `json:"clusterId"`
	Monitors  []string `json:"monitors"`
}

type ComponentSettingCephCSI struct {
	Config []ComponentCephCSIConfigCluster `json:"config"`
}

type ComponentSettingMonitorGrafana struct {
	AdminUser     string `json:"adminUser"`
	AdminPassword string `json:"adminPassword"`
}

type ComponentSettingMonitorPrometheus struct {
	StorageSize string `json:"storageSize"`
}

type ComponentSettingVolume struct {
	HostPath  string `json:"hostPath"`
	MountPath string `json:"mountPath"`
}

type ComponentSettingMonitorPromtail struct {
	DockerVolumeMount ComponentSettingVolume `json:"dockerVolumeMount"`
	PodsVolumeMount   ComponentSettingVolume `json:"podsVolumeMount"`
}

type ComponentSettingMonitor struct {
	Grafana    ComponentSettingMonitorGrafana    `json:"grafana"`
	Prometheus ComponentSettingMonitorPrometheus `json:"prometheus"`
	Promtail   ComponentSettingMonitorPromtail   `json:"promtail"`
}

type ComponentSettingFluentBitBackendTLS struct {
	TLS string `json:"tls"`
	// "off" or "on"
	TLSVerify bool   `json:"tlsVerify"`
	TLSDebug  bool   `json:"tlsDebug"`
	TLSCA     string `json:"tlsCA"`
}

type ComponentSettingFluentBitBackendForward struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	ComponentSettingFluentBitBackendTLS
}

type ComponentSettingFluentBitBackendCommon struct {
	Enabled bool `json:"enabled"`
}

type ComponentSettingFluentBitBackendES struct {
	ComponentSettingFluentBitBackendCommon
	Host string `json:"host"`
	Port int    `json:"port"`
	// Elastic index name, default: fluentbit
	Index string `json:"index"`
	// Type name, default: flb_type
	Type           string `json:"type"`
	LogstashPrefix string `json:"logstashPrefix"`
	LogstashFormat bool   `json:"logstashFormat"`
	ReplaceDots    bool   `json:"replaceDots"`
	// Optional username credential for Elastic X-Pack access
	HTTPUser string `json:"httpUser"`
	// Password for user defined in HTTPUser
	HTTPPassword string `json:"httpPassword"`
	ComponentSettingFluentBitBackendTLS
}

// check: https://fluentbit.io/documentation/0.14/output/kafka.html
type ComponentSettingFluentBitBackendKafka struct {
	ComponentSettingFluentBitBackendCommon
	// specify data format, options available: json, msgpack, default: json
	Format string `json:"format"`
	// Optional key to store the message
	MessageKey string `json:"messageKey"`
	// Set the key to store the record timestamp
	TimestampKey string `json:"timestampKey"`
	// Single of multiple list of kafka brokers
	Brokers []string `json:"brokers"`
	// Single entry or list of topics separated by comma(,)
	Topics []string `json:"topics"`
}

const (
	ComponentSettingFluentBitBackendTypeES    = "es"
	ComponentSettingFluentBitBackendTypeKafka = "kafka"
)

type ComponentSettingFluentBitBackend struct {
	//Forward *ComponentSettingFluentBitBackendForward `json:"forward"`
	ES    *ComponentSettingFluentBitBackendES    `json:"es"`
	Kafka *ComponentSettingFluentBitBackendKafka `json:"kafka"`
}

type ComponentSettingFluentBit struct {
	Backend *ComponentSettingFluentBitBackend `json:"backend"`
}

type ComponentsStatus struct {
	apis.Meta

	CephCSI   *ComponentStatusCephCSI   `json:"cephCSI"`
	Monitor   *ComponentStatusMonitor   `json:"monitor"`
	FluentBit *ComponentStatusFluentBit `json:"fluentbit"`
}

type ComponentStatus struct {
	Id      string `json:"id"`
	Created bool   `json:"created"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
}

type ComponentStatusCephCSI struct {
	ComponentStatus
}

type ComponentStatusMonitor struct {
	ComponentStatus
}

type ComponentStatusFluentBit struct {
	ComponentStatus
}

type ComponentUpdateInput struct {
	apis.Meta

	Type string `json:"type"`

	ComponentSettings
}
