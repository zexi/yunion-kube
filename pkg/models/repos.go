package models

import (
	"context"
	"database/sql"
	"fmt"

	"k8s.io/helm/pkg/repo"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/sqlchemy"

	repobackend "yunion.io/x/yunion-kube/pkg/helm/data"
	"yunion.io/x/yunion-kube/pkg/options"
	helmtypes "yunion.io/x/yunion-kube/pkg/types/helm"
)

const (
	YUNION_REPO_NAME = helmtypes.YUNION_REPO_NAME
)

type SRepoManager struct {
	db.SStandaloneResourceBaseManager
}

var RepoManager *SRepoManager

func init() {
	RepoManager = &SRepoManager{SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(SRepo{}, "repos_tbl", "repo", "repos")}
	RepoManager.SetVirtualObject(RepoManager)
}

func (m *SRepoManager) InitializeData() error {
	// check if default repo exists
	_, err := m.FetchByIdOrName(nil, YUNION_REPO_NAME)
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
		defRepo := SRepo{}
		defRepo.Id = stringutils.UUID4()
		defRepo.Name = YUNION_REPO_NAME
		defRepo.Url = options.Options.YunionChartRepo
		err = m.TableSpec().Insert(&defRepo)
		if err != nil {
			return fmt.Errorf("Insert default repo error: %v", err)
		}
	}
	return nil
}

type SRepo struct {
	db.SStandaloneResourceBase

	Url      string `width:"256" charset:"ascii" nullable:"false" create:"required" list:"user" update:"admin"`
	Source   string `width:"256" charset:"ascii" nullable:"true" list:"user" update:"admin"`
	IsPublic bool   `default:"false" nullable:"false" create:"admin_optional" list:"user"`
}

func (man *SRepoManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (man *SRepoManager) CustomizeFilterList(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*db.CustomizeListFilters, error) {
	filters := db.NewCustomizeListFilters()
	if userCred.HasSystemAdminPrivilege() {
		return filters, nil
	}
	publicFilter := func(obj jsonutils.JSONObject) (bool, error) {
		isPublic, _ := obj.Bool("is_public")
		return isPublic, nil
	}
	filters.Append(publicFilter)
	return filters, nil
}

func (man *SRepoManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	return man.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
}

func (man *SRepoManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.HasSystemAdminPrivilege()
}

func (man *SRepoManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	name, _ := data.GetString("name")
	if name == "" {
		return nil, httperrors.NewInputParameterError("Missing name")
	}
	url, _ := data.GetString("url")
	if url == "" {
		return nil, httperrors.NewInputParameterError("Missing repo url")
	}
	entry := &repo.Entry{
		Name: name,
		URL:  url,
	}
	err := repobackend.RepoBackendManager.Add(entry)
	if err != nil {
		return nil, err
	}

	return man.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
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

func (r *SRepo) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, r) || r.IsPublic
}

func (r *SRepo) AllowPerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return userCred.HasSystemAdminPrivilege()
}

func (r *SRepo) AllowPerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return userCred.HasSystemAdminPrivilege()
}

func (r *SRepo) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !r.IsPublic {
		_, err := r.GetModelManager().TableSpec().Update(r, func() error {
			r.IsPublic = true
			return nil
		})
		return nil, err
	}
	return nil, nil
}

func (r *SRepo) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if r.IsPublic {
		_, err := r.GetModelManager().TableSpec().Update(r, func() error {
			r.IsPublic = false
			return nil
		})
		return nil, err
	}
	return nil, nil
}

func (r *SRepo) ToEntry() *repo.Entry {
	return &repo.Entry{
		Name: r.Name,
		URL:  r.Url,
	}
}

func (r *SRepo) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := repobackend.RepoBackendManager.Delete(r.Name)
	_, ok := err.(repobackend.ErrorRepoNotFound)
	if err != nil && !ok {
		return err
	}
	return r.SStandaloneResourceBase.Delete(ctx, userCred)
}

func (r *SRepo) AddToBackend() error {
	err := repobackend.RepoBackendManager.Add(r.ToEntry())
	return err
}

func (r *SRepo) AllowPerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, r, "sync")
}

func (r *SRepo) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, r.DoSync()
}

func (r *SRepo) DoSync() error {
	return repobackend.RepoBackendManager.Update(r.Name)
}
