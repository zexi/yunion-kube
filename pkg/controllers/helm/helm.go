package helm

import (
	"time"

	"k8s.io/helm/pkg/repo"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/options"
	"yunion.io/x/yunion-kube/pkg/types/helm"
)

var OfficialRepos = []helm.Repo{
	{
		Name:   "stable",
		Url:    "https://kubernetes-charts.storage.googleapis.com",
		Source: "https://github.com/kubernetes/charts/tree/master/stable",
	},
	{
		Name:   "incubator",
		Url:    "https://kubernetes-charts-incubator.storage.googleapis.com",
		Source: "https://github.com/kubernetes/charts/tree/master/incubator",
	},
}

func Start() {
	var err error
	repos, err := models.RepoManager.ListRepos()
	if err != nil {
		log.Fatalf("List all repos error: %v", err)
	}
	entries := make([]*repo.Entry, 0)
	for _, obj := range repos {
		entries = append(entries, obj.ToEntry())
	}
	err = data.SetupRepoBackendManager(entries)
	if err != nil {
		log.Fatalf("Setup RepoController error: %v", err)
	}
	startRepoRefresh(data.RepoBackendManager)
}

func startRepoRefresh(man *data.RepoCacheBackend) {
	tick := time.Tick(time.Duration(options.Options.RepoRefreshDuration) * time.Minute)
	for {
		select {
		case <-tick:
			doRepoRefresh(man)
		}
	}
}

func doRepoRefresh(man *data.RepoCacheBackend) {
	repos, err := models.RepoManager.ListRepos()
	if err != nil {
		log.Errorf("List all repos error: %v", err)
		return
	}
	names := []string{}
	for _, r := range repos {
		names = append(names, r.Name)
	}
	err = man.ReposUpdate(names)
	if err != nil {
		log.Errorf("Update all repos error: %v", err)
	}
	log.Debugf("Finish refresh repos %v", names)
}
