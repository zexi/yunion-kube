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
	"fmt"
	"math/bits"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
)

type SSqlColumnInfo struct {
	Field      string
	Type       string
	Collation  string
	Null       string
	Key        string
	Default    string
	Extra      string
	Privileges string
	Comment    string
}

func decodeSqlTypeString(typeStr string) []string {
	typeReg := regexp.MustCompile(`(\w+)\((\d+)(,\s*(\d+))?\)`)
	matches := typeReg.FindStringSubmatch(typeStr)
	if len(matches) >= 3 {
		return matches[1:]
	} else {
		return []string{typeStr}
	}
}

func (info *SSqlColumnInfo) toColumnSpec() IColumnSpec {
	tagmap := make(map[string]string)

	matches := decodeSqlTypeString(info.Type)
	typeStr := strings.ToUpper(matches[0])
	width := 0
	if len(matches) > 1 {
		width, _ = strconv.Atoi(matches[1])
	}
	if width > 0 {
		tagmap[TAG_WIDTH] = fmt.Sprintf("%d", width)
	}
	if info.Null == "YES" {
		tagmap[TAG_NULLABLE] = "true"
	} else {
		tagmap[TAG_NULLABLE] = "false"
	}
	if info.Key == "PRI" {
		tagmap[TAG_PRIMARY] = "true"
	} else {
		tagmap[TAG_PRIMARY] = "false"
	}
	charset := ""
	if info.Collation == "ascii_general_ci" {
		charset = "ascii"
	} else if info.Collation == "utf8_general_ci" {
		charset = "utf8"
	}
	if len(charset) > 0 {
		tagmap[TAG_CHARSET] = charset
	}
	if info.Default != "NULL" {
		tagmap[TAG_DEFAULT] = info.Default
	}
	if strings.HasSuffix(typeStr, "CHAR") {
		c := NewTextColumn(info.Field, tagmap, false)
		return &c
	} else if strings.HasSuffix(typeStr, "TEXT") {
		tagmap[TAG_TEXT_LENGTH] = typeStr[:len(typeStr)-4]
		c := NewTextColumn(info.Field, tagmap, false)
		return &c
	} else if strings.HasSuffix(typeStr, "INT") {
		if info.Extra == "auto_increment" {
			tagmap[TAG_AUTOINCREMENT] = "true"
		}
		unsigned := false
		if strings.HasSuffix(info.Type, " unsigned") {
			unsigned = true
		}
		c := NewIntegerColumn(info.Field, typeStr, unsigned, tagmap, false)
		return &c
	} else if typeStr == "FLOAT" || typeStr == "DOUBLE" {
		c := NewFloatColumn(info.Field, typeStr, tagmap, false)
		return &c
	} else if typeStr == "DECIMAL" {
		if len(matches) > 3 {
			precision, _ := strconv.Atoi(matches[3])
			if precision > 0 {
				tagmap[TAG_PRECISION] = fmt.Sprintf("%d", precision)
			}
		}
		c := NewDecimalColumn(info.Field, tagmap, false)
		return &c
	} else if typeStr == "DATETIME" {
		c := NewDateTimeColumn(info.Field, tagmap, false)
		return &c
	} else if typeStr == "DATE" || typeStr == "TIMESTAMP" {
		c := NewTimeTypeColumn(info.Field, typeStr, tagmap, false)
		return &c
	} else if strings.HasPrefix(typeStr, "ENUM(") {
		// enum type, force convert to text
		// discourage use of enum, use text instead
		enums := utils.FindWords([]byte(typeStr[5:len(typeStr)-1]), 0)

		width := 0
		for i := range enums {
			if width < len(enums[i]) {
				width = len(enums[i])
			}
		}
		tagmap[TAG_WIDTH] = fmt.Sprintf("%d", 1<<uint(bits.Len(uint(width))))
		c := NewTextColumn(info.Field, tagmap, false)
		return &c
	} else {
		log.Errorf("unsupported type %s", typeStr)
		return nil
	}
}

func (ts *STableSpec) fetchColumnDefs() ([]IColumnSpec, error) {
	sql := fmt.Sprintf("SHOW FULL COLUMNS IN `%s`", ts.name)
	query := NewRawQuery(sql, "field", "type", "collation", "null", "key", "default", "extra", "privileges", "comment")
	infos := make([]SSqlColumnInfo, 0)
	err := query.All(&infos)
	if err != nil {
		return nil, err
	}
	specs := make([]IColumnSpec, 0)
	for _, info := range infos {
		specs = append(specs, info.toColumnSpec())
	}
	return specs, nil
}

func (ts *STableSpec) fetchIndexesAndConstraints() ([]STableIndex, []STableConstraint, error) {
	sql := fmt.Sprintf("SHOW CREATE TABLE `%s`", ts.name)
	query := NewRawQuery(sql, "table", "create table")
	row := query.Row()
	var name, defStr string
	err := row.Scan(&name, &defStr)
	if err != nil {
		log.Errorf("fetch create table info fail %s", err)
		return nil, nil, err
	}
	indexes := parseIndexes(defStr)
	constraints := parseConstraints(defStr)
	return indexes, constraints, nil
}

func compareColumnSpec(c1, c2 IColumnSpec) int {
	return strings.Compare(c1.Name(), c2.Name())
}

