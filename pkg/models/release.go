package models

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/helm/pkg/strvals"
	"sigs.k8s.io/yaml"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/drivers"
	"yunion.io/x/yunion-kube/pkg/helm"
)

var (
	ReleaseManager *SReleaseManager
)

func init() {
	ReleaseManager = &SReleaseManager{
		SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
			&SRelease{},
			"releases_tbl",
			"release",
			"releases",
			"",
			"",
			nil),
		driverManager: drivers.NewDriverManager(""),
	}
	ReleaseManager.SetVirtualObject(ReleaseManager)
}

type SReleaseManager struct {
	SNamespaceResourceBaseManager

	driverManager *drivers.DriverManager
}

type SRelease struct {
	SNamespaceResourceBase

	RepoId       string               `width:"128" charset:"ascii" nullable:"false" index:"true" list:"user"`
	Chart        string               `width:"128" charset:"ascii" nullable:"false" create:"required" index:"true" list:"user"`
	ChartVersion string               `width:"128" charset:"ascii" nullable:"false" list:"user"`
	Config       jsonutils.JSONObject `nullable:"true" list:"user" update:"user"`
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
		return nil, NewCheckIdOrNameError("repo", repo, err)
	}
	data.Repo = repoObj.GetId()

	if data.ReleaseName != "" {
		data.Name = data.ReleaseName
	}

	drv, err := m.GetDriver(repoObj.(*SRepo).GetType())
	if err != nil {
		return nil, err
	}
	data, err = drv.ValidateCreateData(ctx, userCred, ownerCred, data)
	if err != nil {
		return nil, err
	}

	nInput, err := m.SNamespaceResourceBaseManager.ValidateCreateData(ctx, userCred, ownerCred, query, &data.NamespaceResourceCreateInput)
	if err != nil {
		return nil, err
	}
	data.NamespaceResourceCreateInput = *nInput

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

func (m *SReleaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.ReleaseDetailV2 {
	rows := make([]apis.ReleaseDetailV2, len(objs))
	nRows := m.SNamespaceResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range nRows {
		detail := apis.ReleaseDetailV2{
			ReleaseV2: apis.ReleaseV2{
				NamespaceResourceDetail: nRows[i],
			},
		}
		var err error
		if !isList {
			detail, err = objs[i].(*SRelease).fillReleaseDetail(detail)
			if err != nil {
				log.Errorf("Get release detail error: %v", err)
			}
		}
		rows[i] = detail
	}
	return rows
}

func (rls *SRelease) fillReleaseDetail(detail apis.ReleaseDetailV2) (apis.ReleaseDetailV2, error) {
	rel, err := rls.GetHelmRelease()
	if err != nil {
		return detail, errors.Wrap(err, "get helm release detail")
	}
	detail.Info = rel.Info
	detail.Chart = rel.Chart
	detail.Config = rel.Config
	detail.Manifest = rel.Manifest
	detail.Hooks = rel.Hooks
	detail.Version = rel.Version
	detail.Resources = rel.Resources
	detail.Files = rel.Files
	return detail, nil
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
	if err := r.SNamespaceResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
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
	return drv.CustomizeCreate(ctx, userCred, ownerId, r, input)
}

func (r *SRelease) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	r.SNamespaceResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	r.StartCreateTask(r, ctx, userCred, ownerId, query, data)
}

func (m *SReleaseManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	// not invoke ClusterResourceManager general k8s resource create
	return nil, nil
}

func (r *SRelease) doCreate() (*release.Release, error) {
	ns, err := r.GetNamespace()
	if err != nil {
		return nil, errors.Wrap(err, "get relase namespace")
	}
	chart, err := r.GetChart()
	if err != nil {
		return nil, errors.Wrap(err, "get chart")
	}
	cli, err := r.GetHelmClient()
	if err != nil {
		return nil, errors.Wrap(err, "get helm client when create")
	}
	install := cli.Release().Install()
	install.Namespace = ns.GetName()
	install.ReleaseName = r.GetName()
	install.Atomic = true
	install.Replace = true
	vals := r.GetHelmValues()
	rls, err := install.Run(chart, vals)
	if err != nil {
		return nil, errors.Wrap(err, "install release")
	}
	log.Infof("helm release %s installed", rls.Name)
	return rls, nil
}

func (m *SReleaseManager) CreateRemoteObject(model IClusterModel, _ *client.ClusterManager, _ interface{}) (interface{}, error) {
	r := model.(*SRelease)
	ns, err := r.GetNamespace()
	if err != nil {
		return nil, errors.Wrap(err, "get namespace")
	}
	nsActive := false
	for i := 0; i < 10; i++ {
		if ns.Status != string(v1.NamespaceActive) {
			log.Warningf("namespace %s status is %s, wait it be %s", ns.GetName(), ns.Status, v1.NamespaceActive)
			time.Sleep(time.Second * 5)
			continue
		}
		nsActive = true
		break
	}
	if !nsActive {
		return nil, errors.Errorf("namespace status is %s, not %s", ns.GetStatus(), v1.NamespaceActive)
	}
	return r.doCreate()
}

