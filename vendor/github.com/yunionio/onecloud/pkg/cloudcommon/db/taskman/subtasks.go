package taskman

import (
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
)

const (
	SUBTASK_INIT = "init"
	SUBTASK_FAIL = "fail"
	SUBTASK_SUCC = "succ"
)

type SSubTaskmanager struct {
	db.SModelBaseManager
}

var SubTaskManager *SSubTaskmanager

func init() {
	SubTaskManager = &SSubTaskmanager{SModelBaseManager: db.NewModelBaseManager(SSubTask{}, "subtasks_tbl", "subtask", "subtasks")}
}

type SSubTask struct {
	db.SModelBase

	TaskId    string `width:"36" charset:"ascii" nullable:"false" primary:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=False, primary_key=True)
	Stage     string `width:"64" charset:"ascii" nullable:"false" primary:"true"` // Column(VARCHAR(64, charset='ascii'), nullable=False, primary_key=True)
	SubtaskId string `width:"36" charset:"ascii" nullable:"false" primary:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=False, primary_key=True)
	Status    string `width:"36" charset:"ascii" nullable:"false" default:"init"` // Column(VARCHAR(36, charset='ascii'), nullable=False, default=SUBTASK_INIT)
	Result    string `length:"medium" charset:"ascii" nullable:"true"`            // Column(MEDIUMTEXT(charset='ascii'), nullable=True)
}

func (manager *SSubTaskmanager) GetSubTask(ptaskId string, subtaskId string) *SSubTask {
	subtask := SSubTask{}
	err := manager.Query().Equals("task_id", ptaskId).Equals("subtask_id", subtaskId).First(&subtask)
	if err != nil {
		log.Errorf("GetSubTask fail %s", err)
		return nil
	}
	return &subtask
}

func (manager *SSubTaskmanager) GetInitSubtasks(taskId string, stage string) []SSubTask {
	subtasks := make([]SSubTask, 0)
	q := manager.Query().Equals("task_id", taskId).Equals("stage", stage).Equals("status", SUBTASK_INIT)
	err := db.FetchModelObjects(manager, q, &subtasks)
	if err != nil {
		log.Errorf("GetInitSubtasks fail %s", err)
		return nil
	}
	return subtasks
}

func (self *SSubTask) SaveResults(failed bool, result jsonutils.JSONObject) error {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		if failed {
			self.Status = SUBTASK_FAIL
		} else {
			self.Status = SUBTASK_SUCC
		}
		self.Result = result.String()
		return nil
	})
	if err != nil {
		log.Errorf("SaveUpdate save update fail %s", err)
		return err
	}
	return nil
}