func diffCols(tableName string, cols1 []IColumnSpec, cols2 []IColumnSpec) ([]IColumnSpec, []IColumnSpec, []IColumnSpec) {
	sort.Slice(cols1, func(i, j int) bool {
		return compareColumnSpec(cols1[i], cols1[j]) < 0
	})
	sort.Slice(cols2, func(i, j int) bool {
		return compareColumnSpec(cols2[i], cols2[j]) < 0
	})
	i := 0
	j := 0
	remove := make([]IColumnSpec, 0)
	update := make([]IColumnSpec, 0)
	add := make([]IColumnSpec, 0)
	for i < len(cols1) || j < len(cols2) {
		if i < len(cols1) && j < len(cols2) {
			comp := compareColumnSpec(cols1[i], cols2[j])
			if comp == 0 {
				if cols1[i].DefinitionString() != cols2[j].DefinitionString() {
					log.Infof("UPDATE %s: %s => %s", tableName, cols1[i].DefinitionString(), cols2[j].DefinitionString())
					update = append(update, cols2[j])
				}
				i += 1
				j += 1
			} else if comp > 0 {
				add = append(add, cols2[j])
				j += 1
			} else {
				remove = append(remove, cols1[i])
				i += 1
			}
		} else if i < len(cols1) {
			remove = append(remove, cols1[i])
			i += 1
		} else if j < len(cols2) {
			add = append(add, cols2[j])
			j += 1
		}
	}
	return remove, update, add
}

func diffIndexes2(exists []STableIndex, defs []STableIndex) (diff []STableIndex) {
	diff = make([]STableIndex, 0)
	for i := 0; i < len(exists); i += 1 {
		findDef := false
		for j := 0; j < len(defs); j += 1 {
			if defs[j].IsIdentical(exists[i].columns...) {
				findDef = true
				break
			}
		}
		if !findDef {
			diff = append(diff, exists[i])
		}
	}
	return
}

func diffIndexes(exists []STableIndex, defs []STableIndex) (added []STableIndex, removed []STableIndex) {
	return diffIndexes2(defs, exists), diffIndexes2(exists, defs)
}

func (ts *STableSpec) DropForeignKeySQL() []string {
	_, constraints, err := ts.fetchIndexesAndConstraints()
	if err != nil {
		log.Errorf("fetchIndexesAndConstraints fail %s", err)
		return nil
	}

	ret := make([]string, 0)
	for _, constraint := range constraints {
		sql := fmt.Sprintf("ALTER TABLE `%s` DROP FOREIGN KEY `%s`", ts.name, constraint.name)
		ret = append(ret, sql)
		log.Infof(sql)
	}

	return ret
}

func (ts *STableSpec) SyncSQL() []string {
	tables := GetTables()
	in, _ := utils.InStringArray(ts.name, tables)
	if !in {
		log.Debugf("table %s not created yet", ts.name)
		sql := ts.CreateSQL()
		return []string{sql}
	}

	indexes, _, err := ts.fetchIndexesAndConstraints()
	if err != nil {
		log.Errorf("fetchIndexesAndConstraints fail %s", err)
		return nil
	}

	ret := make([]string, 0)
	cols, err := ts.fetchColumnDefs()
	if err != nil {
		log.Errorf("fetchColumnDefs fail: %s", err)
		return nil
	}

	addIndexes, removeIndexes := diffIndexes(indexes, ts.indexes)

	for _, idx := range removeIndexes {
		sql := fmt.Sprintf("DROP INDEX `%s` ON `%s`", idx.name, ts.name)
		ret = append(ret, sql)
		log.Infof(sql)
	}

	alters := make([]string, 0)
	remove, update, add := diffCols(ts.name, cols, ts.columns)
	// first check if primary key is modifed
	changePrimary := false
	for _, col := range remove {
		if col.IsPrimary() {
			changePrimary = true
		}
	}
	// for _, col := range update {
	// 	if col.IsPrimary() {
	// 		changePrimary = true
	// 	}
	// }
	for _, col := range add {
		if col.IsPrimary() {
			changePrimary = true
		}
	}
	if changePrimary {
		sql := fmt.Sprintf("DROP PRIMARY KEY")
		alters = append(alters, sql)
	}
	/* IGNORE DROP STATEMENT */
	for _, col := range remove {
		sql := fmt.Sprintf("DROP COLUMN `%s`", col.Name())
		// alters = append(alters, sql)
		log.Infof("ALTER TABLE %s %s", ts.name, sql)
	}
	for _, col := range update {
		sql := fmt.Sprintf("MODIFY %s", col.DefinitionString())
		alters = append(alters, sql)
	}
	for _, col := range add {
		sql := fmt.Sprintf("ADD %s", col.DefinitionString())
		alters = append(alters, sql)
	}
	if changePrimary {
		primaries := make([]string, 0)
		for _, c := range ts.columns {
			if c.IsPrimary() {
				primaries = append(primaries, fmt.Sprintf("`%s`", c.Name()))
			}
		}
		sql := fmt.Sprintf("ADD PRIMARY KEY(%s)", strings.Join(primaries, ", "))
		alters = append(alters, sql)
	}

	if len(alters) > 0 {
		sql := fmt.Sprintf("ALTER TABLE `%s` %s;", ts.name, strings.Join(alters, ", "))
		ret = append(ret, sql)
	}

	for _, idx := range addIndexes {
		sql := fmt.Sprintf("CREATE INDEX `%s` ON `%s` (%s)", idx.name, ts.name, strings.Join(idx.QuotedColumns(), ","))
		ret = append(ret, sql)
		log.Infof(sql)
	}

	return ret
}

func (ts *STableSpec) Sync() error {
	sqls := ts.SyncSQL()
	if sqls != nil {
		for _, sql := range sqls {
			_, err := _db.Exec(sql)
			if err != nil {
				log.Errorf("exec sql error %s: %s", sql, err)
				return err
			}
		}
	}
	return nil
}

func (ts *STableSpec) CheckSync() {
	sqls := ts.SyncSQL()
	if len(sqls) > 0 {
		for _, sql := range sqls {
			fmt.Println(sql)
		}
		log.Fatalf("DB table %q not in sync", ts.name)
	}
}
