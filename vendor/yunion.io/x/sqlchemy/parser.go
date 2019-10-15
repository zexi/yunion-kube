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

package sqlchemy

import (
	"reflect"

	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/reflectutils"
)

func structField2ColumnSpec(field *reflectutils.SStructFieldValue) IColumnSpec {
	fieldname := field.Info.MarshalName()
	tagmap := field.Info.Tags
	if _, ok := tagmap[TAG_IGNORE]; ok {
		return nil
	}
	fieldType := field.Value.Type()
	var retCol = getFiledTypeCol(fieldType, fieldname, tagmap, false)
	if retCol == nil && fieldType.Kind() == reflect.Ptr {
		retCol = getFiledTypeCol(fieldType.Elem(), fieldname, tagmap, true)
	}
	if retCol == nil {
		panic("unsupported colume data type %s" + fieldType.Name())
	}
	return retCol
}

func getFiledTypeCol(fieldType reflect.Type, fieldname string, tagmap map[string]string, isPointer bool) IColumnSpec {
	switch fieldType {
	case gotypes.StringType:
		col := NewTextColumn(fieldname, tagmap, isPointer)
		return &col
	case gotypes.IntType, gotypes.Int32Type:
		tagmap[TAG_WIDTH] = "11"
		col := NewIntegerColumn(fieldname, "INT", false, tagmap, isPointer)
		return &col
	case gotypes.Int8Type:
		tagmap[TAG_WIDTH] = "4"
		col := NewIntegerColumn(fieldname, "TINYINT", false, tagmap, isPointer)
		return &col
	case gotypes.Int16Type:
		tagmap[TAG_WIDTH] = "6"
		col := NewIntegerColumn(fieldname, "SMALLINT", false, tagmap, isPointer)
		return &col
	case gotypes.Int64Type:
		tagmap[TAG_WIDTH] = "20"
		col := NewIntegerColumn(fieldname, "BIGINT", false, tagmap, isPointer)
		return &col
	case gotypes.UintType, gotypes.Uint32Type:
		tagmap[TAG_WIDTH] = "11"
		col := NewIntegerColumn(fieldname, "INT", true, tagmap, isPointer)
		return &col
	case gotypes.Uint8Type:
		tagmap[TAG_WIDTH] = "4"
		col := NewIntegerColumn(fieldname, "TINYINT", true, tagmap, isPointer)
		return &col
	case gotypes.Uint16Type:
		tagmap[TAG_WIDTH] = "6"
		col := NewIntegerColumn(fieldname, "SMALLINT", true, tagmap, isPointer)
		return &col
	case gotypes.Uint64Type:
		tagmap[TAG_WIDTH] = "20"
		col := NewIntegerColumn(fieldname, "BIGINT", true, tagmap, isPointer)
		return &col
	case gotypes.BoolType:
		tagmap[TAG_WIDTH] = "1"
		col := NewBooleanColumn(fieldname, tagmap, isPointer)
		return &col
	case tristate.TriStateType:
		tagmap[TAG_WIDTH] = "1"
		col := NewTristateColumn(fieldname, tagmap, isPointer)
		return &col
	case gotypes.Float32Type, gotypes.Float64Type:
		if _, ok := tagmap[TAG_WIDTH]; ok {
			col := NewDecimalColumn(fieldname, tagmap, isPointer)
			return &col
		} else {
			colType := "FLOAT"
			if fieldType == gotypes.Float64Type {
				colType = "DOUBLE"
			}
			col := NewFloatColumn(fieldname, colType, tagmap, isPointer)
			return &col
		}
	case gotypes.TimeType:
		col := NewDateTimeColumn(fieldname, tagmap, isPointer)
		return &col
	default:
		if fieldType.Implements(gotypes.ISerializableType) {
			col := NewCompoundColumn(fieldname, tagmap, isPointer)
			return &col
		}
	}
	return nil
}

func struct2TableSpec(sv reflect.Value, table *STableSpec) {
	fields := reflectutils.FetchStructFieldValueSet(sv)
	for i := 0; i < len(fields); i += 1 {
		column := structField2ColumnSpec(&fields[i])
		if column != nil {
			if column.IsIndex() {
				table.AddIndex(column.IsUnique(), column.Name())
			}
			table.columns = append(table.columns, column)
		}
	}
}
