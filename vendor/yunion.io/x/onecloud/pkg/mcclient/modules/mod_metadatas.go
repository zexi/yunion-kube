// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package modules

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type MetadataManager struct {
	modulebase.ResourceManager
}

var (
	Metadatas MetadataManager
)

func init() {
	Metadatas = MetadataManager{NewComputeManager("metadata", "metadatas",
		[]string{"id", "key", "value"},
		[]string{})}
	registerCompute(&Metadatas)
}

func (this *MetadataManager) getModule(session *mcclient.ClientSession, params jsonutils.JSONObject) (*modulebase.ResourceManager, error) {
	service, _ := params.GetString("service")
	if len(service) > 0 {
		_, err := session.GetServiceURL(service, "")
		if err != nil {
			return nil, httperrors.NewNotFoundError("service %s not found error: %v", service, err)
		}
	} else {
		service = "compute"
	}
	return &modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(service, "", "", []string{}, []string{}),
		Keyword:     "metadata", KeywordPlural: "metadatas",
	}, nil
}

func (this *MetadataManager) List(session *mcclient.ClientSession, params jsonutils.JSONObject) (*modulebase.ListResult, error) {
	mod, err := this.getModule(session, params)
	if err != nil {
		return nil, err
	}
	return mod.List(session, params)
}

func (this *MetadataManager) Get(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	mod, err := this.getModule(session, params)
	if err != nil {
		return nil, err
	}
	return mod.Get(session, id, params)
}
