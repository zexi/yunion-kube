package common

import (
	"fmt"
	"os"

	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"

	"yunion.io/x/log"
)

var (
	stateStorePath string
)

func CreateEnvSettings(helmRepoHome string) environment.EnvSettings {
	var settings environment.EnvSettings
	settings.Home = helmpath.Home(helmRepoHome)
	return settings
}

func GenerateHelmRepoPath(path string) string {
	if len(path) == 0 {
		return stateStorePath
	}
	return fmt.Sprintf("%s/%s", stateStorePath, path)
}

func InitStateStoreDir(dirPath string) error {
	if len(dirPath) == 0 {
		return fmt.Errorf("Helm state store path must specified")
	}
	if _, err := os.Stat(dirPath); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(dirPath, 0755)
			if err != nil {
				return fmt.Errorf("Make directory %q: %v", dirPath, err)
			}
		} else {
			return fmt.Errorf("Get directory %s stat: %v", dirPath, err)
		}
	}
	stateStorePath = dirPath
	return nil
}

func EnsureStateStoreDir(dirPath string) {
	err := InitStateStoreDir(dirPath)
	if err != nil {
		log.Fatalf("Init stateStorePath %s: %v", dirPath, err)
	}
}
