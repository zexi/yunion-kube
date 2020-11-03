package api

import "yunion.io/x/onecloud/pkg/apis"

const (
	ClusterComponentCephCSI   = "cephCSI"
	ClusterComponentMonitor   = "monitor"
	ClusterComponentFluentBit = "fluentbit"
)

const (
	ComponentStatusDeploying    = "deploying"
	ComponentStatusDeployFail   = "deploy_fail"
	ComponentStatusDeployed     = "deployed"
	ComponentStatusDeleting     = "deleting"
	ComponentStatusUndeploying  = "undeploying"
	ComponentStatusUndeployFail = "undeploy_fail"
	ComponentStatusDeleteFail   = "delete_fail"
	ComponentStatusUpdating     = "updating"
	ComponentStatusUpdateFail   = "update_fail"
	ComponentStatusInit         = "init"
)

type ComponentCreateInput struct {
	apis.Meta

	Name string `json:"name"`
	Type string `json:"type"`

	Cluster string `json:"cluster"`
	ComponentSettings
}

type ComponentDeleteInput struct {
	apis.Meta

	Name string `json:"name"`
	Type string `json:"type"`
}

type ComponentSettings struct {
	Namespace string `json:"namespace"`
	// Ceph CSI 组件配置
	CephCSI *ComponentSettingCephCSI `json:"cephCSI"`
	// Monitor stack 组件配置
	Monitor *ComponentSettingMonitor `json:"monitor"`
	// Fluentbit 日志收集 agent 配置
	FluentBit *ComponentSettingFluentBit `json:"fluentbit"`
}

type ComponentCephCSIConfigCluster struct {
	// 集群 Id
	// required: true
	// example: office-ceph-cluster
	ClsuterId string `json:"clusterId"`
	// ceph monitor 连接地址, 比如: 192.168.222.12:6239
	// required: true
	// example: ["192.168.222.12:6239", "192.168.222.13:6239", "192.168.222.14:6239"]
	Monitors []string `json:"monitors"`
}

type ComponentSettingCephCSI struct {
	// 集群配置
	// required: true
	Config []ComponentCephCSIConfigCluster `json:"config"`
}

type ComponentStorage struct {
	// 是否启用持久化存储
	Enabled bool `json:"enabled"`
	// 存储大小, 单位 MB
	SizeMB int `json:"sizeMB"`
	// storageClass 名称
	//
	// required: true
	ClassName string `json:"storageClassName"`
}

func (s ComponentStorage) GetAccessModes() []string {
	return []string{"ReadWriteOnce"}
}

type IngressTLS struct {
	SecretName string `json:"secretName"`
}

type TLSKeyPair struct {
	Name        string `json:"name"`
	Certificate string `json:"certificate"`
	Key         string `json:"key"`
}

type ComponentSettingMonitorGrafana struct {
	// grafana 登录用户名
	// default: admin
	AdminUser string `json:"adminUser"`

	// grafana 登录用户密码
	// default: prom-operator
	AdminPassword string `json:"adminPassword"`
	// grafana 持久化存储配置
	Storage *ComponentStorage `json:"storage"`
	// grafana ingress public address
	PublicAddress string `json:"publicAddress"`
	// grafana ingress host
	Host string `json:"host"`
	// Ingress expose https key pair
	TLSKeyPair *TLSKeyPair `json:"tlsKeyPair"`
}

type ComponentSettingMonitorLoki struct {
	// loki 持久化存储配置
	Storage *ComponentStorage `json:"storage"`
}

type ComponentSettingMonitorPrometheus struct {
	// prometheus 持久化存储配置
	Storage *ComponentStorage `json:"storage"`
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
	// Grafana 前端日志、监控展示服务
	//
	// required: true
	Grafana *ComponentSettingMonitorGrafana `json:"grafana"`
	// Loki 后端日志收集服务
	//
	// required: true
	Loki *ComponentSettingMonitorLoki `json:"loki"`
	// Prometheus 监控数据采集服务
	//
	// required: true
	Prometheus *ComponentSettingMonitorPrometheus `json:"prometheus"`
	// Promtail 日志收集 agent
	//
	// required: false
	Promtail *ComponentSettingMonitorPromtail `json:"promtail"`
}

type ComponentSettingFluentBitBackendTLS struct {
	// 是否开启 TLS 连接
	//
	// required: false
	TLS bool `json:"tls"`

	// 是否开启 TLS 教研
	//
	// required: false
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
	// 是否启用该后端
	// required: true
	Enabled bool `json:"enabled"`
}

type ComponentSettingFluentBitBackendES struct {
	ComponentSettingFluentBitBackendCommon
	// Elastic 集群连接地址
	//
	// required: true
	// example: 10.168.26.182
	Host string `json:"host"`

	// Elastic 集群连接地址
	//
	// required: true
	// default: 9200
	// example: 9200
	Port int `json:"port"`

	// Elastic index 名称
	//
	// required: true
	// default: fluentbit
	Index string `json:"index"`

	// 类型
	//
	// required: true
	// default: flb_type
	Type string `json:"type"`

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
	// 上报数据格式
	//
	// required: false
	// default: json
	// example: json|msgpack
	Format string `json:"format"`
	// Optional key to store the message
	MessageKey string `json:"messageKey"`
	// Set the key to store the record timestamp
	TimestampKey string `json:"timestampKey"`
	// kafka broker 地址
	//
	// required: true
	// example: ["192.168.222.10:9092", "192.168.222.11:9092", "192.168.222.13:9092"]
	Brokers []string `json:"brokers"`
	// kafka topic
	//
	// required: true
	// example: ["fluent-bit"]
	Topics []string `json:"topics"`
}

const (
	ComponentSettingFluentBitBackendTypeES    = "es"
	ComponentSettingFluentBitBackendTypeKafka = "kafka"
)

type ComponentSettingFluentBitBackend struct {
	// Elasticsearch 配置
	ES *ComponentSettingFluentBitBackendES `json:"es"`
	// Kafka 配置
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
