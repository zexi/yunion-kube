package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	//"k8s.io/helm/pkg/helm"
	helmchart "k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/repo"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	//yerrors "yunion.io/x/pkg/util/errors"
	//"yunion.io/x/pkg/util/workqueue"

	"yunion.io/x/yunion-kube/pkg/helm/data"
	"yunion.io/x/yunion-kube/pkg/helm/data/cache/common"
	cacherepo "yunion.io/x/yunion-kube/pkg/helm/data/cache/repo"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/types/helm"
)

type cachedCharts struct {
	allCharts map[string][]*helm.ChartPackage
	rwm       *sync.RWMutex
}

type ChartResult struct {
	Repo  string             `json:"repo"`
	Chart *repo.ChartVersion `json:"chart"`
}

func (r ChartResult) RepoChartPath() string {
	return fmt.Sprintf("%s/%s", r.Repo, r.Chart.Metadata.Name)
}

// NewCachedCharts returns a new data.ICharts implementation
func NewCachedCharts() data.ICharts {
	return &cachedCharts{
		rwm:       new(sync.RWMutex),
		allCharts: make(map[string][]*helm.ChartPackage),
	}
}

func (c *cachedCharts) ChartFromRepo(repo, name string) (*helm.ChartPackage, error) {
	c.rwm.RLock()
	defer c.rwm.RUnlock()
	log.Infof("====ChartFromRepo:all: %#v, repo: %#v", c.allCharts, c.allCharts[repo])
	if c.allCharts[repo] != nil {
		chart, err := GetLatestChartVersion(c.allCharts[repo], name)
		if err != nil {
			return nil, err
		}
		return chart, nil
	}
	return nil, fmt.Errorf("no charts found for repo %q", repo)
}

func (c *cachedCharts) ChartVersionFromRepo(repo, name, version string) (*helm.ChartPackage, error) {
	c.rwm.RLock()
	defer c.rwm.RUnlock()
	if c.allCharts[repo] != nil {
		chart, err := GetChartVersion(c.allCharts[repo], name, version)
		if err != nil {
			return nil, err
		}
		return chart, nil
	}
	return nil, fmt.Errorf("no charts found for repo %q", repo)
}

func (c *cachedCharts) ChartVersionsFromRepo(repo, name string) ([]*helm.ChartPackage, error) {
	c.rwm.RLock()
	defer c.rwm.RUnlock()
	if c.allCharts[repo] != nil {
		charts, err := GetChartVersions(c.allCharts[repo], name)
		if err != nil {
			return nil, err
		}
		return charts, nil
	}
	return nil, fmt.Errorf("no charts found for repo %q", repo)
}

func (c *cachedCharts) AllFromRepo(repo string) ([]*helm.ChartPackage, error) {
	c.rwm.RLock()
	defer c.rwm.RUnlock()
	if c.allCharts[repo] != nil {
		return c.allCharts[repo], nil
	}
	return nil, fmt.Errorf("no charts found for repo %q", repo)
}

func (c *cachedCharts) All() ([]*helm.ChartPackage, error) {
	c.rwm.RLock()
	defer c.rwm.RUnlock()
	var allCharts []*helm.ChartPackage

	repos, err := models.RepoManager.ListRepos()
	if err != nil {
		return nil, err
	}
	for _, repo := range repos {
		var charts []*helm.ChartPackage
		for _, chart := range c.allCharts[repo.Name] {
			charts = append(charts, chart)
		}
		allCharts = append(allCharts, charts...)
	}
	return allCharts, nil
}

func (c *cachedCharts) Search(params jsonutils.JSONObject) ([]*helm.ChartPackage, error) {
	c.rwm.RLock()
	defer c.rwm.RUnlock()
	var ret []*helm.ChartPackage
	charts, err := c.All()
	if err != nil {
		return nil, err
	}
	sName, _ := params.GetString("name")
	for _, chart := range charts {
		if strings.Contains(chart.Metadata.Name, sName) {
			ret = append(ret, chart)
		}
	}
	return ret, nil
}

