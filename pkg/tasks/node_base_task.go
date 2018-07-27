package tasks

import (
	//"context"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/yunion-kube/pkg/models"
)

type SNodeBaseTask struct {
	taskman.STask
}

func (t *SNodeBaseTask) getNode() *models.SNode {
	obj := t.GetObject()
	return obj.(*models.SNode)
}
