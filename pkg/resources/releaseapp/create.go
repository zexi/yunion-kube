package releaseapp

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/release"
)

type CreateReleaseAppRequest struct {
	*release.CreateUpdateReleaseRequest
}

func NewCreateReleaseAppRequest(data jsonutils.JSONObject) (*CreateReleaseAppRequest, error) {
	createOpt, err := release.NewCreateUpdateReleaseReq(data)
	if err != nil {
		return nil, err
	}
	return &CreateReleaseAppRequest{
		CreateUpdateReleaseRequest: createOpt,
	}, nil
}

func (r *CreateReleaseAppRequest) ToData() *release.CreateUpdateReleaseRequest {
	return r.CreateUpdateReleaseRequest
}

func (r *CreateReleaseAppRequest) IsSetsEmpty() bool {
	return len(r.Sets) == 0
}

func (app *SReleaseAppManager) ValidateCreateData(req *common.Request) error {
	data := req.Data
	ns, _ := data.GetString("namespace")
	if ns == "" {
		data.Set("namespace", jsonutils.NewString(req.GetDefaultNamespace()))
	}
	name, _ := data.GetString("release_name")
	if name == "" {
		name, err := release.GenerateName("")
		if err != nil {
			return err
		}
		data.Set("release_name", jsonutils.NewString(name))
	}
	return nil
}

func (man *SReleaseAppManager) Create(req *common.Request) (interface{}, error) {
	createOpt, err := NewCreateReleaseAppRequest(req.Data)
	if err != nil {
		return nil, err
	}
	if createOpt.IsSetsEmpty() {
		createOpt.Sets = man.hooker.GetConfigSets().ToSets()
	}
	cli, err := req.GetHelmClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	return release.ReleaseCreate(cli, createOpt.ToData())
}