// DeleteChart is the interface implementation for data.ICharts
// It delete Chart from memory
// NODE: This method does not delete from filesystem for now.
// It is used to test single chart refresh
//func (c *cachedCharts) DeleteChart(repoName string, chartName string, chartVersion string) error {
//repo, err := models.RepoManager.FetchRepoByIdOrName("", repoName)
//if err != nil {
//return fmt.Errorf("[DeleteChart] Fetch repo %q error: %v", repoName, err)
//}

//c.rwm.Lock()

//for k, chart := range c.allCharts[repoName] {
//if chart.Metadata.Name == chartName && chart.Metadata.Version == chartVersion {
//log.Infof("chart deleted, path: '%s - %s - %s'", charthelper.DataDirBase(), chartName, chartVersion)
//c.allCharts[repo.Name] = append(c.allCharts[repo.Name][:k], c.allCharts[repo.Name][k+1:]...)
//break
//}
//}

//c.rwm.Unlock()

//return nil
//}

//func (c *cachedCharts) RefreshChart(repoName string, chartName string) error {
//log.Infof("Using cache directory: %q", charthelper.DataDirBase())

//repo, err := models.RepoManager.FetchRepoByIdOrName("", repoName)
//if err != nil {
//return fmt.Errorf("[RefreshChart] fetch repo %q error: %v", repoName, err)
//}
//charts, err := repohelper.GetChartsFromRepoIndexFile(repo)
//if err != nil {
//return err
//}

//didUpdate := false
//var alreadyExists bool
//for _, chartFromIndex := range charts {
//if chartFromIndex.Name == chartName {
//didUpdate = true
//ch := make(chan chanItem, len(charts))
//defer close(ch)
//go processChartMetadata(chartFromIndex, repo.Url, ch)
//it := <-ch
//if it.err == nil {
//c.rwm.Lock()
//// find the key
//alreadyExists = false
//for k, chart := range c.allCharts[repo.Name] {
//if chart.Name == it.chart.Name && chart.Version == it.chart.Version {
//c.allCharts[repo.Name][k] = it.chart
//alreadyExists = true
//break
//}
//}
//if alreadyExists == false {
//c.allCharts[repo.Name] = append(c.allCharts[repo.Name], it.chart)
//}
//c.rwm.Unlock()
//} else {
//return it.err
//}
//}
//}

//if didUpdate == false {
//return fmt.Errorf("no chart %q found for repo %q, url %q", chartName, repo.Name, repo.Url)
//}
//return nil
//}

type SpotguideFile struct {
	Options []SpotguideOptions `json:"options"`
}

type SpotguideOptions struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Default bool   `json:"default"`
	Info    string `json:"info"`
	Key     string `json:"key"`
}

type ChartDetail struct {
	Name    string `json:"name"`
	Repo    string `json:"repo"`
	Chart   *repo.ChartVersion
	Values  string             `json:"values"`
	Readme  string             `json:"readme"`
	Options []SpotguideOptions `json:"options"`
}

type ChartQuery struct {
	Name       string `json:"name"`
	Repo       string `json:"repo"`
	RepoUrl    string `json:"repo_url"`
	AllVersion bool   `json:"all_version"`
	Keyword    string `json:"keyword"`
}

type RepoFile struct {
	*repo.RepoFile
}

func (r *RepoFile) Get(name string) (*repo.Entry, error) {
	var ret *repo.Entry
	for _, rf := range r.Repositories {
		if rf.Name == name {
			ret = rf
		}
	}
	return ret, nil
}

