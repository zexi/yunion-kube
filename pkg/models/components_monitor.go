package models

import (
	"fmt"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/embed"
	"yunion.io/x/yunion-kube/pkg/templates/components"
)

var (
	MonitorComponentManager *SMonitorComponentManager
)

const (
	MonitorNamespace   = "onecloud-monitoring"
	MonitorReleaseName = "monitor"
)

func init() {
	MonitorComponentManager = NewMonitorComponentManager()
	ComponentManager.RegisterDriver(newComponentDriverMonitor())
}

type SMonitorComponentManager struct {
	SComponentManager
	HelmComponentManager
}

type SMonitorComponent struct {
	SComponent
}

func NewMonitorComponentManager() *SMonitorComponentManager {
	man := new(SMonitorComponentManager)
	man.SComponentManager = *NewComponentManager(SMonitorComponent{},
		"kubecomponentmonitor",
		"kubecomponentmonitors")
	man.HelmComponentManager = *NewHelmComponentManager(MonitorNamespace, MonitorReleaseName, embed.MONITOR_STACK_V1_TGZ)
	man.SetVirtualObject(man)
	return man
}

type componentDriverMonitor struct {
	baseComponentDriver
}

func newComponentDriverMonitor() IComponentDriver {
	return new(componentDriverMonitor)
}

func (c componentDriverMonitor) GetType() string {
	return apis.ClusterComponentMonitor
}

func (c componentDriverMonitor) ValidateCreateData(input *apis.ComponentCreateInput) error {
	return c.validateSetting(input.Monitor)
}

func (c componentDriverMonitor) validateSetting(conf *apis.ComponentSettingMonitor) error {
	if conf == nil {
		return httperrors.NewInputParameterError("monitor config is empty")
	}
	if err := c.validateGrafana(conf.Grafana); err != nil {
		return errors.Wrap(err, "component grafana")
	}
	if err := c.validateLoki(conf.Loki); err != nil {
		return errors.Wrap(err, "component loki")
	}
	if err := c.validatePrometheus(conf.Prometheus); err != nil {
		return errors.Wrap(err, "component prometheus")
	}
	if err := c.validatePromtail(conf.Promtail); err != nil {
		return errors.Wrap(err, "component promtail")
	}
	return nil
}

func (c componentDriverMonitor) validateGrafana(conf *apis.ComponentSettingMonitorGrafana) error {
	if conf.Storage.Enabled {
		if err := c.validateStorage(conf.Storage); err != nil {
			return err
		}
	}
	return nil
}

func (c componentDriverMonitor) validateLoki(conf *apis.ComponentSettingMonitorLoki) error {
	if conf.Storage.Enabled {
		if err := c.validateStorage(conf.Storage); err != nil {
			return err
		}
	}
	return nil
}

func (c componentDriverMonitor) validateStorage(storage *apis.ComponentStorage) error {
	if storage == nil {
		return httperrors.NewInputParameterError("storage config is empty")
	}
	if storage.SizeMB < 1024 {
		return httperrors.NewNotAcceptableError("storage size must large than 1 GB")
	}
	return nil
}

func (c componentDriverMonitor) validatePrometheus(conf *apis.ComponentSettingMonitorPrometheus) error {
	if conf == nil {
		return httperrors.NewInputParameterError("config is empty")
	}
	if conf.Storage.Enabled {
		if err := c.validateStorage(conf.Storage); err != nil {
			return err
		}
	}
	return nil
}

func (c componentDriverMonitor) validatePromtail(conf *apis.ComponentSettingMonitorPromtail) error {
	// TODO
	return nil
}

func (c componentDriverMonitor) ValidateUpdateData(input *apis.ComponentUpdateInput) error {
	return c.validateSetting(input.Monitor)
}

func (c componentDriverMonitor) GetCreateSettings(input *apis.ComponentCreateInput) (*apis.ComponentSettings, error) {
	if input.ComponentSettings.Namespace == "" {
		input.ComponentSettings.Namespace = MonitorNamespace
	}
	return &input.ComponentSettings, nil
}

func (c componentDriverMonitor) GetUpdateSettings(oldSetting *apis.ComponentSettings, input *apis.ComponentUpdateInput) (*apis.ComponentSettings, error) {
	oldSetting.Monitor = input.Monitor
	return oldSetting, nil
}

func (c componentDriverMonitor) DoEnable(cluster *SCluster, setting *apis.ComponentSettings) error {
	return MonitorComponentManager.CreateHelmResource(cluster, setting)
}

