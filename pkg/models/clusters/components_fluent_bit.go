package clusters

import (
	"fmt"
	"strings"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/embed"
	"yunion.io/x/yunion-kube/pkg/templates/components"
)

var (
	FluentBitComponentManager *SFluentBitComponentManager
)

const (
	FluentBitReleaseName = "fluentbit"
)

func init() {
	FluentBitComponentManager = NewFluentBitComponentManager()
	ComponentManager.RegisterDriver(newComponentDriverFluentBit())
}

type SFluentBitComponentManager struct {
	SComponentManager
	HelmComponentManager
}

type SFluentBitComponent struct {
	SComponent
}

func NewFluentBitComponentManager() *SFluentBitComponentManager {
	man := new(SFluentBitComponentManager)
	man.SComponentManager = *NewComponentManager(SFluentBitComponent{},
		"kubecomponentfluentbit",
		"kubecomponentfluentbits")
	man.HelmComponentManager = *NewHelmComponentManager(MonitorNamespace, FluentBitReleaseName, embed.FLUENT_BIT_2_8_12_TGZ)
	man.SetVirtualObject(man)
	return man
}

type componentDriverFluentBit struct {
	baseComponentDriver
}

func newComponentDriverFluentBit() IComponentDriver {
	return new(componentDriverFluentBit)
}

func (c componentDriverFluentBit) GetType() string {
	return apis.ClusterComponentFluentBit
}

func (c componentDriverFluentBit) ValidateCreateData(input *apis.ComponentCreateInput) error {
	return nil
}

func (c componentDriverFluentBit) ValidateUpdateData(input *apis.ComponentUpdateInput) error {
	return nil
}

func (c componentDriverFluentBit) GetCreateSettings(input *apis.ComponentCreateInput) (*apis.ComponentSettings, error) {
	if input.ComponentSettings.Namespace == "" {
		input.ComponentSettings.Namespace = MonitorNamespace
	}
	return &input.ComponentSettings, nil
}

func (c componentDriverFluentBit) GetUpdateSettings(oldSetting *apis.ComponentSettings, input *apis.ComponentUpdateInput) (*apis.ComponentSettings, error) {
	oldSetting.FluentBit = input.FluentBit
	return oldSetting, nil
}

func (c componentDriverFluentBit) DoEnable(cluster *SCluster, setting *apis.ComponentSettings) error {
	return FluentBitComponentManager.CreateHelmResource(cluster, setting)
}

func (c componentDriverFluentBit) DoDisable(cluster *SCluster, setting *apis.ComponentSettings) error {
	return FluentBitComponentManager.DeleteHelmResource(cluster, setting)
}

func (c componentDriverFluentBit) DoUpdate(cluster *SCluster, setting *apis.ComponentSettings) error {
	return FluentBitComponentManager.UpdateHelmResource(cluster, setting)
}

func (c componentDriverFluentBit) FetchStatus(cluster *SCluster, comp *SComponent, status *apis.ComponentsStatus) error {
	if status.FluentBit == nil {
		status.FluentBit = new(apis.ComponentStatusFluentBit)
	}
	c.InitStatus(comp, &status.FluentBit.ComponentStatus)
	return nil
}

func (m SFluentBitComponentManager) getHelmValues(cluster *SCluster, settings *apis.ComponentSettings) (map[string]interface{}, error) {
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
	setting := settings.FluentBit
	conf := components.FluentBit{
		Image: components.FluentBitImage{
			FluentBit: mi("fluent-bit", "1.3.7"),
		},
	}
	be := setting.Backend
	// set backend
	if be.ES != nil && be.ES.Enabled {
		esConf := be.ES
		conf.Backend.ES = &components.FluentBitBackendES{
			FluentBitBackendCommon: components.FluentBitBackendCommon{
				Enabled: true,
			},
			Host:           esConf.Host,
			Port:           esConf.Port,
			Index:          esConf.Index,
			Type:           esConf.Type,
			LogstashPrefix: esConf.LogstashPrefix,
			HTTPUser:       esConf.HTTPUser,
			HTTPPassword:   esConf.HTTPPassword,
			FluentBitBackendTLS: components.FluentBitBackendTLS{
				TLS:   esConf.TLS,
				TLSCA: esConf.TLSCA,
			},
		}
		if esConf.ReplaceDots {
			conf.Backend.ES.ReplaceDots = "On"
		}
		if esConf.LogstashFormat {
			conf.Backend.ES.LogstashFormat = "On"
		}
		if esConf.TLSVerify {
			conf.Backend.ES.TLSVerify = "On"
		}
	}
	if be.Kafka != nil && be.Kafka.Enabled {
		kConf := setting.Backend.Kafka
		conf.Backend.Kafka = &components.FluentBitBackendKafka{
			FluentBitBackendCommon: components.FluentBitBackendCommon{
				Enabled: true,
			},
			Format:       kConf.Format,
			MessageKey:   kConf.MessageKey,
			TimestampKey: kConf.TimestampKey,
			Brokers:      strings.Join(kConf.Brokers, ","),
			Topics:       strings.Join(kConf.Topics, ","),
		}
	}
	return components.GenerateHelmValues(conf), nil
}

func (m SFluentBitComponentManager) CreateHelmResource(cluster *SCluster, setting *apis.ComponentSettings) error {
	vals, err := m.getHelmValues(cluster, setting)
	if err != nil {
		return errors.Wrap(err, "get helm config values")
	}
	return m.HelmComponentManager.CreateHelmResource(cluster, vals)
}

func (m SFluentBitComponentManager) DeleteHelmResource(cluster *SCluster, setting *apis.ComponentSettings) error {
	return m.HelmComponentManager.DeleteHelmResource(cluster)
}

func (m SFluentBitComponentManager) UpdateHelmResource(cluster *SCluster, setting *apis.ComponentSettings) error {
	vals, err := m.getHelmValues(cluster, setting)
	if err != nil {
		return errors.Wrap(err, "get helm config values")
	}
	return m.HelmComponentManager.UpdateHelmResource(cluster, vals)
}
