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

package db

import (
	"context"
	"database/sql"

	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// +onecloud:model-api-gen
type SExternalizedResourceBase struct {
	// 外部Id, 对用公有云私有资源自身的Id
	ExternalId string `width:"256" charset:"utf8" index:"true" list:"user" create:"domain_optional" update:"admin" json:"external_id"`
}

type SExternalizedResourceBaseManager struct{}

func (model SExternalizedResourceBase) GetExternalId() string {
	return model.ExternalId
}

func (model *SExternalizedResourceBase) SetExternalId(idStr string) {
	model.ExternalId = idStr
}

type IExternalizedModelManager interface {
	IModelManager
	FetchByExternalId(idStr string) IExternalizedModel
}

type IExternalizedModel interface {
	IModel

	GetExternalId() string
	SetExternalId(idStr string)
}

func SetExternalId(model IExternalizedModel, userCred mcclient.TokenCredential, idStr string) error {
	if model.GetExternalId() != idStr {
		diff, err := Update(model, func() error {
			model.SetExternalId(idStr)
			return nil
		})
		if err == nil {
			OpsLog.LogEvent(model, ACT_UPDATE, diff, userCred)
		}
		return err
	}
	return nil
}

func FetchByExternalId(manager IModelManager, idStr string) (IExternalizedModel, error) {
	return FetchByExternalIdAndManagerId(manager, idStr, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q
	})
}

func FetchByExternalIdAndManagerId(manager IModelManager, idStr string, filter func(q *sqlchemy.SQuery) *sqlchemy.SQuery) (IExternalizedModel, error) {
	q := manager.Query().Equals("external_id", idStr)
	q = filter(q)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count == 1 {
		obj, err := NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		err = q.First(obj)
		if err != nil {
			return nil, err
		} else {
			return obj.(IExternalizedModel), nil
		}
	} else if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else {
		return nil, sql.ErrNoRows
	}
}

func (manager *SExternalizedResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.ExternalizedResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.ExternalId) > 0 {
		q = q.Equals("external_id", query.ExternalId)
	}
	return q, nil
}