func (c componentDriverMonitor) DoDisable(cluster *SCluster, setting *apis.ComponentSettings) error {
	return MonitorComponentManager.DeleteHelmResource(cluster, setting)
}

func (c componentDriverMonitor) DoUpdate(cluster *SCluster, setting *apis.ComponentSettings) error {
	return MonitorComponentManager.UpdateHelmResource(cluster, setting)
}

func (c componentDriverMonitor) FetchStatus(cluster *SCluster, comp *SComponent, status *apis.ComponentsStatus) error {
	if status.Monitor == nil {
		status.Monitor = new(apis.ComponentStatusMonitor)
	}
	c.InitStatus(comp, &status.Monitor.ComponentStatus)
	return nil
}

func (m SMonitorComponentManager) getHelmValues(cluster *SCluster, setting *apis.ComponentSettings) (map[string]interface{}, error) {
	imgRepo, err := cluster.GetImageRepository()
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster %s repo", cluster.GetName())
	}
	repo := imgRepo.Url
	mi := func(name, tag string) components.Image {
		return components.Image{
			Repository: fmt.Sprintf("%s/%s", repo, name),
			Tag:        tag,
		}
	}
	conf := components.MonitorStack{
		Image:                         mi("prometheus-operator", "v0.37.0"),
		ConfigmapReloadImage:          mi("configmap-reload", "v0.0.1"),
		PrometheusConfigReloaderImage: mi("prometheus-config-reloader", "v0.37.0"),
		TLSProxy: components.PromTLSProxy{
			Image: mi("ghostunnel", "v1.5.2"),
		},
		Prometheus: components.Prometheus{
			Spec: components.PrometheusSpec{
				Image: mi("prometheus", "v2.15.2"),
			},
		},
		Alertmanager: components.Alertmanager{
			Spec: components.AlertmanagerSpec{
				Image: mi("alertmanager", "v0.20.0"),
			},
		},
		PrometheusNodeExporter: components.PrometheusNodeExporter{
			Image: mi("node-exporter", "v0.18.1"),
		},
		KubeStateMetrics: components.KubeStateMetrics{
			Image: mi("kube-state-metrics", "v1.9.4"),
		},
		Grafana: components.Grafana{
			Sidecar: components.GrafanaSidecar{
				Image: mi("k8s-sidecar", "0.1.99"),
			},
			Image: mi("grafana", "6.7.1"),
		},
		Loki: components.Loki{
			Image: mi("loki", "1.4.1"),
		},
		Promtail: components.Promtail{
			Image: mi("promtail", "1.4.1"),
		},
	}
	input := setting.Monitor
	if input.Prometheus.Storage != nil && input.Prometheus.Storage.Enabled {
		spec, err := components.NewPrometheusStorageSpec(*input.Prometheus.Storage)
		if err != nil {
			return nil, errors.Wrap(err, "prometheus storage spec")
		}
		conf.Prometheus.Spec.StorageSpec = spec
	}
	if input.Grafana.Storage != nil && input.Grafana.Storage.Enabled {
		spec, err := components.NewPVCStorage(input.Grafana.Storage)
		if err != nil {
			return nil, errors.Wrap(err, "grafana storage spec")
		}
		conf.Grafana.Storage = spec
	}
	if input.Loki.Storage != nil && input.Loki.Storage.Enabled {
		spec, err := components.NewPVCStorage(input.Loki.Storage)
		if err != nil {
			return nil, errors.Wrap(err, "loki storage")
		}
		conf.Loki.Storage = spec
	}
	return components.GenerateHelmValues(conf), nil
}

func (m SMonitorComponentManager) CreateHelmResource(cluster *SCluster, setting *apis.ComponentSettings) error {
	vals, err := m.getHelmValues(cluster, setting)
	if err != nil {
		return errors.Wrap(err, "get helm config values")
	}
	return m.HelmComponentManager.CreateHelmResource(cluster, vals)
}

func (m SMonitorComponentManager) DeleteHelmResource(cluster *SCluster, setting *apis.ComponentSettings) error {
	return m.HelmComponentManager.DeleteHelmResource(cluster)
}

func (m SMonitorComponentManager) UpdateHelmResource(cluster *SCluster, setting *apis.ComponentSettings) error {
	vals, err := m.getHelmValues(cluster, setting)
	if err != nil {
		return errors.Wrap(err, "get helm config values")
	}
	return m.HelmComponentManager.UpdateHelmResource(cluster, vals)
}
