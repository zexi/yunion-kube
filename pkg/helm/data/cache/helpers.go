package cache

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/ghodss/yaml"

	"yunion.io/x/yunion-kube/pkg/types/helm"
)

func IsYAML(b []byte) bool {
	var yml map[string]interface{}
	ret := yaml.Unmarshal(b, &yml)
	return ret == nil
}

func GetLatestChartVersion(charts []*helm.ChartPackage, name string) (*helm.ChartPackage, error) {
	latest := "0.0.0"
	var ret *helm.ChartPackage
	for _, chart := range charts {
		if chart.Metadata.Name == name {
			newest, err := newestSemVer(latest, chart.Metadata.Version)
			if err != nil {
				return nil, err
			}
			latest = newest
			if latest == chart.Metadata.Version {
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
		if chart.Metadata.Name == name && chart.Metadata.Version == version {
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
	for _, chart := range charts {
		if chart.Metadata.Name == name {
			ret = append(ret, chart)
		}
	}
	if ret == nil {
		return nil, fmt.Errorf("no chart versions found for %s", name)
	}
	return ret, nil
}

// newestSemVer returns the newest (largest) semver string
func newestSemVer(v1 string, v2 string) (string, error) {
	v1Semver, err := semver.NewVersion(v1)
	if err != nil {
		return "", err
	}

	v2Semver, err := semver.NewVersion(v2)
	if err != nil {
		return "", err
	}

	if v1Semver.LessThan(v2Semver) {
		return v2, nil
	}
	return v1, nil
}

func downloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	tarContent := new(bytes.Buffer)
	io.Copy(tarContent, resp.Body)
	gzf, err := gzip.NewReader(tarContent)
	if err != nil {
		return nil, err
	}
	rawContent, _ := ioutil.ReadAll(gzf)
	return rawContent, nil
}

// getChartFile Download file from chart repository
func getChartFile(file []byte, fileName string) (string, error) {
	tarReader := tar.NewReader(bytes.NewReader(file))
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if strings.Contains(header.Name, fileName) {
			valuesContent := new(bytes.Buffer)
			if _, err := io.Copy(valuesContent, tarReader); err != nil {
				return "", err
			}
			base64Str := base64.StdEncoding.EncodeToString(valuesContent.Bytes())
			return base64Str, nil
		}
	}
	return "", nil
}

func getChartOption(file []byte) (*SpotguideFile, error) {
	so := &SpotguideFile{}
	tarReader := tar.NewReader(bytes.NewReader(file))
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if strings.Contains(header.Name, "spotguide.json") {
			valuesContent := new(bytes.Buffer)
			if _, err := io.Copy(valuesContent, tarReader); err != nil {
				return nil, err
			}
			err := json.Unmarshal(valuesContent.Bytes(), so)
			if err != nil {
				return nil, err
			}
			return so, nil
		}
	}
	return so, nil
}
