package models

import (
	"context"
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/helm/pkg/strvals"
	"sigs.k8s.io/yaml"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/drivers"
	"yunion.io/x/yunion-kube/pkg/helm"
)

var (
	ReleaseManager *SReleaseManager
)

func init() {
	ReleaseManager = &SReleaseManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			&SRelease{},
			"releases_tbl",
			"release",
			"releases"),
		driverManager: drivers.NewDriverManager(""),
	}
	ReleaseManager.SetVirtualObject(ReleaseManager)
}

type SReleaseManager struct {
	db.SVirtualResourceBaseManager
	SNamespaceResourceBaseManager

	driverManager *drivers.DriverManager
}

type SRelease struct {
	db.SVirtualResourceBase
	SNamespaceResourceBase

	RepoId       string               `width:"128" charset:"ascii" nullable:"true" index:"true" list:"user"`
	Chart        string               `width:"128" charset:"ascii" nullable:"true" index:"true" list:"user"`
	ChartVersion string               `width:"128" charset:"ascii" nullable:"false" list:"user"`
	Config       jsonutils.JSONObject `list:"user" update:"user"`
}

type IReleaseDriver interface {
	GetType() apis.RepoType
	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, data *apis.ReleaseCreateInput) (*apis.ReleaseCreateInput, error)
	CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, r *SRelease, data *apis.ReleaseCreateInput) error
}

func (m *SReleaseManager) RegisterDriver(driver IReleaseDriver) {
	if err := m.driverManager.Register(driver, string(driver.GetType())); err != nil {
		panic(errors.Wrapf(err, "release register driver %q", driver.GetType()))
	}
}

func (m *SReleaseManager) GetDriver(typ apis.RepoType) (IReleaseDriver, error) {
	drv, err := m.driverManager.Get(string(typ))
	if err != nil {
		if errors.Cause(err) == drivers.ErrDriverNotFound {
			return nil, httperrors.NewNotFoundError("release get %s driver", typ)
		}
	}
	return drv.(IReleaseDriver), nil
}

func (m *SReleaseManager) GetChartClient(repo *SRepo) *helm.ChartClient {
	return repo.GetChartClient()
}

func (m *SReleaseManager) ShowChart(repo *SRepo, chartName string, version string) (*chart.Chart, error) {
	chartCli := m.GetChartClient(repo)
	chartObj, err := chartCli.Show(repo.GetName(), chartName, version)
	if err != nil {
		return nil, errors.Wrapf(err, "get chart %s/%s:%s", repo.GetName(), chartName, version)
	}
	return chartObj, nil
}

func (m *SReleaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, data *apis.ReleaseCreateInput) (*apis.ReleaseCreateInput, error) {
	if data.ReleaseName != "" {
		data.Name = data.ReleaseName
	}

	var (
		repo  string
		chart string
	)

	if len(data.ChartName) != 0 {
		segs := strings.Split(data.ChartName, "/")
		if len(segs) == 2 {
			repo, chart = segs[0], segs[1]
		} else {
			return nil, httperrors.NewInputParameterError("Illegal chart name: %q", data.ChartName)
		}
	}
	if data.Repo != "" {
		repo = data.Repo
	}
	if data.Chart != "" {
		chart = data.Chart
	}
	if repo == "" {
		return nil, httperrors.NewNotEmptyError("repo must provided")
	}
	if chart == "" {
		return nil, httperrors.NewNotEmptyError("chart must provided")
	}
	repoObj, err := db.FetchByIdOrName(RepoManager, userCred, repo)
	if err != nil {
		return nil, NewCheckIdOrNameError("repo %v", err)
	}
	data.Repo = repoObj.GetId()
	drv, err := m.GetDriver(repoObj.(*SRepo).GetType())
	if err != nil {
		return nil, err
	}
	data, err = drv.ValidateCreateData(ctx, userCred, ownerCred, data)
	if err != nil {
		return nil, err
	}

	if data.Version == "" {
		data.Version = ">0.0.0-0"
	}
	chartObj, err := m.ShowChart(repoObj.(*SRepo), chart, data.Version)
	if err != nil {
		return nil, err
	}
	data.Version = chartObj.Metadata.Version
	data.Chart = chart
	return data, nil
}

