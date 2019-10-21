package initial

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"yunion.io/x/yunion-kube/pkg/client"

	_ "yunion.io/x/yunion-kube/pkg/drivers/clusters"
	_ "yunion.io/x/yunion-kube/pkg/drivers/machines"
	_ "yunion.io/x/yunion-kube/pkg/tasks"
)

func InitClient() {
	go wait.Forever(client.BuildApiserverClient, 5*time.Second)
}
