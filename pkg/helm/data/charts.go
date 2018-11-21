package data

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/provenance"
	"k8s.io/helm/pkg/repo"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/yunion-kube/pkg/helm/data/common"
	"yunion.io/x/yunion-kube/pkg/types/helm"
)

// ICharts is an interface for managing chart data sourced from a repository index
type ICharts interface {
	// ChartFromRepo retrieves the latest version of a particular chart from a repo
	ChartFromRepo(repo, name string) (*helm.ChartPackage, error)
	// ChartVersionFromRepo retrieves a specific chart version from a repo
	ChartVersionFromRepo(repo, name, version string) (*helm.ChartPackage, error)
	// ChartVersionsFromRepo retrieves all chart versions from a repo
	ChartVersionsFromRepo(repo, name string) ([]*helm.ChartPackage, error)
	// AllFromRepo retrieves all charts from a repo
	AllFromRepo(repo string) ([]*helm.ChartPackage, error)
	// All retrieves all charts from all repos
	All() ([]*helm.ChartPackage, error)
	// Search operates against all charts/repos
	Search(params jsonutils.JSONObject) ([]*helm.ChartPackage, error)
	// Refresh charts data
	Refresh() error
	//RefreshChart(repo string, chartName string) error
	//DeleteChart(repo string, chartName string, version string) error
}

func DownloadChartFromRepo(name, version string) (string, error) {
	return downloadChartFromRepo(name, version, common.GenerateHelmRepoPath(""))
}

func getHelmHomePath() helmpath.Home {
	return common.CreateEnvSettings(common.GenerateHelmRepoPath("")).Home
}

func getChartTGZPath(name, version string) string {
	return filepath.Join(getHelmHomePath().Archive(), fmt.Sprintf("%s-%s.tgz", name, version))
}