func MergeValues(yamlStr string, sets map[string]string) (map[string]interface{}, error) {
	base := map[string]interface{}{}
	if len(yamlStr) != 0 {
		currentMap := map[string]interface{}{}
		if err := yaml.Unmarshal([]byte(yamlStr), &currentMap); err != nil {
			return nil, errors.Wrapf(err, "failed to parse %s", yamlStr)
		}
		base = mergeMaps(base, currentMap)
	}

	for key, value := range sets {
		setStr := fmt.Sprintf("%s=%s", key, value)
		if err := strvals.ParseInto(setStr, base); err != nil {
			return nil, errors.Wrapf(err, "failed parsing set %q", value)
		}
	}
	return base, nil
}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

func (r *SRelease) GetRepo() (*SRepo, error) {
	obj, err := RepoManager.FetchById(r.RepoId)
	if err != nil {
		return nil, err
	}
	return obj.(*SRepo), nil
}

func (r *SRelease) GetType() (apis.RepoType, error) {
	repo, err := r.GetRepo()
	if err != nil {
		return "", errors.Wrap(err, "get repo")
	}
	return repo.GetType(), nil
}

func (r *SRelease) GetDriver() (IReleaseDriver, error) {
	typ, err := r.GetType()
	if err != nil {
		return nil, errors.Wrap(err, "get type")
	}
	return ReleaseManager.GetDriver(typ)
}

func (r *SRelease) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := r.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	input := new(apis.ReleaseCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return errors.Wrap(err, "unmarshal data")
	}
	config, err := MergeValues(input.Values, input.Sets)
	if err != nil {
		return errors.Wrap(err, "generate config")
	}
	r.Config = jsonutils.Marshal(config)
	r.RepoId = input.Repo
	r.Chart = input.Chart
	r.ChartVersion = input.Version
	drv, err := r.GetDriver()
	if err != nil {
		return errors.Wrap(err, "customize create get driver")
	}
	r.Namespace = input.Namespace
	return drv.CustomizeCreate(ctx, userCred, ownerId, r, input)
}

func (r *SRelease) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	if err := r.StartCreateReleaseTask(ctx, userCred, ""); err != nil {
		log.Errorf("Create release task error: %v", err)
	}
}

func (r *SRelease) StartCreateReleaseTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	if err := r.SetStatus(userCred, apis.ReleaseStatusDeploying, ""); err != nil {
		return err
	}
	task, err := taskman.TaskManager.NewTask(ctx, "ReleaseCreateTask", r, userCred, jsonutils.NewDict(), parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (r *SRelease) GetHelmClient() (*helm.Client, error) {
	cls, err := r.GetCluster()
	if err != nil {
		return nil, err
	}
	return NewHelmClient(cls, r.Namespace)
}

func (r *SRelease) GetChartClient() (*helm.ChartClient, error) {
	repo, err := r.GetRepo()
	if err != nil {
		return nil, err
	}
	return ReleaseManager.GetChartClient(repo), nil
}

func (r *SRelease) GetChart() (*chart.Chart, error) {
	repo, err := r.GetRepo()
	if err != nil {
		return nil, err
	}
	return ReleaseManager.ShowChart(repo, r.Chart, r.ChartVersion)
}

func (r *SRelease) GetHelmValues() map[string]interface{} {
	vals := map[string]interface{}{}
	if r.Config == nil {
		return vals
	}
	yamlStr := r.Config.YAMLString()
	yaml.Unmarshal([]byte(yamlStr), &vals)
	return vals
}

func (r *SRelease) DoCreate() error {
	chart, err := r.GetChart()
	if err != nil {
		return errors.Wrap(err, "get chart")
	}
	cli, err := r.GetHelmClient()
	if err != nil {
		return errors.Wrap(err, "get helm client when create")
	}
	install := cli.Release().Install()
	install.Namespace = r.Namespace
	install.ReleaseName = r.GetName()
	install.Atomic = true
	install.Replace = true
	vals := r.GetHelmValues()
	rls, err := install.Run(chart, vals)
	if err != nil {
		return errors.Wrap(err, "install release")
	}
	log.Infof("helm release %s installed", rls.Name)
	return nil
}
