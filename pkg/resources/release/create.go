package release

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/ghodss/yaml"
	"k8s.io/helm/pkg/helm"
	rls "k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/helm/pkg/strvals"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/yunion-kube/pkg/helm/client"
	helmdata "yunion.io/x/yunion-kube/pkg/helm/data"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	helmtypes "yunion.io/x/yunion-kube/pkg/types/helm"
)

func generateName(nameTemplate string) (string, error) {
	t, err := template.New("name-template").Funcs(sprig.TxtFuncMap()).Parse(nameTemplate)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	err = t.Execute(&b, nil)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func GenerateName(nameTemplate string) (string, error) {
	return generateName(nameTemplate)
}

type CreateUpdateReleaseRequest struct {
	ChartName   string   `json:"chart_name"`
	Namespace   string   `json:"namespace"`
	ReleaseName string   `json:"release_name"`
	Version     string   `json:"version"`
	ReUseValues bool     `json:"reuse_values"`
	ResetValues bool     `json:"reset_values"`
	DryRun      bool     `json:"dry_run"`
	Values      string   `json:"values"`
	Sets        []string `json:"sets"`
	Timeout     int64    `json:"timeout"`
}

func NewCreateUpdateReleaseReq(params jsonutils.JSONObject) (*CreateUpdateReleaseRequest, error) {
	var req CreateUpdateReleaseRequest
	err := params.Unmarshal(&req)
	if err != nil {
		return nil, err
	}
	if req.Timeout == 0 {
		req.Timeout = 1500 // set default 15 mins timeout
	}
	return &req, nil
}

func (c *CreateUpdateReleaseRequest) Vals() ([]byte, error) {
	return MergeBytesValues([]byte(c.Values), c.Sets)
}

type valueFiles []string

func (v valueFiles) String() string {
	return fmt.Sprintf("%s", v)
}

func (v valueFiles) Type() string {
	return "valueFiles"
}

func (v *valueFiles) Set(value string) error {
	for _, fp := range strings.Split(value, ",") {
		*v = append(*v, fp)
	}
	return nil
}

func mergeValues(dest map[string]interface{}, src map[string]interface{}) map[string]interface{} {
	for k, v := range src {
		// If the key doesn't exist already, then just set the key to that value
		if _, exists := dest[k]; !exists {
			dest[k] = v
			continue
		}
		nextMap, ok := v.(map[string]interface{})
		// If it isn't another map, overwrite the value
		if !ok {
			dest[k] = v
			continue
		}
		// If the key doesn't exist already, then just set the key to that value
		if _, exists := dest[k]; !exists {
			dest[k] = nextMap
			continue
		}
		// Edge case: If the key exists in the destination, but isn't a map
		destMap, isMap := dest[k].(map[string]interface{})
		// If the source map has a map for this key, prefer it
		if !isMap {
			dest[k] = v
			continue
		}
		// If we got to this point, it is a map in both, so merge them
		dest[k] = mergeValues(destMap, nextMap)
	}
	return dest
}

func MergeValues(values, stringValues []string) ([]byte, error) {
	return MergeValuesF([]string{}, values, stringValues)
}

func MergeBytesValues(vbytes []byte, values []string) ([]byte, error) {
	base := map[string]interface{}{}
	currentMap := map[string]interface{}{}
	if err := yaml.Unmarshal(vbytes, &currentMap); err != nil {
		return []byte{}, fmt.Errorf("Failed to parse: %s, error: %v", string(vbytes), err)
	}
	base = mergeValues(base, currentMap)

	for _, value := range values {
		if err := strvals.ParseInto(value, base); err != nil {
			return []byte{}, fmt.Errorf("failed parsing set data: %s", err)
		}
	}
	return yaml.Marshal(base)
}

func MergeValuesF(valueFiles valueFiles, values, stringValues []string) ([]byte, error) {
	base := map[string]interface{}{}

	// parse values files
	for _, filePath := range valueFiles {
		currentMap := map[string]interface{}{}

		var bbytes []byte
		var err error
		bbytes, err = ioutil.ReadFile(filePath)
		if err != nil {
			return []byte{}, err
		}

		if err := yaml.Unmarshal(bbytes, &currentMap); err != nil {
			return []byte{}, fmt.Errorf("Failed to parse %s: %s", filePath, err)
		}
		// Merge with the previous map
		base = mergeValues(base, currentMap)
	}

	// parse set values
	for _, value := range values {
		if err := strvals.ParseInto(value, base); err != nil {
			return []byte{}, fmt.Errorf("failed parsing set data: %s", err)
		}
	}

	// parse set string values
	for _, value := range stringValues {
		if err := strvals.ParseIntoString(value, base); err != nil {
			return []byte{}, fmt.Errorf("failed parsing set string: %s", err)
		}
	}

	return yaml.Marshal(base)
}

func (man *SReleaseManager) ValidateCreateData(req *common.Request) error {
	data := req.Data
	ns, _ := data.GetString("namespace")
	if ns == "" {
		data.Set("namespace", jsonutils.NewString(req.GetDefaultNamespace()))
	}
	name, _ := data.GetString("release_name")
	if name == "" {
		name, err := generateName("")
		if err != nil {
			return err
		}
		data.Set("release_name", jsonutils.NewString(name))
	}
	return nil
}

func (man *SReleaseManager) Create(req *common.Request) (interface{}, error) {
	createOpt, err := NewCreateUpdateReleaseReq(req.Data)
	if err != nil {
		return nil, err
	}
	cli, err := req.GetHelmClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	return ReleaseCreate(cli, createOpt)
}

func validateInfraCreate(cli *client.HelmTunnelClient, chartPkg *helmtypes.ChartPackage) error {
	releases, err := ListReleases(cli, ReleaseListQuery{All: true})
	if err != nil {
		return err
	}
	for _, rls := range releases.Releases {
		if rls.Chart.Metadata.Name == chartPkg.Metadata.Name {
			return httperrors.NewBadRequestError("Release %s already created by chart %s", rls.Name, rls.Chart.Metadata.Name)
		}
	}
	return nil
}

func ReleaseCreate(helmclient *client.HelmTunnelClient, opt *CreateUpdateReleaseRequest) (*rls.InstallReleaseResponse, error) {
	log.Infof("Deploying chart=%q, release name=%q", opt.ChartName, opt.ReleaseName)
	segs := strings.Split(opt.ChartName, "/")
	if len(segs) != 2 {
		return nil, fmt.Errorf("Illegal chart name: %q", opt.ChartName)
	}
	repoName, chartName := segs[0], segs[1]
	pkg, err := helmdata.ChartFromRepo(repoName, chartName, opt.Version)
	if err != nil {
		return nil, err
	}
	if repoName == helmtypes.YUNION_REPO_NAME {
		err = validateInfraCreate(helmclient, pkg)
		if err != nil {
			return nil, err
		}
	}
	chartRequest := pkg.Chart
	vals, err := opt.Vals()
	if err != nil {
		return nil, err
	}
	installRes, err := helmclient.InstallReleaseFromChart(
		chartRequest,
		opt.Namespace,
		helm.ValueOverrides(vals),
		helm.ReleaseName(opt.ReleaseName),
		helm.InstallDryRun(opt.DryRun),
		helm.InstallReuseName(true),
		helm.InstallDisableHooks(true),
		helm.InstallTimeout(opt.Timeout),
		helm.InstallWait(false))
	if err != nil {
		return nil, fmt.Errorf("Error deploying chart: %v", err)
	}
	return installRes, nil
}
