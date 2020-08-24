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
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/informer"
	"yunion.io/x/onecloud/pkg/util/nopanic"
)

type ITableSpec interface {
	Name() string
	Columns() []sqlchemy.IColumnSpec
	PrimaryColumns() []sqlchemy.IColumnSpec
	DataType() reflect.Type
	CreateSQL() string
	Instance() *sqlchemy.STable
	ColumnSpec(name string) sqlchemy.IColumnSpec
	Insert(ctx context.Context, dt interface{}) error
	InsertOrUpdate(ctx context.Context, dt interface{}) error
	Update(ctx context.Context, dt interface{}, doUpdate func() error) (sqlchemy.UpdateDiffs, error)
	Fetch(dt interface{}) error
	FetchAll(dest interface{}) error
	SyncSQL() []string
	DropForeignKeySQL() []string
	AddIndex(unique bool, cols ...string) bool
	Increment(diff interface{}, target interface{}) error
	Decrement(diff interface{}, target interface{}) error
}

type sTableSpec struct {
	*sqlchemy.STableSpec
}

func newTableSpec(model interface{}, tableName string) ITableSpec {
	return &sTableSpec{
		STableSpec: sqlchemy.NewTableSpecFromStruct(model, tableName),
	}
}

func (ts *sTableSpec) newInformerModel(dt interface{}) (*informer.ModelObject, error) {
	obj, ok := dt.(IModel)
	if !ok {
		return nil, errors.Errorf("informer model is not IModel")
	}
	if obj.GetVirtualObject() == nil {
		return nil, errors.Errorf("object %#v virtual object is nil", obj)
	}
	if obj.GetModelManager() == nil {
		return nil, errors.Errorf("object %#v model manager is nil", obj)
	}
	jointObj, isJoint := obj.(IJointModel)
	if isJoint {
		mObj := JointMaster(jointObj)
		sObj := JointSlave(jointObj)
		return informer.NewJointModel(jointObj, jointObj.KeywordPlural(), mObj.GetId(), sObj.GetId()), nil
	}
	return informer.NewModel(obj, obj.KeywordPlural(), obj.GetId()), nil
}

func (ts *sTableSpec) isMarkDeleted(dt interface{}) (bool, error) {
	if vObj, ok := dt.(IVirtualModel); ok {
		if vObj.GetPendingDeleted() {
			return true, nil
		}
	}
	obj, ok := dt.(IModel)
	if !ok {
		return false, errors.Errorf("informer model is not IModel")
	}
	return obj.GetDeleted(), nil
}

func (ts *sTableSpec) Insert(ctx context.Context, dt interface{}) error {
	if err := ts.STableSpec.Insert(dt); err != nil {
		return err
	}
	ts.inform(ctx, dt, informer.Create)
	return nil
}

func (ts *sTableSpec) InsertOrUpdate(ctx context.Context, dt interface{}) error {
	if err := ts.STableSpec.InsertOrUpdate(dt); err != nil {
		return err
	}
	ts.inform(ctx, dt, informer.Create)
	return nil
}

func (ts *sTableSpec) Update(ctx context.Context, dt interface{}, doUpdate func() error) (sqlchemy.UpdateDiffs, error) {
	oldObj := jsonutils.Marshal(dt)
	diffs, err := ts.STableSpec.Update(dt, doUpdate)
	if err != nil {
		return nil, err
	}
	if diffs == nil {
		// no data to update
		return nil, nil
	}
	isDeleted, err := ts.isMarkDeleted(dt)
	if err != nil {
		return nil, errors.Wrap(err, "check is mark deleted")
	}
	if isDeleted {
		ts.inform(ctx, dt, informer.Delete)
	} else {
		ts.informUpdate(ctx, dt, oldObj.(*jsonutils.JSONDict))
	}
	return diffs, nil
}

func (ts *sTableSpec) inform(ctx context.Context, dt interface{}, f func(ctx context.Context, obj *informer.ModelObject) error) {
	if !informer.IsInit() {
		return
	}
	nf := func() {
		obj, err := ts.newInformerModel(dt)
		if err != nil {
			log.Warningf("newInformerModel error: %v", err)
			return
		}
		if err := f(ctx, obj); err != nil {
			if errors.Cause(err) == informer.ErrBackendNotInit {
				log.V(4).Warningf("informer backend not init")
			} else {
				log.Errorf("call informer func error: %v", err)
			}
		}
	}
	nopanic.Run(nf)
}

func (ts *sTableSpec) informUpdate(ctx context.Context, dt interface{}, oldObj *jsonutils.JSONDict) {
	if !informer.IsInit() {
		return
	}
	nf := func() {
		obj, err := ts.newInformerModel(dt)
		if err != nil {
			log.Warningf("newInformerModel error: %v", err)
			return
		}
		if err := informer.Update(ctx, obj, oldObj); err != nil {
			if errors.Cause(err) == informer.ErrBackendNotInit {
				log.V(4).Warningf("informer backend not init")
			} else {
				log.Errorf("call informer update func error: %v", err)
			}
		}
	}
	nopanic.Run(nf)
}
