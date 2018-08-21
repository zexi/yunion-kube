package repo

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/repo"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/helm/data/cache/common"
)

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

var BackendManager *RepoCacheBackend

func init() {
	BackendManager = &RepoCacheBackend{}
}

type RepoCacheBackend struct{}

func (r *RepoCacheBackend) Add(repo *repo.Entry) error {
	return RepoAdd(repo)
}

func (r *RepoCacheBackend) Delete(repoName string) error {
	return RepoDelete(repoName)
}

func (r *RepoCacheBackend) Modify(repoName string, newRepo *repo.Entry) error {
	return RepoModify(repoName, newRepo)
}

func (r *RepoCacheBackend) Show(repoName string) (*repo.Entry, error) {
	return RepoShow(repoName)
}

func (r *RepoCacheBackend) Update(repoName string) error {
	return RepoUpdate(repoName)
}

func RepoAdd(hRepo *repo.Entry) error {
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
			return fmt.Errorf("Can't create a new chart repo: %#v", hRepo)
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
		return fmt.Errorf("Can't create a new chart repo: %q", hRepo.Name)
	}
	log.Debugf("New repo added: %q", hRepo.Name)
	err = os.MkdirAll(filepath.Dir(r.Config.Cache), 0755)
	if err != nil {
		return fmt.Errorf("Make cache dir %q: %v", r.Config.Cache, err)
	}

	errIdx := r.DownloadIndexFile("")
	if errIdx != nil {
		return fmt.Errorf("Repo %q index download failed: %v", hRepo.Name, errIdx)
	}
	f.Add(&c)
	if errW := f.WriteFile(repoFile, 0644); errW != nil {
		return fmt.Errorf("Can't write helm repo profile %q: %v", repoFile, errW)
	}
	return nil
}

func RepoDelete(repoName string) error {
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

func RepoModify(repoName string, newRepo *repo.Entry) error {
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

//func ReposUpdate(repos []string) error {
//errsChannel := make(chan error, len(repos))
//uf := func(i int) {
//err := RepoUpdate(repos[i])
//if err != nil {
//errsChannel <- err
//}
//}
//workqueue.Parallellize(4, len(repos), uf)
//if len(errsChannel) > 0 {
//errs := make([]error, 0)
//length := len(errsChannel)
//for ; length > 0; length-- {
//errs = append(errs, <-errsChannel)
//}
//return yerrors.NewAggregate(errs)
//}
//return nil
//}

func ReposList() ([]*repo.Entry, error) {
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

func RepoShow(name string) (*repo.Entry, error) {
	repos, err := ReposList()
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

func RepoUpdate(repoName string) error {
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
