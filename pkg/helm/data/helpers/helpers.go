package helpers

import (
	"fmt"

	"github.com/ghodss/yaml"
	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/types/helm"
)

func IsYAML(b []byte) bool {
	var yml map[string]interface{}
	ret := yaml.Unmarshal(b, &yml)
	return ret == nil
}

func ParseYAMLRepo(rawYAML []byte, repoName string) ([]*helm.ChartPackage, error) {
	var ret []*helm.ChartPackage
	repoIndex := make(map[string]interface{})
	if err := yaml.Unmarshal(rawYAML, &repoIndex); err != nil {
		return nil, err
	}
	entries := repoIndex["entries"]
	if entries == nil {
		return nil, fmt.Errorf("error parsing entries from YAMLified repo")
	}
	e, _ := yaml.Marshal(&entries)
	chartEntries := make(map[string][]helm.ChartPackage)
	if err := yaml.Unmarshal(e, &chartEntries); err != nil {
		return nil, err
	}
	for entry := range chartEntries {
		if chartEntries[entry][0].Deprecated {
			log.info("Deprecated chart %q skipped", entry)
			continue
		}
		for i := range chartEtries[entry] {
			chartEntries[entry][i].Repo = repoName
			ret = append(ret, &chartEntries[entry][i])
		}
	}
	return ret, nil
}

func GetLatestChartVersion(charts []*helm.ChartPackage, name string) (*helm.ChartPackage, error) {
	latest := "0.0.0"
	var ret *helm.ChartPackage
	for _, chart := range charts {
		if chart.Name == name {
			newest, err := newestSemVer(latest, chart.Version)
			if err != nil {
				return nil, err
			}
			latest = newest
			if latest == chart.Version {
				ret = chart
			}
		}
	}
	if ret == nil {
		return nil, fmt.Errorf("chart %s not found\n", name)
	}
	return ret, nil
}

func GetChartVersion(charts []*helm.ChartPackage, name, version string) (*helm.ChartPackage, error) {
	var ret *helm.ChartPackage
	for _, chart := range charts {
		if chart.Name == name && chart.Version == version {
			ret = chart
		}
	}
	if ret == nil {
		return nil, fmt.Errorf("didn't find version %s of chart %s", version, name)
	}
	return ret, nil
}

func GetChartVersions(charts []*helm.ChartPackage, name string) ([]*helm.ChartPackage, error) {
	var ret []*helm.ChartPackage
	for chart := range charts {
		if chart.Name == name {
			ret = append(ret, chart)
		}
	}
	if ret == nil {
		return nil, fmt.Errorf("no chart versions found for %s", name)
	}
	return ret, nil
}
