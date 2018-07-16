package ykecerts

import (
	"fmt"

	"yunion.io/yke/pkg/pki"

	"yunion.io/yunion-kube/pkg/clusterdriver/yke/ykecerts"
	"yunion.io/yunion-kube/pkg/models"
	"yunion.io/yunion-kube/pkg/types/apis"
)

type BundleLookup struct {
	clusterManager *models.SClusterManager
}

func NewLookup(man *models.SClusterManager) *BundleLookup {
	return &BundleLookup{clusterManager: man}
}

func (r *BundleLookup) Lookup(cluster *apis.Cluster) (*Bundle, error) {
	c := r.clusterManager.FetchCluster(cluster.Name)
	if c == nil {
		return nil, fmt.Errorf("Not found cluster %q", cluster.Name)
	}

	certs := c.Certs
	certMap, err := ykecerts.LoadString(certs)
	if err != nil {
		return nil, err
	}

	newCertMap := map[string]pki.CertificatePKI{}
	for k, v := range certMap {
		if v.Config != "" {
			v.ConfigPath = pki.GetConfigPath(k)
		}
		if v.Key != nil {
			v.KeyPath = pki.GetKeyPath(k)
		}
		if v.Certificate != nil {
			v.Path = pki.GetCertPath(k)
		}
		newCertMap[k] = v
	}

	return newBundle(newCertMap), nil
}
