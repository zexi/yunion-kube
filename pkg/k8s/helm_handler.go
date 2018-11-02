package k8s

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"

	helmclient "yunion.io/x/yunion-kube/pkg/helm/client"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/resources/chart"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	helmtypes "yunion.io/x/yunion-kube/pkg/types/helm"
)

const (
	DefaultTillerImage = "yunion/tiller:v2.9.0"
)

func AddHelmDispatcher(prefix string, app *appsrv.Application) {
	log.Infof("Register helm dispatcher handler")

	// handle helm tiller install
	app.AddHandler("POST",
		fmt.Sprintf("%s/tiller", prefix),
		auth.Authenticate(handleHelmTillerInstall))

	// handle helm charts actions
	app.AddHandler("GET",
		fmt.Sprintf("%s/charts/<name>", prefix),
		auth.Authenticate(chartShowHandler))

	app.AddHandler("GET",
		fmt.Sprintf("%s/charts", prefix),
		auth.Authenticate(chartlistHandler))
}

func handleHelmTillerInstall(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, query, data := _fetchEnv(ctx, w, r)
	body := data.(*jsonutils.JSONDict)
	if img, _ := body.GetString("tiller_image"); img == "" {
		body.Set("tiller_image", jsonutils.NewString(DefaultTillerImage))
	}
	if sa, _ := body.GetString("service_account"); sa == "" {
		body.Set("service_account", jsonutils.NewString("tiller"))
	}
	if ns, _ := body.GetString("namespace"); ns == "" {
		body.Set("namespace", jsonutils.NewString("kube-system"))
	}
	request, err := NewCloudK8sRequest(ctx, query.(*jsonutils.JSONDict), body)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	cli := request.GetK8sClient()
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	opt := helmclient.InstallOption{}
	err = body.Unmarshal(&opt)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	err = helmclient.Install(cli, &opt)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func getQuery(ctx context.Context, w http.ResponseWriter, r *http.Request) (*helmtypes.ChartQuery, *dataselect.DataSelectQuery, error) {
	_, query, _ := _fetchEnv(ctx, w, r)
	dsq := common.NewDataSelectQuery(query)
	var cq helmtypes.ChartQuery
	err := query.Unmarshal(&cq)
	if err != nil {
		return nil, nil, err
	}
	return &cq, dsq, nil
}

func chartlistHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, query, _ := _fetchEnv(ctx, w, r)
	if jsonutils.QueryBoolean(query, "all_version", false) {
		query.(*jsonutils.JSONDict).Set("all_version", jsonutils.JSONTrue)
	}
	cq, dsq, err := getQuery(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	list, err := chart.ChartManager.List(cq, dsq)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	SendJSON(w, common.ListResource2JSONWithKey(list, "charts"))
}

func chartShowHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, query, _ := _fetchEnv(ctx, w, r)
	repoName, _ := query.GetString("repo")
	if repoName == "" {
		httperrors.InvalidInputError(w, "repo not provided")
		return
	}
	chartName := params["<name>"]
	userCred := getUserCredential(ctx)
	repo, err := models.RepoManager.FetchRepoByIdOrName(userCred.GetProjectId(), repoName)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	version, _ := query.GetString("version")
	resp, err := chart.ChartManager.Show(repo.Name, chartName, version)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	SendJSON(w, wrapBody(resp, "chart"))
}