func (r *SRelease) UpdateFromRemoteObject(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	extObj interface{}) error {
	rls := extObj.(*release.Release)
	if r.GetName() != rls.Name {
		r.SetName(rls.Name)
	}
	if r.Status != string(rls.Info.Status) {
		r.Status = string(rls.Info.Status)
	}
	return nil
}

func (r *SRelease) GetHelmClient() (*helm.Client, error) {
	cls, err := r.GetCluster()
	if err != nil {
		return nil, err
	}
	ns, err := r.GetNamespace()
	if err != nil {
		return nil, err
	}
	return NewHelmClient(cls, ns.GetName())
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

func (r *SRelease) GetHelmRelease() (*apis.ReleaseDetail, error) {
	helmCli, err := r.GetHelmClient()
	if err != nil {
		return nil, errors.Wrap(err, "get helm client")
	}
	rls, err := helmCli.Release().ReleaseContent(r.GetName(), -1)
	if err != nil {
		return nil, errors.Wrap(err, "get helm release")
	}
	clusCli, err := r.GetClusterClient()
	if err != nil {
		return nil, errors.Wrap(err, "get cluster client")
	}
	res, err := GetReleaseResources(helmCli, rls, clusCli)
	return &apis.ReleaseDetail{
		Release:   *ToRelease(rls, clusCli.Cluster.(apis.ICluster)),
		Resources: res,
		Files:     GetChartRawFiles(rls.Chart),
	}, nil
}

func ToRelease(release *release.Release, cluster apis.ICluster) *apis.Release {
	return &apis.Release{
		Release:     release,
		ClusterMeta: apis.NewClusterMeta(cluster),
		Status:      release.Info.Status.String(),
	}
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

func (m *SReleaseManager) ListRemoteObjects(cli *client.ClusterManager) ([]interface{}, error) {
	helmCli, err := NewHelmClient(cli.Cluster.(*SCluster), "")
	if err != nil {
		return nil, errors.Wrap(err, "new helm client")
	}
	listAct := helmCli.Release().List()
	listAct.All = true
	listAct.AllNamespaces = true
	resp, err := listAct.Run()
	if err != nil {
		return nil, errors.Wrap(err, "list helm all release")
	}
	ret := make([]interface{}, len(resp))
	for i := range resp {
		ret[i] = resp[i]
	}
	return ret, nil
}

func (m *SReleaseManager) getRemoteReleaseGlobalId(clusterId, namespace, name string) string {
	return fmt.Sprintf("%s/%s/%s", clusterId, namespace, name)
}

func (m *SReleaseManager) GetRemoteObjectGlobalId(cluster *SCluster, obj interface{}) string {
	rls := obj.(*release.Release)
	return m.getRemoteReleaseGlobalId(cluster.GetId(), rls.Namespace, rls.Name)
}

func (m *SReleaseManager) NewFromRemoteObject(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cluster *SCluster,
	obj interface{}) (IClusterModel, error) {
	dbObj, err := db.NewModelObject(m)
	if err != nil {
		return nil, err
	}
	rls := obj.(*release.Release)
	dbObj.(db.IExternalizedModel).SetExternalId(m.GetRemoteObjectGlobalId(cluster, obj))
	dbObj.(IClusterModel).SetName(rls.Name)
	dbObj.(IClusterModel).SetCluster(userCred, cluster)
	// set local db namespace object
	localNs, err := NamespaceManager.GetByName(userCred, cluster.GetId(), rls.Namespace)
	if err != nil {
		return nil, err
	}
	dbObj.(INamespaceModel).SetNamespace(userCred, localNs.(*SNamespace))
	return dbObj.(IClusterModel), nil
}

func (m *SReleaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *apis.ReleaseListInputV2) (*sqlchemy.SQuery, error) {
	q, err := m.SNamespaceResourceBaseManager.ListItemFilter(ctx, q, userCred, &input.NamespaceResourceListInput)
	if err != nil {
		return nil, err
	}
	repos := RepoManager.Query().SubQuery()
	if input.Type != "" {
		sq := repos.Query(repos.Field("id")).Equals("type", input.Type).SubQuery()
		q.In("repo_id", sq)
	}
	return q, nil
}

func (obj *SRelease) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	obj.SNamespaceResourceBase.PostDelete(ctx, userCred)
	obj.StartDeleteTask(obj, ctx, userCred)
}

func (obj *SRelease) DeleteRemoteObject(_ *client.ClusterManager) error {
	helmCli, err := obj.GetHelmClient()
	if err != nil {
		return errors.Wrap(err, "get helm client when delete")
	}
	act := helmCli.Release().UnInstall()
	if _, err := act.Run(obj.GetName()); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}
	return nil
}