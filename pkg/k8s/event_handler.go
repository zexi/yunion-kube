package k8s

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/models"
)

func AddEventHandler(prefix string, app *appsrv.Application) {
	prefix = fmt.Sprintf("%s/k8s_events", prefix)
	app.AddHandler2("GET", prefix, auth.Authenticate(listEvents), nil, "list_events", nil)
}

func listEvents(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	ownerId, scope, err := db.FetchUsageOwnerScope(ctx, userCred, query)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	input := new(api.EventListInput)
	if err := query.Unmarshal(input); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	events, err := models.GetEventManager().ListEvents(ctx, scope, userCred, ownerId, input)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	retJson := jsonutils.Marshal(events)

}
