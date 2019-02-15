package addons

import (
	"bytes"
	"text/template"
)

func CompileTemplateFromMap(tmplt string, configMap interface{}) (string, error) {
	out := new(bytes.Buffer)
	t := template.Must(template.New("compiled_template").Parse(tmplt))
	if err := t.Execute(out, configMap); err != nil {
		return "", err
	}
	return out.String(), nil
}

type ManifestConfig struct {
	ClusterCIDR   string
	AuthURL       string
	AdminUser     string
	AdminPassword string
	AdminProject  string
	KubeCluster   string
	Region        string

	// cni config
	CNIImage string
	// cloudprovider config
	CloudProviderImage string
	// csi config
	CSIAttacher    string
	CSIProvisioner string
	CSIRegistrar   string
	CSIImage       string
}

func GetYunionManifest(config ManifestConfig) (string, error) {
	return CompileTemplateFromMap(YunionManifestTemplate, config)
}
