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
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/helm"
	"yunion.io/x/yunion-kube/pkg/options"
)

type SRepoManager struct {
	db.SSharableVirtualResourceBaseManager
}

var RepoManager *SRepoManager

func init() {
	RepoManager = &SRepoManager{SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(SRepo{}, "repos_tbl", "repo", "repos")}
	RepoManager.SetVirtualObject(RepoManager)
}

// TODO: insert stable and incubator repo
/* func (m *SRepoManager) InitializeData() error {
	// check if default repo exists
	_, err := m.FetchByIdOrName(nil, STABLE_REPO_NAME)
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
		defRepo := SRepo{}
		defRepo.Id = stringutils.UUID4()
		defRepo.Name = STABLE_REPO_NAME
		defRepo.Url = options.Options.StableChartRepo
		err = m.TableSpec().Insert(&defRepo)
		if err != nil {
			return fmt.Errorf("Insert default repo error: %v", err)
		}
	}
	return nil
}*/

type SRepo struct {
	db.SSharableVirtualResourceBase

	Url      string `width:"256" charset:"ascii" nullable:"false" create:"required" list:"user" update:"admin"`
	Username string `width:"256" charset:"ascii" nullable:"false"`
	Password string `width:"256" charset:"ascii" nullable:"false"`
	Type     string `charset:"ascii" width:"128" nullable:"true" list:"user"`
}

func (man *SRepoManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (man *SRepoManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	return man.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
}

func (man *SRepoManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, man)
}

func (man *SRepoManager) GetClient(projectId string) (*helm.RepoClient, error) {
	dataDir := path.Join(options.Options.HelmDataDir, projectId)
	return helm.NewRepoClient(dataDir)
}

func (man *SRepoManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.RepoCreateInput) (*api.RepoCreateInput, error) {
	shareInput, err := man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data.SharableVirtualResourceCreateInput)
	if err != nil {
		return nil, err
	}
	data.SharableVirtualResourceCreateInput = shareInput
	if data.Url == "" {
		return nil, httperrors.NewInputParameterError("Missing repo url")
	}
	if _, err := url.Parse(data.Url); err != nil {
		return nil, httperrors.NewNotAcceptableError("Invalid repo url: %v", err)
	}

	if data.Type == "" {
		data.Type = api.RepoTypeCommunity
	}
	if !utils.IsInStringArray(data.Type, []string{api.RepoTypeCommunity, api.RepoTypeOneCloud}) {
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

func (r *SRepo) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	cli, err := RepoManager.GetClient(r.ProjectId)
	if err != nil {
		return err
	}
	if err := cli.Remove(r.Name); err != nil {
		return err
	}
	return r.SSharableVirtualResourceBase.Delete(ctx, userCred)
}

func (r *SRepo) AllowPerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, r, "sync")
}

func (r *SRepo) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, r.DoSync()
}

func (r *SRepo) DoSync() error {
	cli, err := RepoManager.GetClient(r.ProjectId)
	if err != nil {
		return err
	}
	return cli.Update(r.Name)
}
