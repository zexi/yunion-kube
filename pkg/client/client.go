package client

import (
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/manager"
)

const (
	// High enough QPS to fit all expected use cases.
	defaultQPS = 1e6
	// High enough Burst to fit all expected use cases.
	defaultBurst = 1e6
	// full resync cache resource time
	defaultResyncPeriod = 30 * time.Second
)

var (
	ErrNotExist = errors.New("cluster not exist.")
	ErrStatus   = errors.New("cluster invalid status, please try again later.")
)

var (
	clusterManagerSets = &sync.Map{}
)

type ClusterManager struct {
	Cluster    manager.ICluster
	Config     *rest.Config
	KubeClient ResourceHandler
	APIServer  string
	KubeConfig string
	// KubeConfigPath used for kubectl or helm client
	KubeConfigPath string
}

func (c ClusterManager) GetIndexer() *CacheFactory {
	return c.KubeClient.GetIndexer()
}

func (c ClusterManager) Close() {
	os.RemoveAll(c.KubeConfigPath)
	c.KubeClient.Close()
}

func BuildApiserverClient() {
	newClusters, err := clusters.ClusterManager.GetRunningClusters()
	if err != nil {
		log.Errorf("build apiserver client get all cluster error.", err)
		return
	}

	changed := clusterChanged(newClusters)
	if changed {
		log.Infof("cluster changed, so resync info...")

		shouldRemoveClusters(newClusters)
		// build new clientManager
		for i := 0; i < len(newClusters); i++ {
			cluster := newClusters[i]
			apiServer, err := cluster.GetAPIServer()
			if err != nil {
				log.Warningf("get cluster %s apiserver: %v", cluster.GetName(), err)
				continue
			}
			kubeconfig, err := cluster.GetKubeconfig()
			if err != nil {
				log.Warningf("get cluster %s kubeconfig: %v", cluster.GetName(), err)
				continue
			}
			clientSet, config, err := BuildClient(apiServer, kubeconfig)
			if err != nil {
				log.Warningf("build cluster (%s) client error: %v", cluster.GetName(), err)
				continue
			}
			kubeconfigPath, err := BuildKubeConfigPath(cluster, kubeconfig)
			if err != nil {
				log.Warningf("build cluster %s kubeconfig path: %v", cluster.GetName(), err)
				continue
			}
			cacheFactory, err := buildCacheController(clientSet)
			if err != nil {
				log.Warningf("build cluster (%s) cache controller error: %v", cluster.GetName(), err)
				continue
			}

			clusterManager := &ClusterManager{
				Cluster:        cluster,
				Config:         config,
				KubeClient:     NewResourceHandler(clientSet, cacheFactory),
				APIServer:      apiServer,
				KubeConfig:     kubeconfig,
				KubeConfigPath: kubeconfigPath,
			}
			managerInterface, ok := clusterManagerSets.Load(cluster.GetId())
			if ok {
				man := managerInterface.(*ClusterManager)
				man.Close()
			}

			clusterManagerSets.Store(cluster.GetId(), clusterManager)
		}
		log.Infof("resync cluster finished!")
	}
}

func SyncMapLen(m *sync.Map) int {
	length := 0
	m.Range(func(key, value interface{}) bool {
		length++
		return true
	})
	return length
}

func clusterChanged(clusters []manager.ICluster) bool {
	if SyncMapLen(clusterManagerSets) != len(clusters) {
		log.Infof("cluster length (%d) changed to (%d).", SyncMapLen(clusterManagerSets), len(clusters))
		return true
	}

	for _, cluster := range clusters {
		manInterface, ok := clusterManagerSets.Load(cluster.GetId())
		if !ok {
			// maybe add new cluster
			return true
		}
		man := manInterface.(*ClusterManager)
		// apiserver changed, the cluster is changed, ignore others
		apiServer, err := cluster.GetAPIServer()
		if err != nil {
			log.Warningf("get cluster %s apiserver: %v", cluster.GetName(), err)
			return true
		}
		kubeconfig, err := cluster.GetKubeconfig()
		if err != nil {
			log.Warningf("get cluster %s kubeconfig: %v", cluster.GetName(), err)
			return true
		}
		if man.APIServer != apiServer {
			log.Infof("cluster apiserver %q changed to %q.", man.APIServer, apiServer)
			return true
		}
		if man.KubeConfig != kubeconfig {
			log.Infof("cluster kubeConfig %q changed to %q", man.KubeConfig, kubeconfig)
			return true
		}
		if man.Cluster.GetStatus() != cluster.GetStatus() {
			log.Infof("cluster status %q changed to %q", man.Cluster.GetStatus(), cluster.GetStatus())
			return true
		}
	}
	return false
}

// deal with deleted cluster
func shouldRemoveClusters(changedClusters []manager.ICluster) {
	changedClusterMap := make(map[string]struct{})
	for _, cluster := range changedClusters {
		changedClusterMap[cluster.GetId()] = struct{}{}
	}

	clusterManagerSets.Range(func(key, value interface{}) bool {
		if _, ok := changedClusterMap[key.(string)]; !ok {
			manInterface, _ := clusterManagerSets.Load(key)
			man := manInterface.(*ClusterManager)
			man.Close()
			clusterManagerSets.Delete(key)
		}
		return true
	})
}

func GetManagerByCluster(c *clusters.SCluster) (*ClusterManager, error) {
	return GetManager(c.GetId())
}

func GetManager(cluster string) (*ClusterManager, error) {
	manInterface, exist := clusterManagerSets.Load(cluster)
	if !exist {
		BuildApiserverClient()
		_, exist = clusterManagerSets.Load(cluster)
		if !exist {
			return nil, errors.Wrapf(ErrNotExist, "cluster %s", cluster)
		}
	}
	man := manInterface.(*ClusterManager)
	status := man.Cluster.GetStatus()
	if status != apis.ClusterStatusRunning {
		return nil, errors.Wrapf(ErrStatus, "cluster %s status %s", cluster, status)
	}
	return man, nil
}

func BuildClient(master string, kubeconfig string) (*kubernetes.Clientset, *rest.Config, error) {
	configInternal, err := clientcmd.Load([]byte(kubeconfig))
	if err != nil {
		return nil, nil, err
	}

	clientConfig, err := clientcmd.NewDefaultClientConfig(*configInternal, &clientcmd.ConfigOverrides{
		ClusterDefaults: clientcmdapi.Cluster{Server: master},
	}).ClientConfig()

	if err != nil {
		log.Errorf("build client config error. %v ", err)
		return nil, nil, err
	}

	clientConfig.QPS = defaultQPS
	clientConfig.Burst = defaultBurst

	clientSet, err := kubernetes.NewForConfig(clientConfig)

	if err != nil {
		log.Errorf("(%s) kubernetes.NewForConfig(%v) error.%v", master, err, clientConfig)
		return nil, nil, err
	}

	return clientSet, clientConfig, nil
}

func ClusterKubeConfigPath(c manager.ICluster) string {
	return path.Join("/tmp", strings.Join([]string{"kubecluster", c.GetName(), c.GetId(), ".kubeconfig"}, "-"))
}

func BuildKubeConfigPath(c manager.ICluster, kubeconfig string) (string, error) {
	configPath := ClusterKubeConfigPath(c)
	if err := ioutil.WriteFile(configPath, []byte(kubeconfig), 0666); err != nil {
		return "", errors.Wrapf(err, "write %s", configPath)
	}
	return configPath, nil
}
