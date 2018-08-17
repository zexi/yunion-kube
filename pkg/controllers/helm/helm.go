package helm

import (
	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/helm/data"
	"yunion.io/x/yunion-kube/pkg/helm/data/cache"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/types/helm"
)

func setupRepos(repos []helm.Repo) {
	if err := models.RepoManager.CreateRepos(repos); err != nil {
		log.Errorf("Create default repositories error: %v", err)
	}
}

func setupCharts() data.ICharts {
	chartsImpl := cache.NewCachedCharts()
	// Run foreground repository refresh
	chartsImpl.Refresh()
	// Setup background index refreshes
	cacheRefreshInterval := 3600
	periodcRefresh := cache.NewRefreshChartsData(chartsImpl, freshness, "refresh-charts", false)
	toDo := []jobs.Periodic{periodcRefresh}
	jobs.DoPeriodic(toDo)

	return chartsImpl
}

func Start() {

}
