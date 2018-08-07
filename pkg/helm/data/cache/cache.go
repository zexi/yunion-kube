package cache

import (
	"fmt"
	"strings"
	"sync"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"

	"yunion.io/yunion-kube/pkg/helm/data"
	"yunion.io/yunion-kube/pkg/helm/data/cache/charthelper"
	"yunion.io/yunion-kube/pkg/helm/data/helpers"
	"yunion.io/yunion-kube/pkg/models"
	"yunion.io/yunion-kube/pkg/types/helm"
)

type cachedCharts struct {
	allCharts map[string][]*helm.ChartPackage
	rwm       *sync.RWMutex
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
	if c.allCharts[repo] != nil {
		chart, err := helpers.GetLatestChartVersion(c.allCharts[repo], name)
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
		chart, err := helpers.GetChartVersion(c.allCharts[repo], name, version)
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
		charts, err := helpers.GetChartVersions(c.allCharts[repo], name)
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

	repos, err := models.ListRepos()
	if err != nil {
		return nil, err
	}
	for repo := range repos {
		var charts []*helm.ChartPackage
		for chart := range c.allCharts[repo.Name] {
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
		if strings.Contains(*chart.Name, sName) {
			ret = append(ret, chart)
		}
	}
	return ret, nil
}

// DeleteChart is the interface implementation for data.ICharts
// It delete Chart from memory
// NODE: This method does not delete from filesystem for now.
// It is used to test single chart refresh
func (c *cachedCharts) DeleteChart(repoName string, chartName string, chartVersion string) error {
	repo := models.RepoManager.FetchRepoByIdOrName("", repoName)
	if repo == nil {
		return fmt.Errorf("Repo %q not found", repoName)
	}

	c.rwm.Lock()

	for k, chart := range c.allCharts[repoName] {
		if chart.Name == chartName && chart.Version == chartVersion {
			log.Infof("chart deleted, path: '%s - %s - %s'", charthelper.DataDirBase(), chartName, chartVersion)
			c.allCharts[repo.Name] = append(c.allCharts[repo.Name][:k], c.allCharts[repo.Name][k+1:]...)
			break
		}
	}

	c.rwm.Unlock()

	return nil
}

func (c *cachedCharts) RefreshChart(repoName string, chartName string) error {
	log.Infof("Using cache directory: %q", charthelper.DataDirBase())

	repo := models.RepoManager.FetchRepoByIdOrName("", repoName)
	if repo == nil {
		return fmt.Errorf("Repo %q not found", repoName)
	}
	charts, err := repohelper.GetChartsFromRepoIndexFile(repo)
	if err != nil {
		return err
	}

	didUpdate := false
	var alreadyExists bool
	for _, chartFromIndex := range charts {
		if chartFromIndex.Name == chartName {
			didUpdate = true
			ch := make(chan chanItem, len(charts))
			defer close(ch)
			go processChartMetadata(chartFromIndex, repo.Url, ch)
			it := <-ch
			if it.err == nil {
				c.rwm.Lock()
				// find the key
				alreadyExists = false
				for k, chart := range c.allCharts[repo.Name] {
					if chart.Name == it.chart.Name && chart.Version == it.chart.Version {
						c.allCharts[repo.Name][k] = it.chart
						alreadyExists = true
						break
					}
				}
				if alreadyExists == fale {
					c.allCharts[repo.Name] = append(c.allCharts[repo.Name], it.charts)
				}
				c.rwm.Unlock()
			} else {
				return it.err
			}
		}
	}

	if didUpdate == false {
		return fmt.Errorf("no chart %q found for repo %q, url %q", chartName, repo.Name, repo.Url)
	}
	return nil
}

type chanItem struct {
	chart *helm.ChartPackage
	err   error
}

var tokens = make(chan struct{}, 25)

func processChartMetadata(chart *helm.ChartPackage, repoUrl string, out chan<- chanItem) {
	tokens <- struct{}{}
	defer func() {
		<-tokens
	}()

	var it chanItem
	it.chart = chart

	// Extra files. Skipped if the directory exists
	dataExists, err := charthelper.ChartDataExists(chart)
	if err != nil {
		it.err = err
		out <- it
		return
	}

	if !dataExists {
		log.Infof("Local cache missing, name: %s, version: %s", chart.Name, chart.Version)
		err := charthelper.DownloadAndExtractChartTarball(chart, repoUrl)
		if err != nil {
			log.Errorf("Error on DownloadAndExtractChartTarball: %v", err)
			it.err = err
			out <- it
			return
		}
		// if we have a problem processing an image it will fallback to the default one
		err = charthelper.DownloadAndProcessChartIcon(chart)
		if err != nil {
			log.Errorf("DownloadAndProcessChartIcon error: %v", err)
		}
	}
	out <- it
}
