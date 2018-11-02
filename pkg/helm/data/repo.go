package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/repo"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/util/errors"
	"yunion.io/x/pkg/util/workqueue"

	"yunion.io/x/yunion-kube/pkg/helm/data/common"
	"yunion.io/x/yunion-kube/pkg/options"
	"yunion.io/x/yunion-kube/pkg/types/helm"
)

type IRepoBackend interface {
	Add(repo *repo.Entry) error
	Delete(repoName string) error
	Modify(repoName string, newRepo *repo.Entry) error
	Show(repoName string) (*repo.Entry, error)
	Update(repoName string) error
}

type ErrorRepoNotFound struct {
	repo string
}

func (e ErrorRepoNotFound) Error() string {
	return fmt.Sprintf("Helm repository not found: %q", e.repo)
}

type ErrorRepoAlreadyAdded struct {
	repo string
}

func (e ErrorRepoAlreadyAdded) Error() string {
	return fmt.Sprintf("Repo %q already added", e.repo)
}

var RepoBackendManager *RepoCacheBackend

func SetupRepoBackendManager(repos []*repo.Entry) error {
	common.EnsureStateStoreDir(options.Options.HelmDataDir)
	RepoBackendManager = newRepoCacheBackend()
	for _, r := range repos {
		RepoBackendManager.Add(r)
	}
	return nil
}

type ChartResult struct {
	Repo  string             `json:"repo"`
	Chart *repo.ChartVersion `json:"chart"`
}

// RepoCacheBackend implements data.IReposBackend
type RepoCacheBackend struct{}

func newRepoCacheBackend() *RepoCacheBackend {
	backend := &RepoCacheBackend{}
	return backend
}

func (r *RepoCacheBackend) Add(repo *repo.Entry) error {
	if oldRepo, _ := r.Show(repo.Name); oldRepo != nil {
		return httperrors.NewDuplicateNameError("Repo %s already added", repo.Name)
	}
	err := repoAddLocal(repo)
	if err != nil {
		return err
	}
	return nil
}

func (r *RepoCacheBackend) Delete(repoName string) error {
	err := repoDeleteLocal(repoName)
	if err != nil {
		return err
	}
	return nil
}

func (r *RepoCacheBackend) Modify(repoName string, newRepo *repo.Entry) error {
	err := repoModifyLocal(repoName, newRepo)
	if err != nil {
		return err
	}
	return nil
}

func (r *RepoCacheBackend) Show(repoName string) (*repo.Entry, error) {
	return repoShowLocal(repoName)
}

func (r *RepoCacheBackend) Update(repoName string) error {
	return repoUpdateLocal(repoName)
}

func repoAddLocal(hRepo *repo.Entry) error {
	settings := common.CreateEnvSettings(common.GenerateHelmRepoPath(""))
	repoFile := settings.Home.RepositoryFile()
	var f *repo.RepoFile
	if _, err := os.Stat(repoFile); err != nil {
		log.Infof("Creating %s", repoFile)
		err = os.MkdirAll(filepath.Dir(repoFile), 0755)
		if err != nil {
			return err
		}
		f = repo.NewRepoFile()
	} else {
		f, err = repo.LoadRepositoriesFile(repoFile)
		if err != nil {
			return fmt.Errorf("Can't load repositories file %s: %v", repoFile, err)
		}
		log.Debugf("Prfile %q loaded", repoFile)
	}

	for _, n := range f.Repositories {
		if n.Name == hRepo.Name {
			return ErrorRepoAlreadyAdded{n.Name}
		}
	}

	c := repo.Entry{
		Name:  hRepo.Name,
		URL:   hRepo.URL,
		Cache: settings.Home.CacheIndex(hRepo.Name),
	}
	r, err := repo.NewChartRepository(&c, getter.All(settings))
	if err != nil {
		return httperrors.NewBadRequestError("Can't create a new chart repo %s: %v", hRepo.Name, err)
	}
	log.Debugf("New repo added: %q", hRepo.Name)
	err = os.MkdirAll(filepath.Dir(r.Config.Cache), 0755)
	if err != nil {
		return fmt.Errorf("Make cache dir %q: %v", r.Config.Cache, err)
	}

	errIdx := r.DownloadIndexFile("")
	if errIdx != nil {
		return httperrors.NewBadRequestError("Repo %q index download failed: %v", hRepo.Name, errIdx)
	}
	f.Add(&c)
	if errW := f.WriteFile(repoFile, 0644); errW != nil {
		return fmt.Errorf("Can't write helm repo profile %q: %v", repoFile, errW)
	}
	return nil
}

func repoDeleteLocal(repoName string) error {
	repoPath := common.GenerateHelmRepoPath("")
	settings := common.CreateEnvSettings(repoPath)
	repoFile := settings.Home.RepositoryFile()

	r, err := repo.LoadRepositoriesFile(repoFile)
	if err != nil {
		return err
	}

	if !r.Remove(repoName) {
		return ErrorRepoNotFound{repoName}
	}
	if err := r.WriteFile(repoFile, 0644); err != nil {
		return err
	}

	if _, err := os.Stat(settings.Home.CacheIndex(repoName)); err == nil {
		err = os.Remove(settings.Home.CacheIndex(repoName))
		if err != nil {
			return err
		}
	}
	return nil
}