func isChartTGZChanged(dstChart *repo.ChartVersion) (string, bool) {
	pkgPath := getChartTGZPath(dstChart.Name, dstChart.Version)
	digest, err := provenance.DigestFile(pkgPath)
	if err != nil {
		log.Errorf("Calcuate %s package digest error: %v", pkgPath, err)
		return pkgPath, true
	}
	log.Debugf("cached package '%s-%s' digest: %q, expected digest: %q", dstChart.Name, dstChart.Version, digest, dstChart.Digest)
	if digest != dstChart.Digest {
		return pkgPath, true
	}
	return pkgPath, false
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

func getRepoIndexFile(repoName string) (*repo.IndexFile, error) {
	r, err := repoShowLocal(repoName)
	if err != nil {
		return nil, err
	}
	i, err := repo.LoadIndexFile(r.Cache)
	if err != nil {
		return nil, err
	}
	return i, nil
}

func ChartShowDetails(repoName, chartName, chartVersion string) (*helm.ChartDetail, error) {
	idxFile, err := getRepoIndexFile(repoName)
	if err != nil {
		return nil, err
	}
	if idxFile == nil {
		return nil, fmt.Errorf("Repo %q index.yaml object is nil", repoName)
	}
	dstChart, err := idxFile.Get(chartName, chartVersion)
	if err != nil {
		return nil, err
	}
	if dstChart == nil {
		return nil, fmt.Errorf("Not found chart '%s/%s:%s'", repoName, chartName, chartVersion)
	}

	downloadedChartPath, changed := isChartTGZChanged(dstChart)

	if changed {
		joinName := fmt.Sprintf("%s/%s", repoName, dstChart.Name)
		downloadedChartPath, err = DownloadChartFromRepo(joinName, dstChart.Version)
		if err != nil {
			return nil, fmt.Errorf("Download chart '%s/%s:%s': %v", repoName, chartName, chartVersion, err)
		}
	} else {
		log.Debugf("Chart '%s/%s:%s' digest not change, use cached package", repoName, chartName, chartVersion)
	}

	reader, err := newChartTGZReader(downloadedChartPath)
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
	chartD := &helm.ChartDetail{
		Name:    chartName,
		Repo:    repoName,
		Chart:   dstChart,
		Values:  valuesStr,
		Readme:  readmeStr,
		Options: options.Options,
	}
	return chartD, nil
}

func ChartsList(query *helm.ChartQuery) ([]*ChartResult, error) {
	q := chartQuery{
		Name:       query.Name,
		Repo:       query.Repo,
		Keyword:    query.Keyword,
		AllVersion: query.AllVersion,
	}
	allVersion := query.AllVersion
	chartList, err := allRepoCharts(query, allVersion)
	if err != nil {
		return nil, err
	}
	return chartList.QueryItems(q), nil
}

func getChartLatestVersionFromRepo(repoName, chartName string) (string, error) {
	chartList, err := repoCharts(repoName, false)
	if err != nil {
		return "", err
	}
	ret := chartList.QueryItems(chartQuery{Name: chartName})
	if len(ret) == 0 {
		return "", fmt.Errorf("Not found chart %s/%s", repoName, chartName)
	}
	return ret[0].Chart.Version, nil
}

func ChartFromRepo(repoName, chartName, chartVersion string) (*helm.ChartPackage, error) {
	var err error
	if chartVersion == "" {
		chartVersion, err = getChartLatestVersionFromRepo(repoName, chartName)
		if err != nil {
			return nil, err
		}
	}
	_, err = ChartShowDetails(repoName, chartName, chartVersion)
	if err != nil {
		return nil, err
	}
	chartPkgPath := getChartTGZPath(chartName, chartVersion)
	chartPkg, err := chartutil.Load(chartPkgPath)
	if err != nil {
		return nil, err
	}
	return &helm.ChartPackage{
		Repo:  repoName,
		Chart: chartPkg,
	}, nil
}

type chartFilterFunc func(*ChartResult, chartQuery) bool

type chartList struct {
	charts  []*ChartResult
	filters []chartFilterFunc
}

type chartQuery struct {
	Name       string
	Repo       string
	Keyword    string
	AllVersion bool
}

func newChartList(charts []*ChartResult) *chartList {
	generalFilter := func(item *ChartResult, query chartQuery) bool {
		if query.Name != "" {
			if query.Name != item.Chart.Name {
				return false
			}
		}
		if query.Repo != "" {
			if query.Repo != item.Repo {
				return false
			}
		}

		if query.Keyword != "" {
			kwString := strings.ToLower(strings.Join(item.Chart.Keywords, " "))
			kwMatched, _ := regexp.MatchString(query.Keyword, kwString)
			if !kwMatched {
				return false
			}
		}
		return true
	}

	yunionFilter := func(item *ChartResult, query chartQuery) bool {
		if item.Repo != helm.YUNION_REPO_NAME {
			return true
		}
		if utils.IsInStringArray(helm.YUNION_REPO_HIDE_KEYWORD, item.Chart.Keywords) && !query.AllVersion {
			return false
		}
		return true
	}
	return &chartList{
		charts: charts,
		filters: []chartFilterFunc{
			generalFilter,
			yunionFilter,
		},
	}
}

func (l *chartList) QueryItems(query chartQuery) []*ChartResult {
	ret := make([]*ChartResult, 0)
	for _, item := range l.charts {
		if l.doFilter(item, query) {
			ret = append(ret, item)
		}
	}
	nl := newChartList(ret)
	sort.Sort(nl)
	return nl.charts
}

func (l *chartList) doFilter(item *ChartResult, query chartQuery) bool {
	for _, filter := range l.filters {
		if !filter(item, query) {
			return false
		}
	}
	return true
}

func (l *chartList) Add(nl *chartList) *chartList {
	for _, item := range nl.charts {
		l.charts = append(l.charts, item)
	}
	return l
}

func (l *chartList) Len() int {
	return len(l.charts)
}

func (l *chartList) Swap(i, j int) {
	l.charts[i], l.charts[j] = l.charts[j], l.charts[i]
}

func (l *chartList) Less(i, j int) bool {
	first := l.charts[i]
	second := l.charts[j]

	name1 := fmt.Sprintf("%s/%s", first.Repo, first.Chart.Name)
	name2 := fmt.Sprintf("%s/%s", second.Repo, second.Chart.Name)

	if name1 == name2 {
		v1, err := semver.NewVersion(first.Chart.Version)
		if err != nil {
			return true
		}
		v2, err := semver.NewVersion(second.Chart.Version)
		if err != nil {
			return true
		}
		// Sort so that the newest chart is higher than the oldest chart
		return v1.GreaterThan(v2)
	}
	return name1 < name2
}
