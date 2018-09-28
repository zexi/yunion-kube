package ykecerts

import (
	"yunion.io/x/yke/pkg/pki"

	"yunion.io/x/yunion-kube/pkg/clusterdriver/yke/ykecerts"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/types/apis"
)

type BundleLookup struct {
	clusterManager *models.SClusterManager
}

func NewLookup(man *models.SClusterManager) *BundleLookup {
	return &BundleLookup{clusterManager: man}
}

func (r *BundleLookup) Lookup(cluster *apis.Cluster) (*Bundle, error) {
	c, err := r.clusterManager.FetchClusterById(cluster.Id)
	if err != nil {
		return nil, err
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
