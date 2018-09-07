package models

import (
	"context"

	"k8s.io/helm/pkg/repo"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/sqlchemy"

	repobackend "yunion.io/x/yunion-kube/pkg/helm/data/cache/repo"
)

type SRepoManager struct {
	db.SStandaloneResourceBaseManager
	models.SInfrastructureManager
}

var RepoManager *SRepoManager

func init() {
	RepoManager = &SRepoManager{SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(SRepo{}, "repos_tbl", "repo", "repos")}
}

type SRepo struct {
	db.SStandaloneResourceBase
	models.SInfrastructure

	Url    string `width:"256" charset:"ascii" nullable:"false" create:"required" list:"user" update:"admin"`
	Source string `width:"256" charset:"ascii" nullable:"true" list:"user" update:"admin"`
}

func (man *SRepoManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (man *SRepoManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	return man.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
}

func (man *SRepoManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	name, _ := data.GetString("name")
	if name == "" {
		return nil, httperrors.NewInputParameterError("Missing name")
	}
	url, _ := data.GetString("url")
	if url == "" {
		return nil, httperrors.NewInputParameterError("Missing repo url")
	}
	return man.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (man *SRepoManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	repos := make([]*SRepo, len(items))
	for i, t := range items {
		repos[i] = t.(*SRepo)
	}
	for _, r := range repos {
		go func() {
			err := r.AddToBackend()
			if err != nil {
				log.Errorf("Add repo to backend: %v", err)
			}
		}()
	}
}

func (man *SRepoManager) FetchRepoById(id string) (*SRepo, error) {
	repo, err := man.FetchById(id)
	if err != nil {
		return nil, err
	}
	return repo.(*SRepo), nil
}

func (man *SRepoManager) FetchRepoByIdOrName(ownerProjId, ident string) (*SRepo, error) {
	repo, err := man.FetchByIdOrName(ownerProjId, ident)
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
		Name: r.Name,
		URL:  r.Url,
	}
}

func (r *SRepo) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := repobackend.BackendManager.Delete(r.Name)
	_, ok := err.(repobackend.ErrorRepoNotFound)
	if err != nil && !ok {
		return err
	}
	return r.SStandaloneResourceBase.Delete(ctx, userCred)
}

func (r *SRepo) AddToBackend() error {
	err := repobackend.BackendManager.Add(r.ToEntry())
	return err
}

func (r *SRepo) AllowPerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return allowPerformAction(ctx, userCred, query, data)
}

func (r *SRepo) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, r.DoSync()
}

func (r *SRepo) DoSync() error {
	return repobackend.BackendManager.Update(r.Name)
}