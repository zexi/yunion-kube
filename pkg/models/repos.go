package models

import (
	"context"
	"net/url"
	"path"

	"helm.sh/helm/v3/pkg/repo"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/helm"
	"yunion.io/x/yunion-kube/pkg/options"
)

type SRepoManager struct {
	db.SStatusInfrasResourceBaseManager
}

var RepoManager *SRepoManager

func init() {
	RepoManager = &SRepoManager{SStatusInfrasResourceBaseManager: db.NewStatusInfrasResourceBaseManager(SRepo{}, "repos_tbl", "repo", "repos")}
	RepoManager.SetVirtualObject(RepoManager)
}

func (m *SRepoManager) InitializeData() error {
	// 填充 v2 没有 tenant_id 的 repo，默认变为 system project
	repos := []SRepo{}
	q := m.Query()
	q = q.Filter(sqlchemy.OR(
		sqlchemy.IsNullOrEmpty(q.Field("domain_id")),
		sqlchemy.IsNullOrEmpty(q.Field("domain_src")),
		sqlchemy.IsNullOrEmpty(q.Field("type")),
	))
	if err := db.FetchModelObjects(m, q, &repos); err != nil {
		return errors.Wrap(err, "fetch empty project repos")
	}
	userCred := GetAdminCred()
	for _, r := range repos {
		tmpRepo := &r
		if _, err := db.Update(tmpRepo, func() error {
			tmpRepo.DomainId = userCred.GetProjectDomainId()
			tmpRepo.Type = string(api.RepoTypeExternal)
			return nil
		}); err != nil {
			return errors.Wrapf(err, "update empty project repo %s", tmpRepo.GetName())
		}
	}

	return nil
}

type SRepo struct {
	db.SStatusInfrasResourceBase

	Url      string `width:"256" charset:"ascii" nullable:"false" create:"required" list:"user"`
	Username string `width:"256" charset:"ascii" nullable:"false"`
	Password string `width:"256" charset:"ascii" nullable:"false"`
	Type     string `charset:"ascii" width:"128" create:"required" nullable:"true" list:"user"`
}

func (man *SRepoManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (man *SRepoManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.RepoListInput) (*sqlchemy.SQuery, error) {
	q, err := man.SStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, err
	}
	if input.Type != "" {
		q = q.Equals("type", input.Type)
	}
	return q, nil
}

func (man *SRepoManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool) []api.RepoDetail {
	rows := make([]api.RepoDetail, len(objs))
	svRows := man.SStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range svRows {
		rObj := objs[i].(*SRepo)
		detail := api.RepoDetail{
			StatusInfrasResourceBaseDetails: svRows[i],
			Url:                             rObj.Url,
			Type:                            rObj.Type,
		}
		rows[i] = detail
	}
	return rows
}

func (man *SRepoManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, man)
}

func (man *SRepoManager) GetRepoDataDir(projectId string) string {
	return path.Join(options.Options.HelmDataDir, projectId)
}

func (man *SRepoManager) GetClient(projectId string) (*helm.RepoClient, error) {
	dataDir := man.GetRepoDataDir(projectId)
	return helm.NewRepoClient(dataDir)
}

func (man *SRepoManager) GetChartClient(projectId string) *helm.ChartClient {
	dataDir := man.GetRepoDataDir(projectId)
	return helm.NewChartClient(dataDir)
}

func (man *SRepoManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.RepoCreateInput) (*api.RepoCreateInput, error) {
	shareInput, err := man.SStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data.StatusInfrasResourceBaseCreateInput)
	if err != nil {
		return nil, err
	}
	data.StatusInfrasResourceBaseCreateInput = shareInput
	if data.Url == "" {
		return nil, httperrors.NewInputParameterError("Missing repo url")
	}
	if _, err := url.Parse(data.Url); err != nil {
		return nil, httperrors.NewNotAcceptableError("Invalid repo url: %v", err)
	}

	if data.Type == "" {
		data.Type = string(api.RepoTypeExternal)
	}
	if !utils.IsInStringArray(data.Type, []string{string(api.RepoTypeExternal), string(api.RepoTypeInternal)}) {
		return nil, httperrors.NewInputParameterError("Not support type %q", data.Type)
	}

	entry := &repo.Entry{
		Name: data.Name,
		URL:  data.Url,
	}
	cli, err := man.GetClient(ownerId.GetProjectId())
	if err != nil {
		return nil, err
	}
	if err := cli.Add(entry); err != nil {
		return nil, err
	}

	return data, nil
}

func (man *SRepoManager) FetchRepoById(id string) (*SRepo, error) {
	repo, err := man.FetchById(id)
	if err != nil {
		return nil, err
	}
	return repo.(*SRepo), nil
}

func (man *SRepoManager) FetchRepoByIdOrName(userCred mcclient.IIdentityProvider, ident string) (*SRepo, error) {
	repo, err := man.FetchByIdOrName(userCred, ident)
	if err != nil {
		return nil, err
	}
	return repo.(*SRepo), nil
}

func (man *SRepoManager) ListRepos() ([]SRepo, error) {
	q := man.Query()
	repos := make([]SRepo, 0)
	err := db.FetchModelObjects(RepoManager, q, &repos)
	return repos, err
}

func (r *SRepo) ToEntry() *repo.Entry {
	return &repo.Entry{
		Name:     r.Name,
		URL:      r.Url,
		Username: r.Username,
		Password: r.Password,
	}
}

func (r *SRepo) ValidateDeleteCondition(ctx context.Context) error {
	rlsCnt, err := ReleaseManager.Query().Equals("repo_id", r.GetId()).CountWithError()
	if err != nil {
		return errors.Wrap(err, "check release count")
	}
	if rlsCnt != 0 {
		return httperrors.NewNotAcceptableError("%d release use this repo", rlsCnt)
	}
	return nil
}

func (r *SRepo) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	cli, err := RepoManager.GetClient(r.DomainId)
	if err != nil {
		return err
	}
	if err := cli.Remove(r.Name); err != nil {
		return err
	}
	return r.SStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (r *SRepo) AllowPerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, r, "sync")
}

func (r *SRepo) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, r.DoSync()
}

func (r *SRepo) DoSync() error {
	cli, err := RepoManager.GetClient(r.DomainId)
	if err != nil {
		return err
	}
	entry := &repo.Entry{
		Name: r.Name,
		URL:  r.Url,
	}
	if err := cli.Add(entry); err != nil && errors.Cause(err) != helm.ErrRepoAlreadyExists {
		return err
	}
	return cli.Update(r.Name)
}

func (r *SRepo) GetType() api.RepoType {
	return api.RepoType(r.Type)
}

func (r *SRepo) GetChartClient() *helm.ChartClient {
	return RepoManager.GetChartClient(r.DomainId)
}