func LoadRepositoriesFile() (*repo.RepoFile, error) {
	repoPath := fmt.Sprintf("%s/repository/repositories.yaml", common.GenerateHelmRepoPath(""))
	log.Infof("Helm repo path: %s", repoPath)
	f, err := repo.LoadRepositoriesFile(repoPath)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func ChartShowDetails(repoName, chartName, chartVersion string) (*ChartDetail, error) {
	f, err := LoadRepositoriesFile()
	if err != nil {
		return nil, err
	}
	if len(f.Repositories) == 0 {
		return nil, nil
	}

	dstRepo, err := (&RepoFile{f}).Get(repoName)
	if err != nil {
		return nil, err
	}
	if dstRepo == nil {
		return nil, nil
	}
	i, errIdx := repo.LoadIndexFile(dstRepo.Cache)
	if errIdx != nil {
		return nil, errIdx
	}
	dstChart, err := i.Get(chartName, chartVersion)
	if err != nil {
		return nil, err
	}
	if dstChart == nil {
		return nil, nil
	}
	chartSrc := dstChart.URLs[0]
	log.Debugf("Get chartsource: %s", chartSrc)
	reader, err := downloadFile(chartSrc)
	if err != nil {
		return nil, err
	}
	valuesStr, err := getChartFile(reader, "values.yaml")
	if err != nil {
		return nil, err
	}
	options, err := getChartOption(reader)
	if err != nil {
		return nil, err
	}
	readmeStr, err := getChartFile(reader, "README.md")
	if err != nil {
		return nil, err
	}
	chartD := &ChartDetail{
		Name:    chartName,
		Repo:    repoName,
		Chart:   dstChart,
		Values:  valuesStr,
		Readme:  readmeStr,
		Options: options.Options,
	}
	return chartD, nil
}

func ChartsList(query *ChartQuery) ([]*ChartResult, error) {
	f, err := LoadRepositoriesFile()
	if err != nil {
		return nil, err
	}
	if len(f.Repositories) == 0 {
		return nil, nil
	}

	trans := func(repoName string, versions repo.ChartVersions) []*ChartResult {
		ret := make([]*ChartResult, 0)
		for _, c := range versions {
			ret = append(ret, &ChartResult{
				Repo:  repoName,
				Chart: c,
			})
		}
		return ret
	}

	cl := make([]*ChartResult, 0)
	for _, r := range f.Repositories {
		log.Debugf("Load repository: %s", r.Name)
		i, errIdx := repo.LoadIndexFile(r.Cache)
		if errIdx != nil {
			return nil, errIdx
		}
		if query.Repo != "" {
			repoMatched := query.Repo == r.Name
			if !repoMatched {
				continue
			}
		}
		if query.RepoUrl != "" {
			repoUrlMatched := strings.TrimRight(query.RepoUrl, "/") == strings.TrimRight(r.URL, "/")
			if !repoUrlMatched {
				continue
			}
		}
		for n := range i.Entries {
			nameMatched, _ := regexp.MatchString(query.Name, n)
			kwString := strings.ToLower(strings.Join(i.Entries[n][0].Keywords, " "))
			kwMatched, _ := regexp.MatchString(query.Keyword, kwString)
			if (nameMatched || query.Name == "") && (kwMatched || query.Keyword == "") {
				if !query.AllVersion {
					cl = append(cl, &ChartResult{
						Repo:  r.Name,
						Chart: i.Entries[n][0],
					})
				} else {
					cl = append(cl, trans(r.Name, i.Entries[n])...)
				}
			}
		}
	}
	return cl, nil
}

func DownloadChartFromRepo(name, version string) (string, error) {
	return downloadChartFromRepo(name, version, common.GenerateHelmRepoPath(""))
}

func downloadChartFromRepo(name, version, path string) (string, error) {
	settings := common.CreateEnvSettings(path)
	dl := downloader.ChartDownloader{
		HelmHome: settings.Home,
		Getters:  getter.All(settings),
	}
	if _, err := os.Stat(settings.Home.Archive()); os.IsNotExist(err) {
		log.Infof("Creating %q directory", settings.Home.Archive())
		os.MkdirAll(settings.Home.Archive(), 0744)
	}

	log.Infof("Downloading helm chart %q to %q, version: %q", name, settings.Home.Archive(), version)
	filename, _, err := dl.DownloadTo(name, version, settings.Home.Archive())
	if err == nil {
		lname, err := filepath.Abs(filename)
		if err != nil {
			return filename, fmt.Errorf("Could't create absolute path from %q", filename)
		}
		log.Debugf("Fetched helm chart %q to %q", name, filename)
		return lname, nil
	}
	return filename, fmt.Errorf("Failed to download chart %q: %v", name, err)
}

// Refresh implementation for data.ICharts
// It refreshes cached data for all repository and chart data
func (c *cachedCharts) Refresh() error {
	var updatedCharts = make(map[string][]*helm.ChartPackage)

	repos, err := models.RepoManager.ListRepos()
	if err != nil {
		log.Errorf("Get all repos error: %v", err)
		return err
	}

	for _, repo := range repos {
		err := repo.AddToBackend()
		_, ok := err.(cacherepo.ErrorRepoAlreadyAdded)
		if err != nil && !ok {
			log.Errorf("Error on add repo %q to backend: %v", repo.Name, err)
			continue
		}
		err = repo.DoSync()
		if err != nil {
			log.Errorf("Sync repo %q error: %v", repo.Name, err)
			continue
		}
		charts, err := ChartsList(&ChartQuery{Repo: repo.Name})
		if err != nil {
			log.Errorf("Error on refresh charts from repo: %q, error: %v", repo.Name, err)
			continue
		}

		// Process elements in index
		var chartsWithData []*helm.ChartPackage
		// Buffered channel
		ch := make(chan chanItem, len(charts))
		defer close(ch)

		// parallellize processing
		for _, chart := range charts {
			go processChartMetadata(chart, repo.Url, ch)
		}
		// Channel drain
		for range charts {
			it := <-ch
			// Only append the ones that have not failed
			if it.err == nil {
				chartsWithData = append(chartsWithData,
					&helm.ChartPackage{
						Chart: it.chart,
						Repo:  repo.Name,
					})
			}
		}
		updatedCharts[repo.Name] = chartsWithData
	}

	// Update the stored cache with the new elements if everything went well
	c.rwm.Lock()
	c.allCharts = updatedCharts
	c.rwm.Unlock()
	return nil
}

type chanItem struct {
	chart *helmchart.Chart
	err   error
}

func checkDependencies(ch *helmchart.Chart, reqs *chartutil.Requirements) error {
	missing := []string{}

	deps := ch.GetDependencies()
	for _, r := range reqs.Dependencies {
		found := false
		for _, d := range deps {
			if d.Metadata.Name == r.Name {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, r.Name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("found in requirements.yaml, but missing in charts/ directory: %s", strings.Join(missing, ", "))
	}
	return nil
}

var tokens = make(chan struct{}, 25)

func processChartMetadata(chart *ChartResult, repoUrl string, out chan<- chanItem) {
	tokens <- struct{}{}
	defer func() {
		<-tokens
	}()

	var it chanItem

	downloadedChartPath, err := downloadChartFromRepo(chart.RepoChartPath(), chart.Chart.Version, common.GenerateHelmRepoPath(""))
	if err != nil {
		log.Errorf("Download chart error: %v", err)
		it.err = err
		out <- it
		return
	}
	chartRequested, err := chartutil.Load(downloadedChartPath)
	if err != nil {
		it.err = fmt.Errorf("Error loading chart: %v", err)
		out <- it
		return
	}
	if req, err := chartutil.LoadRequirements(chartRequested); err == nil {
		if err := checkDependencies(chartRequested, req); err != nil {
			it.err = err
			out <- it
			return
		}
	} else if err != chartutil.ErrRequirementsNotFound {
		it.err = fmt.Errorf("cannot load requirements: %v", err)
		return
	}

	//// Extra files. Skipped if the directory exists
	//dataExists, err := charthelper.ChartDataExists(chart)
	//if err != nil {
	//it.err = err
	//out <- it
	//return
	//}

	//if !dataExists {
	//log.Infof("Local cache missing, name: %s, version: %s", chart.Name, chart.Version)
	//err := charthelper.DownloadAndExtractChartTarball(chart, repoUrl)
	//if err != nil {
	//log.Errorf("Error on DownloadAndExtractChartTarball: %v", err)
	//it.err = err
	//out <- it
	//return
	//}
	//// if we have a problem processing an image it will fallback to the default one
	//err = charthelper.DownloadAndProcessChartIcon(chart)
	//if err != nil {
	//log.Errorf("DownloadAndProcessChartIcon error: %v", err)
	//}
	//}
	it.chart = chartRequested
	out <- it
}