func repoModifyLocal(repoName string, newRepo *repo.Entry) error {
	repoPath := common.GenerateHelmRepoPath("")
	settings := common.CreateEnvSettings(repoPath)
	repoFile := settings.Home.RepositoryFile()
	log.Debugf("New repo content: %#v", newRepo)

	f, err := repo.LoadRepositoriesFile(repoFile)
	if err != nil {
		return err
	}

	if !f.Has(repoName) {
		return ErrorRepoNotFound{repoName}
	}

	f.Update(newRepo)

	if errW := f.WriteFile(repoFile, 0644); errW != nil {
		return fmt.Errorf("Can't write helm repo profile: %v", err)
	}
	return nil
}

func (man *RepoCacheBackend) ReposUpdate(repos []string) error {
	errsChannel := make(chan error, len(repos))
	uf := func(i int) {
		err := man.Update(repos[i])
		if err != nil {
			errsChannel <- err
		}
	}
	workqueue.Parallelize(4, len(repos), uf)
	if len(errsChannel) > 0 {
		errs := make([]error, 0)
		length := len(errsChannel)
		for ; length > 0; length-- {
			errs = append(errs, <-errsChannel)
		}
		return errors.NewAggregate(errs)
	}
	return nil
}

func reposListLocal() ([]*repo.Entry, error) {
	repoPath := filepath.Join(common.GenerateHelmRepoPath(""), "repository", "repositories.yaml")
	log.Debugf("[ReposList] helm repo path: %q", repoPath)

	f, err := repo.LoadRepositoriesFile(repoPath)
	if err != nil {
		return nil, err
	}
	if len(f.Repositories) == 0 {
		return make([]*repo.Entry, 0), nil
	}
	return f.Repositories, nil
}

func repoShowLocal(name string) (*repo.Entry, error) {
	repos, err := reposListLocal()
	if err != nil {
		return nil, err
	}
	for _, rf := range repos {
		if rf.Name == name {
			return rf, nil
		}
	}
	return nil, ErrorRepoNotFound{name}
}

func repoUpdateLocal(repoName string) error {
	repoPath := common.GenerateHelmRepoPath("")
	settings := common.CreateEnvSettings(repoPath)
	repoFile := settings.Home.RepositoryFile()

	f, err := repo.LoadRepositoriesFile(repoFile)
	if err != nil {
		return fmt.Errorf("Load chart repo: %v", err)
	}

	for _, cfg := range f.Repositories {
		if cfg.Name == repoName {
			log.Debugf("Updating %q chart repo url: %s", cfg.Name, cfg.URL)
			c, err := repo.NewChartRepository(cfg, getter.All(settings))
			if err != nil {
				return fmt.Errorf("Can't get chart repo %q: %v", repoName, err)
			}
			errIdx := c.DownloadIndexFile("")
			if errIdx != nil {
				return fmt.Errorf("Unable to get an update from the %q chart repo (%s): \n\t%v\n", cfg.Name, cfg.URL, errIdx)
			}
			log.Debugf("Successfully update %q chart repo", cfg.Name)
			return nil
		}
	}
	return ErrorRepoNotFound{repoName}
}

func newChartResult(repoName string, chart *repo.ChartVersion) *ChartResult {
	return &ChartResult{Repo: repoName, Chart: chart}
}

func trans(repoName string, versions repo.ChartVersions) []*ChartResult {
	ret := make([]*ChartResult, 0)
	for _, c := range versions {
		ret = append(ret, newChartResult(repoName, c))
	}
	return ret
}

func repoEntryCharts(r *repo.Entry, allVersion bool) (*chartList, error) {
	indexes, err := repo.LoadIndexFile(r.Cache)
	if err != nil {
		return nil, err
	}

	chartsList := make([]*ChartResult, 0)
	for chartName := range indexes.Entries {
		log.Debugf("repo %q chart: %q", r.Name, chartName)
		if !allVersion {
			chartsList = append(chartsList, newChartResult(r.Name, indexes.Entries[chartName][0]))
		} else {
			chartsList = append(chartsList, trans(r.Name, indexes.Entries[chartName])...)
		}
	}
	return newChartList(chartsList), nil
}

func repoCharts(repoName string, allVersion bool) (*chartList, error) {
	r, err := repoShowLocal(repoName)
	if err != nil {
		return nil, err
	}
	return repoEntryCharts(r, allVersion)
}

func allRepoCharts(q *helm.ChartQuery, allVersion bool) (*chartList, error) {
	repos, err := reposListLocal()
	if err != nil {
		return nil, err
	}
	ret := newChartList([]*ChartResult{})
	for _, r := range repos {
		// filter by repo
		if q.Repo != "" && q.Repo != r.Name {
			continue
		}
		// filter by repoUrl
		if q.RepoUrl != "" {
			matched := strings.TrimRight(q.RepoUrl, "/") == strings.TrimRight(r.URL, "/")
			if !matched {
				continue
			}
		}
		charts, err := repoEntryCharts(r, allVersion)
		if err != nil {
			log.Errorf("Get repo %s, url %s charts error: %v", r.Name, r.URL, err)
			continue
		}
		ret = ret.Add(charts)
	}
	return ret, nil
}
