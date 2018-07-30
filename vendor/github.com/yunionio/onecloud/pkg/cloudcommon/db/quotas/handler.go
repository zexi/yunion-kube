package quotas

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/mcclient/auth"
	"github.com/yunionio/pkg/appctx"
	"github.com/yunionio/pkg/appsrv"
	"github.com/yunionio/pkg/httperrors"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
)

var _manager *SQuotaManager

func AddQuotaHandler(manager *SQuotaManager, prefix string, app *appsrv.Application) {
	_manager = manager

	app.AddHandler2("GET",
		fmt.Sprintf("%s/quotas", prefix),
		auth.Authenticate(getQuotaHanlder), nil, "get_quota", nil)

	app.AddHandler2("GET",
		fmt.Sprintf("%s/quotas/<tenantid>", prefix),
		auth.Authenticate(getQuotaHanlder), nil, "get_quota", nil)

	app.AddHandler2("POST",
		fmt.Sprintf("%s/quotas", prefix),
		auth.Authenticate(setQuotaHanlder), nil, "set_quota", nil)

	app.AddHandler2("POST",
		fmt.Sprintf("%s/quotas/<tenantid>", prefix),
		auth.Authenticate(setQuotaHanlder), nil, "set_quota", nil)

	app.AddHandler2("POST",
		fmt.Sprintf("%s/quotas/<tenantid>/<action>", prefix),
		auth.Authenticate(checkQuotaHanlder), nil, "check_quota", nil)
}

func queryQuota(ctx context.Context, projectId string) (*jsonutils.JSONDict, error) {
	ret := jsonutils.NewDict()

	quota := _manager.newQuota()
	err := _manager.GetQuota(ctx, projectId, quota)
	if err != nil {
		return nil, err
	}
	usage := _manager.newQuota()
	err = usage.FetchUsage(projectId)
	if err != nil {
		return nil, err
	}
	pending := _manager.newQuota()
	err = _manager.GetPendingUsage(ctx, projectId, pending)
	if err != nil {
		return nil, err
	}

	ret.Update(quota.ToJSON(""))
	ret.Update(usage.ToJSON("usage"))
	ret.Update(pending.ToJSON("pending"))

	return ret, nil
}

func getQuotaHanlder(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx)
	params := appctx.AppContextParams(ctx)

	projectId := params["<tenantid>"]
	if len(projectId) == 0 {
		projectId = userCred.GetProjectId()
	} else {
		if !userCred.IsSystemAdmin() {
			httperrors.ForbiddenError(w, "not allow to query quota")
			return
		}
		tenant, err := db.TenantCacheManager.FetchTenantByIdOrName(ctx, projectId)
		if err != nil {
			if err == sql.ErrNoRows {
				httperrors.TenantNotFoundError(w, "project %s not found", projectId)
				return
			} else {
				httperrors.GeneralServerError(w, err)
				return
			}
		}
		projectId = tenant.Id
	}
	quota, err := queryQuota(ctx, projectId)

	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	body := jsonutils.NewDict()
	body.Add(quota, "quotas")

	appsrv.SendJSON(w, body)
}

func setQuotaHanlder(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx)
	if !userCred.IsSystemAdmin() {
		httperrors.ForbiddenError(w, "not allow to set quota")
		return
	}
	params := appctx.AppContextParams(ctx)
	projectId := params["<tenantid>"]
	if len(projectId) == 0 {
		projectId = userCred.GetProjectId()
	} else {
		tenant, err := db.TenantCacheManager.FetchTenantByIdOrName(ctx, projectId)
		if err != nil {
			if err == sql.ErrNoRows {
				httperrors.TenantNotFoundError(w, "project %s not found", projectId)
				return
			} else {
				httperrors.GeneralServerError(w, err)
				return
			}
		}
		projectId = tenant.Id
	}
	body, err := appsrv.FetchJSON(r)
	if err != nil {
		log.Errorf("Fail to decode JSON request body: %s", err)
		httperrors.InvalidInputError(w, "fail to decode body")
		return
	}
	quota := _manager.newQuota()
	err = body.Unmarshal(quota, "quotas")
	if err != nil {
		log.Errorf("Fail to decode JSON request body: %s", err)
		httperrors.InvalidInputError(w, "fail to decode body")
		return
	}
	oquota := _manager.newQuota()
	err = _manager.GetQuota(ctx, projectId, oquota)
	if err != nil {
		log.Errorf("get quota fail %s", err)
		httperrors.GeneralServerError(w, err)
		return
	}
	oquota.Update(quota)
	err = _manager.SetQuota(ctx, userCred, projectId, oquota)
	if err != nil {
		log.Errorf("set quota fail %s", err)
		httperrors.GeneralServerError(w, err)
		return
	}
	rbody := jsonutils.NewDict()
	rbody.Add(oquota.ToJSON(""), "quotas")
	appsrv.SendJSON(w, rbody)
}

func checkQuotaHanlder(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx)
	if !userCred.IsSystemAdmin() {
		httperrors.ForbiddenError(w, "not allow to set quota")
		return
	}
	params := appctx.AppContextParams(ctx)
	projectId := params["<tenantid>"]
	if len(projectId) == 0 {
		projectId = userCred.GetProjectId()
	} else {
		tenant, err := db.TenantCacheManager.FetchTenantByIdOrName(ctx, projectId)
		if err != nil {
			if err == sql.ErrNoRows {
				httperrors.TenantNotFoundError(w, "project %s not found", projectId)
				return
			} else {
				httperrors.GeneralServerError(w, err)
				return
			}
		}
		projectId = tenant.Id
	}
	body, err := appsrv.FetchJSON(r)
	quota := _manager.newQuota()
	err = body.Unmarshal(quota, "quotas")
	if err != nil {
		log.Errorf("Fail to decode JSON request body: %s", err)
		httperrors.InvalidInputError(w, "fail to decode body")
		return
	}
	used, err := _manager.CheckQuota(ctx, userCred, projectId, quota)
	if err != nil {
		httperrors.OutOfQuotaError(w, "Out of quota: %s", err)
		return
	}
	rbody := jsonutils.NewDict()
	rbody.Add(used.ToJSON(""), "quotas")
	appsrv.SendJSON(w, rbody)
}
