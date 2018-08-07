package repohelper

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"yunion.io/yunion-kube/pkg/helm/data/helpers"
	"yunion.io/yunion-kube/pkg/models"
	"yunion.io/yunion-kube/pkg/types/helm"
)

func GetChartsFromRepoIndexFile(repo *models.SRepo) ([]*helm.ChartPackage, error) {
	u, _ := url.Parse(repo.Url)
	u.Path = path.Join(u.Path, "index.yaml")

	// 1. Download repo index
	var client http.Client
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "yunion-kube/2.0.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 2. Parse repo index
	charts, err := helpers.ParseYAMLRepo(body, repo.Name)
	if err != nil {
		return nil, err
	}
	return charts, nil
}
