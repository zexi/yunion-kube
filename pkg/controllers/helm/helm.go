package helm

import (
	"time"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/helm/data"
	"yunion.io/x/yunion-kube/pkg/helm/data/cache"
	"yunion.io/x/yunion-kube/pkg/helm/data/cache/common"
	"yunion.io/x/yunion-kube/pkg/helm/jobs"
	"yunion.io/x/yunion-kube/pkg/options"
)

var ChartController data.ICharts

//func setupRepos(repos []helm.Repo) {
//if err := models.RepoManager.CreateRepos(repos); err != nil {
//log.Errorf("Create default repositories error: %v", err)
//}
//}

func setupCharts() data.ICharts {
	common.EnsureStateStoreDir(options.Options.HelmDataDir)
	chartsImpl := cache.NewCachedCharts()
	// Run foreground repository refresh
	chartsImpl.Refresh()
	// Setup background index refreshes
	cacheRefreshInterval := 3600
	freshness := time.Duration(cacheRefreshInterval) * time.Second
	periodcRefresh := cache.NewRefreshChartsData(chartsImpl, freshness, "refresh-charts", false)
	toDo := []jobs.Periodic{periodcRefresh}
	jobs.DoPeriodic(toDo)

	return chartsImpl
}

func Start() {
	ChartController = setupCharts()
	log.Infof("Charts impl: %v", ChartController)
}
