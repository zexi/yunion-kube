package addons

type CNICalicoConfig struct {
	ControllerImage string
	NodeImage       string
	CNIImage        string
	ClusterCIDR     string
}

func (c CNICalicoConfig) GenerateYAML() (string, error) {
	return CompileTemplateFromMap(CNICalicoTemplate, c)
}

type MetricsPluginConfig struct {
	MetricsServerImage string
}

func (c MetricsPluginConfig) GenerateYAML() (string, error) {
	return CompileTemplateFromMap(MetricsTemplate, c)
}

type HelmPluginConfig struct {
	TillerImage string
}

func (c HelmPluginConfig) GenerateYAML() (string, error) {
	return CompileTemplateFromMap(HelmTemplate, c)
}

type CloudProviderYunionConfig struct {
	CloudProviderImage string
	AuthUrl            string
	AdminUser          string
	AdminPassword      string
	AdminProject       string
	Cluster            string
	InstanceType       string
	Region             string // DEP
}

func (c CloudProviderYunionConfig) GenerateYAML() (string, error) {
	return CompileTemplateFromMap(YunionCloudProviderTemplate, c)
}
