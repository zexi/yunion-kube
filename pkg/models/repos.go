package models

import (
	"context"


	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type SRepoManager struct {
	db.SSharableVirtualResourceBaseManager
}

var RepoManager *SRepoManager

func init() {
	RepoManager = &SRepoManager{SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(SRepo{}, "repos_tbl", "repo", "repos")}
}

type SRepo struct {
	db.SStandaloneResourceBase

	Url string `width:"256" charset:"ascii" nullable:"false" create:"required" list:"user" update:"admin"`
	Source string `width:"256" charset:"ascii" nullable:"true" list:"user" update:"admin"`
}

func (man *SRepoManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (man *SRepoManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	return man.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
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
	return man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (man *SRepoManager) FetchRepoById(id string) *SRepo {
	repo, err := man.FetchById(id)
	if err  != nil {
		log.Errorf("Fetch repo by id %q error: %v", id, err)
		return nil
	}
	return repo.(*SRepo)
}

func (man *SRepoManager) FetchRepoByIdOrName(ownerProjId, ident string) *SRepo {
	repo, err := man.FetchByIdOrName(ownerProjId, ident)
	if err  != nil {
		log.Errorf("Fetch repo by id or name %q error: %v", ident, err)
		return nil
	}
	return repo.(*SRepo)
}