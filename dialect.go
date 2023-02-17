/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package zorm

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// wrapPageSQL 包装分页的SQL语句
// wrapPageSQL SQL statement for wrapping paging
func wrapPageSQL(dialect string, sqlstr *string, page *Page) error {
	if page.PageNo < 1 { // 默认第一页
		page.PageNo = 1
	}
	var sqlbuilder strings.Builder
	sqlbuilder.Grow(stringBuilderGrowLen)
	sqlbuilder.WriteString(*sqlstr)
	switch dialect {
	case "mysql", "sqlite", "dm", "gbase", "clickhouse", "tdengine", "db2": // MySQL,sqlite3,dm,南通,clickhouse,TDengine,db2 7.2+
		sqlbuilder.WriteString(" LIMIT ")
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize * (page.PageNo - 1)))
		sqlbuilder.WriteByte(',')
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize))

	case "postgresql", "kingbase", "shentong": // postgresql,kingbase,神通数据库
		sqlbuilder.WriteString(" LIMIT ")
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize))
		sqlbuilder.WriteString(" OFFSET ")
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize * (page.PageNo - 1)))
	case "mssql": // sqlserver 2012+
		locOrderBy := findOrderByIndex(sqlstr)
		if len(locOrderBy) < 1 { // 如果没有 order by,增加默认的排序
			sqlbuilder.WriteString(" ORDER BY (SELECT NULL) ")
		}
		sqlbuilder.WriteString(" OFFSET ")
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize * (page.PageNo - 1)))
		sqlbuilder.WriteString(" ROWS FETCH NEXT ")
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize))
		sqlbuilder.WriteString(" ROWS ONLY ")
	case "oracle": // oracle 12c+
		locOrderBy := findOrderByIndex(sqlstr)
		if len(locOrderBy) < 1 { // 如果没有 order by,增加默认的排序
			sqlbuilder.WriteString(" ORDER BY NULL ")
		}
		sqlbuilder.WriteString(" OFFSET ")
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize * (page.PageNo - 1)))
		sqlbuilder.WriteString(" ROWS FETCH NEXT ")
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize))
		sqlbuilder.WriteString(" ROWS ONLY ")
	default:
		return errors.New("->wrapPageSQL-->不支持的数据库类型:" + dialect)

	}
	*sqlstr = sqlbuilder.String()
	// return reBindSQL(dialect, sqlstr)
	return nil
}

// wrapInsertSQL  包装保存Struct语句.返回语句,是否自增,错误信息
// 数组传递,如果外部方法有调用append的逻辑，append会破坏指针引用，所以传递指针
// wrapInsertSQL Pack and save 'Struct' statement. Return  SQL statement, whether it is incremented, error message
// Array transfer, if the external method has logic to call append, append will destroy the pointer reference, so the pointer is passed
func wrapInsertSQL(ctx context.Context, typeOf *reflect.Type, entity IEntityStruct, columns *[]reflect.StructField, values *[]interface{}) (string, int, string, error) {
	sqlstr := ""
	inserColumnName, valuesql, autoIncrement, pktype, err := wrapInsertValueSQL(ctx, typeOf, entity, columns, values)
	if err != nil {
		return sqlstr, autoIncrement, pktype, err
	}

	var sqlBuilder strings.Builder
	// sqlBuilder.Grow(len(entity.GetTableName()) + len(inserColumnName) + len(entity.GetTableName()) + len(valuesql) + 19)
	sqlBuilder.Grow(stringBuilderGrowLen)
	// sqlstr := "INSERT INTO " + insersql + " VALUES" + valuesql
	sqlBuilder.WriteString("INSERT INTO ")
	sqlBuilder.WriteString(entity.GetTableName())
	sqlBuilder.WriteString(inserColumnName)
	sqlBuilder.WriteString(" VALUES")
	sqlBuilder.WriteString(valuesql)
	sqlstr = sqlBuilder.String()
	return sqlstr, autoIncrement, pktype, err
}

// wrapInsertValueSQL 包装保存Struct语句.返回语句,没有rebuild,返回原始的InsertSQL,ValueSQL,是否自增,主键类型,错误信息
// 数组传递,如果外部方法有调用append的逻辑,传递指针,因为append会破坏指针引用
// Pack and save Struct statement. Return  SQL statement, no rebuild, return original SQL, whether it is self-increment, error message
// Array transfer, if the external method has logic to call append, append will destroy the pointer reference, so the pointer is passed
func wrapInsertValueSQL(ctx context.Context, typeOf *reflect.Type, entity IEntityStruct, columns *[]reflect.StructField, values *[]interface{}) (string, string, int, string, error) {
	var inserColumnName, valuesql string
	// 自增类型  0(不自增),1(普通自增),2(序列自增)
	// Self-increment type： 0（Not increase）,1(Ordinary increment),2(Sequence increment)
	autoIncrement := 0
	// 主键类型
	// Primary key type
	pktype := ""
	// SQL语句的构造器
	// SQL statement constructor
	var sqlBuilder strings.Builder
	sqlBuilder.Grow(stringBuilderGrowLen)
	// sqlBuilder.WriteString(entity.GetTableName())
	sqlBuilder.WriteByte('(')

	// SQL语句中,VALUES(?,?,...)语句的构造器
	// In the SQL statement, the constructor of the VALUES(?,?,...) statement
	var valueSQLBuilder strings.Builder
	valueSQLBuilder.Grow(stringBuilderGrowLen)
	valueSQLBuilder.WriteString(" (")
	// 主键的名称
	// The name of the primary key.
	pkFieldName, e := entityPKFieldName(entity, typeOf)
	if e != nil {
		return inserColumnName, valuesql, autoIncrement, pktype, e
	}

	sequence := entity.GetPkSequence()
	if sequence != "" {
		// 序列自增 Sequence increment
		autoIncrement = 2
	}

	for i := 0; i < len(*columns); i++ {
		field := (*columns)[i]

		if field.Name == pkFieldName { // 如果是主键 | If it is the primary key
			// 获取主键类型 | Get the primary key type.
			pkKind := field.Type.Kind()
			switch pkKind {
			case reflect.String:
				pktype = "string"
			case reflect.Int, reflect.Int32, reflect.Int16, reflect.Int8:
				pktype = "int"
			case reflect.Int64:
				pktype = "int64"
			default:
				return inserColumnName, valuesql, autoIncrement, pktype, errors.New("->wrapInsertValueSQL不支持的主键类型")
			}

			// 主键的值
			// The value of the primary key
			pkValue := (*values)[i]
			valueIsZero := reflect.ValueOf(pkValue).IsZero()
			if autoIncrement == 2 { // 如果是序列自增 | If it is a sequence increment
				// 去掉这一列,后续不再处理
				// Remove this column and will not process it later.
				*columns = append((*columns)[:i], (*columns)[i+1:]...)
				*values = append((*values)[:i], (*values)[i+1:]...)
				i = i - 1
				if i > 0 { // i+1<len(*columns)会有风险:id是最后的字段,而且还是自增,被忽略了,但是前面的已经处理,是 逗号, 结尾的,就会bug,实际概率极低
					sqlBuilder.WriteByte(',')
					valueSQLBuilder.WriteByte(',')
				}
				colName := getFieldTagName(&field)
				sqlBuilder.WriteString(colName)
				valueSQLBuilder.WriteString(sequence)

				continue

			} else if valueIsZero && (pktype == "string") { // 主键是字符串类型,并且值为"",赋值id
				// 生成主键字符串
				// Generate primary key string
				id := FuncGenerateStringID(ctx)
				(*values)[i] = id
				// 给对象主键赋值
				// Assign a value to the primary key of the object
				v := reflect.ValueOf(entity).Elem()
				v.FieldByName(field.Name).Set(reflect.ValueOf(id))
			} else if valueIsZero && (pktype == "int" || pktype == "int64") {
				// 标记是自增主键
				// Mark is auto-incrementing primary key
				autoIncrement = 1
				// 去掉这一列,后续不再处理
				// Remove this column and will not process it later.
				*columns = append((*columns)[:i], (*columns)[i+1:]...)
				*values = append((*values)[:i], (*values)[i+1:]...)
				i = i - 1
				continue
			}
		}

		if i > 0 { // i+1<len(*columns)会有风险:id是最后的字段,而且还是自增,被忽略了,但是前面的已经处理,是 逗号, 结尾的,就会bug,实际概率极低
			sqlBuilder.WriteByte(',')
			valueSQLBuilder.WriteByte(',')
		}

		colName := getFieldTagName(&field)
		sqlBuilder.WriteString(colName)
		valueSQLBuilder.WriteByte('?')

	}

	sqlBuilder.WriteByte(')')
	valueSQLBuilder.WriteByte(')')
	inserColumnName = sqlBuilder.String()
	valuesql = valueSQLBuilder.String()
	return inserColumnName, valuesql, autoIncrement, pktype, nil
}

// wrapInsertSliceSQL 包装批量保存StructSlice语句.返回语句,是否自增,错误信息
// 数组传递,如果外部方法有调用append的逻辑，append会破坏指针引用，所以传递指针
// wrapInsertSliceSQL Package and save Struct Slice statements in batches. Return SQL statement, whether it is incremented, error message
// Array transfer, if the external method has logic to call append, append will destroy the pointer reference, so the pointer is passed
func wrapInsertSliceSQL(ctx context.Context, config *DataSourceConfig, typeOf *reflect.Type, entityStructSlice []IEntityStruct, columns *[]reflect.StructField, values *[]interface{}) (string, int, error) {
	sliceLen := len(entityStructSlice)
	sqlstr := ""
	if entityStructSlice == nil || sliceLen < 1 {
		return sqlstr, 0, errors.New("->wrapInsertSliceSQL对象数组不能为空")
	}

	// 第一个对象,获取第一个Struct对象,用于获取数据库字段,也获取了值
	// The first object, get the first Struct object, used to get the database field, and also get the value
	entity := entityStructSlice[0]

	// 先生成一条语句
	// Generate a statement first
	inserColumnName, valuesql, autoIncrement, _, firstErr := wrapInsertValueSQL(ctx, typeOf, entity, columns, values)
	if firstErr != nil {
		return sqlstr, autoIncrement, firstErr
	}
	var sqlBuilder strings.Builder
	// sqlBuilder.Grow(len(entity.GetTableName()) + len(inserColumnName) + len(entity.GetTableName()) + len(valuesql) + 19)
	sqlBuilder.Grow(stringBuilderGrowLen)
	sqlBuilder.WriteString("INSERT INTO ")
	sqlBuilder.WriteString(entity.GetTableName())
	// sqlstr := "INSERT INTO "
	if config.Dialect == "tdengine" && !config.TDengineInsertsColumnName { // 如果是tdengine,拼接类似 INSERT INTO table1 values('2','3')  table2 values('4','5'),目前要求字段和类型必须一致,如果不一致,改动略多
	} else {
		// sqlstr = sqlstr + insertsql + " VALUES" + valuesql
		sqlBuilder.WriteString(inserColumnName)
	}
	sqlBuilder.WriteString(" VALUES")
	sqlBuilder.WriteString(valuesql)
	// 如果只有一个Struct对象
	// If there is only one Struct object
	if sliceLen == 1 {
		return sqlBuilder.String(), autoIncrement, firstErr
	}
	// 主键的名称
	// The name of the primary key
	pkFieldName, e := entityPKFieldName(entity, typeOf)
	if e != nil {
		return sqlBuilder.String(), autoIncrement, e
	}

	for i := 1; i < sliceLen; i++ {
		// 拼接字符串
		// Splicing string
		if config.Dialect == "tdengine" { // 如果是tdengine,拼接类似 INSERT INTO table1 values('2','3')  table2 values('4','5'),目前要求字段和类型必须一致,如果不一致,改动略多
			sqlBuilder.WriteByte(' ')
			sqlBuilder.WriteString(entityStructSlice[i].GetTableName())
			if config.TDengineInsertsColumnName {
				sqlBuilder.WriteString(inserColumnName)
			}
			sqlBuilder.WriteString(" VALUES")
			sqlBuilder.WriteString(valuesql)
		} else { // 标准语法 类似 INSERT INTO table1(id,name) values('2','3'),('4','5')
			sqlBuilder.WriteByte(',')
			sqlBuilder.WriteString(valuesql)
		}

		entityStruct := entityStructSlice[i]
		for j := 0; j < len(*columns); j++ {
			// 获取实体类的反射,指针下的struct
			// Get the reflection of the entity class, the struct under the pointer
			valueOf := reflect.ValueOf(entityStruct).Elem()
			field := (*columns)[j]
			// 字段的值
			// The value of the primary key
			fieldValue := valueOf.FieldByName(field.Name)
			if field.Name == pkFieldName { // 如果是主键 ｜ If it is the primary key
				pkKind := field.Type.Kind()
				// pkValue := valueOf.FieldByName(field.Name).Interface()
				// 只处理字符串类型的主键,其他类型,columns中并不包含
				// Only handle primary keys of string type, other types, not included in columns
				if (pkKind == reflect.String) && fieldValue.IsZero() {
					// 主键是字符串类型,并且值为"",赋值'id'
					// 生成主键字符串
					// The primary key is a string type, and the value is "", assigned the value'id'
					// Generate primary key string
					id := FuncGenerateStringID(ctx)
					*values = append(*values, id)
					// 给对象主键赋值
					// Assign a value to the primary key of the object
					fieldValue.Set(reflect.ValueOf(id))
					continue
				}
			}

			// 给字段赋值
			// Assign a value to the field.
			*values = append(*values, fieldValue.Interface())

		}
	}

	sqlstr = sqlBuilder.String()
	return sqlstr, autoIncrement, nil
}

// wrapInsertEntityMapSliceSQL 包装批量保存EntityMapSlice语句.返回语句,值,错误信息
func wrapInsertEntityMapSliceSQL(ctx context.Context, config *DataSourceConfig, entityMapSlice []IEntityMap) (string, []interface{}, error) {
	sliceLen := len(entityMapSlice)
	sqlstr := ""
	if entityMapSlice == nil || sliceLen < 1 {
		return sqlstr, nil, errors.New("->wrapInsertSliceSQL对象数组不能为空")
	}
	// 第一个对象,获取第一个Struct对象,用于获取数据库字段,也获取了值
	entity := entityMapSlice[0]
	// 检查是否是指针对象
	_, err := checkEntityKind(entity)
	if err != nil {
		return sqlstr, nil, err
	}
	dbFieldMapKey := entity.GetDBFieldMapKey()
	// SQL语句
	inserColumnName, valuesql, values, _, err := wrapInsertValueEntityMapSQL(entity)
	if err != nil {
		return sqlstr, values, err
	}

	var sqlBuilder strings.Builder
	// sqlBuilder.Grow(len(entity.GetTableName()) + len(inserColumnName) + len(valuesql) + 19)
	sqlBuilder.Grow(stringBuilderGrowLen)
	sqlBuilder.WriteString("INSERT INTO ")
	sqlBuilder.WriteString(entity.GetTableName())
	// sqlstr = sqlstr + insertsql + " VALUES" + valuesql
	sqlBuilder.WriteString(inserColumnName)
	sqlBuilder.WriteString(" VALUES")
	sqlBuilder.WriteString(valuesql)
	for i := 1; i < sliceLen; i++ {
		// 拼接字符串
		// Splicing string
		if config.Dialect == "tdengine" { // 如果是tdengine,拼接类似 INSERT INTO table1 values('2','3')  table2 values('4','5'),目前要求字段和类型必须一致,如果不一致,改动略多
			sqlBuilder.WriteByte(' ')
			sqlBuilder.WriteString(entityMapSlice[i].GetTableName())
			if config.TDengineInsertsColumnName {
				sqlBuilder.WriteString(inserColumnName)
			}
			sqlBuilder.WriteString(" VALUES")
			sqlBuilder.WriteString(valuesql)
		} else { // 标准语法 类似 INSERT INTO table1(id,name) values('2','3'), values('4','5')
			sqlBuilder.WriteByte(',')
			sqlBuilder.WriteString(valuesql)
		}

		entityMap := entityMapSlice[i]
		for j := 0; j < len(dbFieldMapKey); j++ {
			key := dbFieldMapKey[j]
			value := entityMap.GetDBFieldMap()[key]
			values = append(values, value)
		}
	}

	sqlstr = sqlBuilder.String()
	return sqlstr, values, err
}

// wrapUpdateSQL 包装更新Struct语句
// 数组传递,如果外部方法有调用append的逻辑，append会破坏指针引用，所以传递指针
// wrapUpdateSQL Package update Struct statement
// Array transfer, if the external method has logic to call append, append will destroy the pointer reference, so the pointer is passed
func wrapUpdateSQL(typeOf *reflect.Type, entity IEntityStruct, columns *[]reflect.StructField, values *[]interface{}, onlyUpdateNotZero bool) (string, error) {
	sqlstr := ""
	// SQL语句的构造器
	// SQL statement constructor
	var sqlBuilder strings.Builder
	sqlBuilder.Grow(stringBuilderGrowLen)
	sqlBuilder.WriteString("UPDATE ")
	sqlBuilder.WriteString(entity.GetTableName())
	sqlBuilder.WriteString(" SET ")

	// 主键的值
	// The value of the primary key
	var pkValue interface{}
	// 主键的名称
	// The name of the primary key
	pkFieldName, e := entityPKFieldName(entity, typeOf)
	if e != nil {
		return sqlstr, e
	}

	for i := 0; i < len(*columns); i++ {
		field := (*columns)[i]
		if field.Name == pkFieldName {
			// 如果是主键
			// If it is the primary key.
			pkValue = (*values)[i]
			// 去掉这一列,最后处理主键
			// Remove this column, and finally process the primary key
			*columns = append((*columns)[:i], (*columns)[i+1:]...)
			*values = append((*values)[:i], (*values)[i+1:]...)
			i = i - 1
			continue
		}

		// 如果是默认值字段,删除掉,不更新
		// If it is the default value field, delete it and do not update
		if onlyUpdateNotZero && (reflect.ValueOf((*values)[i]).IsZero()) {
			// 去掉这一列,不再处理
			// Remove this column and no longer process
			*columns = append((*columns)[:i], (*columns)[i+1:]...)
			*values = append((*values)[:i], (*values)[i+1:]...)
			i = i - 1
			continue

		}
		if i > 0 {
			sqlBuilder.WriteByte(',')
		}
		colName := getFieldTagName(&field)
		sqlBuilder.WriteString(colName)
		sqlBuilder.WriteString("=?")

	}
	// 主键的值是最后一个
	// The value of the primary key is the last
	*values = append(*values, pkValue)

	// sqlstr = sqlstr + " WHERE " + entity.GetPKColumnName() + "=?"
	sqlBuilder.WriteString(" WHERE ")
	sqlBuilder.WriteString(entity.GetPKColumnName())
	sqlBuilder.WriteString("=?")
	sqlstr = sqlBuilder.String()

	return sqlstr, nil
}

// wrapDeleteSQL 包装删除Struct语句
// wrapDeleteSQL Package delete Struct statement
func wrapDeleteSQL(entity IEntityStruct) (string, error) {
	// SQL语句的构造器
	// SQL statement constructor
	var sqlBuilder strings.Builder
	sqlBuilder.Grow(stringBuilderGrowLen)
	sqlBuilder.WriteString("DELETE FROM ")
	sqlBuilder.WriteString(entity.GetTableName())
	sqlBuilder.WriteString(" WHERE ")
	sqlBuilder.WriteString(entity.GetPKColumnName())
	sqlBuilder.WriteString("=?")
	sqlstr := sqlBuilder.String()

	return sqlstr, nil
}

// wrapInsertEntityMapSQL 包装保存Map语句,Map因为没有字段属性,无法完成Id的类型判断和赋值,需要确保Map的值是完整的
// wrapInsertEntityMapSQL Pack and save the Map statement. Because Map does not have field attributes,
// it cannot complete the type judgment and assignment of Id. It is necessary to ensure that the value of Map is complete
func wrapInsertEntityMapSQL(entity IEntityMap) (string, []interface{}, bool, error) {
	sqlstr := ""
	inserColumnName, valuesql, values, autoIncrement, err := wrapInsertValueEntityMapSQL(entity)
	if err != nil {
		return sqlstr, nil, autoIncrement, err
	}
	// 拼接SQL语句,带上列名,因为Map取值是无序的
	// sqlstr := "INSERT INTO " + insertsql + " VALUES" + valuesql

	var sqlBuilder strings.Builder
	// sqlBuilder.Grow(len(inserColumnName) + len(entity.GetTableName()) + len(valuesql) + 19)
	sqlBuilder.Grow(stringBuilderGrowLen)
	sqlBuilder.WriteString("INSERT INTO ")
	sqlBuilder.WriteString(entity.GetTableName())
	sqlBuilder.WriteString(inserColumnName)
	sqlBuilder.WriteString(" VALUES")
	sqlBuilder.WriteString(valuesql)
	sqlstr = sqlBuilder.String()

	return sqlstr, values, autoIncrement, nil
}

// wrapInsertValueEntityMapSQL 包装保存Map语句,Map因为没有字段属性,无法完成Id的类型判断和赋值,需要确保Map的值是完整的
// wrapInsertValueEntityMapSQL Pack and save the Map statement. Because Map does not have field attributes,
// it cannot complete the type judgment and assignment of Id. It is necessary to ensure that the value of Map is complete
func wrapInsertValueEntityMapSQL(entity IEntityMap) (string, string, []interface{}, bool, error) {
	var inserColumnName, valuesql string
	// 是否自增,默认false
	autoIncrement := false
	dbFieldMap := entity.GetDBFieldMap()
	if len(dbFieldMap) < 1 {
		return inserColumnName, inserColumnName, nil, autoIncrement, errors.New("->wrapInsertEntityMapSQL-->GetDBFieldMap返回值不能为空")
	}
	// SQL对应的参数
	// SQL corresponding parameters
	values := []interface{}{}

	// SQL语句的构造器
	// SQL statement constructor
	var sqlBuilder strings.Builder
	sqlBuilder.Grow(stringBuilderGrowLen)
	// sqlBuilder.WriteString("INSERT INTO ")
	// sqlBuilder.WriteString(entity.GetTableName())
	sqlBuilder.WriteByte('(')

	// SQL语句中,VALUES(?,?,...)语句的构造器
	// In the SQL statement, the constructor of the VALUES(?,?,...) statement.
	var valueSQLBuilder strings.Builder
	valueSQLBuilder.Grow(stringBuilderGrowLen)
	valueSQLBuilder.WriteString(" (")
	// 是否Set了主键
	// Whether the primary key is set.
	_, hasPK := dbFieldMap[entity.GetPKColumnName()]
	if entity.GetPKColumnName() != "" && !hasPK { // 如果有主键字段,却没值,认为是自增或者序列 | If the primary key is not set, it is considered to be auto-increment or sequence
		autoIncrement = true
		if entity.GetEntityMapPkSequence() != "" { // 如果是序列 | If it is a sequence.
			sqlBuilder.WriteString(entity.GetPKColumnName())
			valueSQLBuilder.WriteString(entity.GetEntityMapPkSequence())
			if len(dbFieldMap) > 1 { // 如果不只有序列
				sqlBuilder.WriteByte(',')
				valueSQLBuilder.WriteByte(',')
			}

		}
	}

	dbFieldMapKey := entity.GetDBFieldMapKey()
	for dbFieldMapIndex := 0; dbFieldMapIndex < len(dbFieldMapKey); dbFieldMapIndex++ {
		if dbFieldMapIndex > 0 {
			sqlBuilder.WriteByte(',')
			valueSQLBuilder.WriteByte(',')
		}
		k := dbFieldMapKey[dbFieldMapIndex]
		v := dbFieldMap[k]
		// 拼接字符串
		// Concatenated string
		sqlBuilder.WriteString(k)
		valueSQLBuilder.WriteByte('?')
		values = append(values, v)
	}

	sqlBuilder.WriteByte(')')
	valueSQLBuilder.WriteByte(')')
	inserColumnName = sqlBuilder.String()
	valuesql = valueSQLBuilder.String()

	return inserColumnName, valuesql, values, autoIncrement, nil
}

// wrapUpdateEntityMapSQL 包装Map更新语句,Map因为没有字段属性,无法完成Id的类型判断和赋值,需要确保Map的值是完整的
// wrapUpdateEntityMapSQL Wrap the Map update statement. Because Map does not have field attributes,
// it cannot complete the type judgment and assignment of Id. It is necessary to ensure that the value of Map is complete
func wrapUpdateEntityMapSQL(entity IEntityMap) (string, []interface{}, error) {
	dbFieldMap := entity.GetDBFieldMap()
	sqlstr := ""
	if len(dbFieldMap) < 1 {
		return sqlstr, nil, errors.New("->wrapUpdateEntityMapSQL-->GetDBFieldMap返回值不能为空")
	}
	// SQL语句的构造器
	// SQL statement constructor
	var sqlBuilder strings.Builder
	sqlBuilder.Grow(stringBuilderGrowLen)
	sqlBuilder.WriteString("UPDATE ")
	sqlBuilder.WriteString(entity.GetTableName())
	sqlBuilder.WriteString(" SET ")

	// SQL对应的参数
	// SQL corresponding parameters
	values := []interface{}{}
	// 主键名称
	// Primary key name
	var pkValue interface{}
	dbFieldMapIndex := 0
	for k, v := range dbFieldMap {

		if k == entity.GetPKColumnName() { // 如果是主键  | If it is the primary key
			pkValue = v
			continue
		}
		if dbFieldMapIndex > 0 {
			sqlBuilder.WriteByte(',')
		}

		// 拼接字符串 | Splicing string.
		sqlBuilder.WriteString(k)
		sqlBuilder.WriteString("=?")
		values = append(values, v)
		dbFieldMapIndex++
	}
	// 主键的值是最后一个
	// The value of the primary key is the last
	values = append(values, pkValue)

	sqlBuilder.WriteString(" WHERE ")
	sqlBuilder.WriteString(entity.GetPKColumnName())
	sqlBuilder.WriteString("=?")
	sqlstr = sqlBuilder.String()

	return sqlstr, values, nil
}

// wrapQuerySQL 封装查询语句
// wrapQuerySQL Encapsulated query statement
func wrapQuerySQL(dialect string, finder *Finder, page *Page) (string, error) {
	// 获取到没有page的sql的语句
	// Get the SQL statement without page.
	sqlstr, err := finder.GetSQL()
	if err != nil {
		return "", err
	}
	if page != nil {
		err = wrapPageSQL(dialect, &sqlstr, page)
	}
	if err != nil {
		return "", err
	}
	return sqlstr, err
}

// 查询'order by'在sql中出现的开始位置和结束位置
// Query the start position and end position of'order by' in SQL
var (
	orderByExpr      = "(?i)\\s(order)\\s+by\\s"
	orderByRegexp, _ = regexp.Compile(orderByExpr)
)

// findOrderByIndex 查询order by在sql中出现的开始位置和结束位置
// findOrderByIndex Query the start position and end position of'order by' in SQL
func findOrderByIndex(strsql *string) []int {
	loc := orderByRegexp.FindStringIndex(*strsql)
	return loc
}

// 查询'group by'在sql中出现的开始位置和结束位置
// Query the start position and end position of'group by' in sql。
var (
	groupByExpr      = "(?i)\\s(group)\\s+by\\s"
	groupByRegexp, _ = regexp.Compile(groupByExpr)
)

// findGroupByIndex 查询group by在sql中出现的开始位置和结束位置
// findGroupByIndex Query the start position and end position of'group by' in sql
func findGroupByIndex(strsql *string) []int {
	loc := groupByRegexp.FindStringIndex(*strsql)
	return loc
}

// 查询 from 在sql中出现的开始位置和结束位置
// Query the start position and end position of 'from' in sql
// var fromExpr = "(?i)(^\\s*select)(.+?\\(.+?\\))*.*?(from)"
// 感谢奔跑(@zeqjone)提供的正则,排除不在括号内的from,已经满足绝大部分场景,
// select id1,(select (id2) from t1 where id=2) _s FROM table select的子查询 _s中的 id2还有括号,才会出现问题,建议使用CountFinder处理分页语句
// countFinder := zorm.NewFinder().Append("select count(*) from (")
// countFinder.AppendFinder(finder)
// countFinder.Append(") tempcountfinder")
// finder.CountFinder = countFinder
var (
	fromExpr      = "(?i)(^\\s*select)(\\(.*?\\)|[^()]+)*?(from)"
	fromRegexp, _ = regexp.Compile(fromExpr)
)

// findFromIndexa 查询from在sql中出现的开始位置和结束位置
// findSelectFromIndex Query the start position and end position of 'from' in sql
func findSelectFromIndex(strsql *string) []int {
	// 匹配出来的是完整的字符串,用最后的FROM即可
	loc := fromRegexp.FindStringIndex(*strsql)
	if len(loc) < 2 {
		return loc
	}
	// 最后的FROM前推4位字符串
	loc[0] = loc[1] - 4
	return loc
}

/*
var fromExpr = `\(([\s\S]+?)\)`
var fromRegexp, _ = regexp.Compile(fromExpr)

//查询 from 在sql中出现的开始位置
//Query the start position of 'from' in sql
func findSelectFromIndex(strsql string) int {
	sql := strings.ToLower(strsql)
	m := fromRegexp.FindAllString(sql, -1)
	for i := 0; i < len(m); i++ {
		str := m[i]
		strnofrom := strings.ReplaceAll(str, " from ", " zorm ")
		sql = strings.ReplaceAll(sql, str, strnofrom)
	}
	fromIndex := strings.LastIndex(sql, " from ")
	if fromIndex < 0 {
		return fromIndex
	}
	//补上一个空格
	fromIndex = fromIndex + 1
	return fromIndex
}
*/
/*
// 从更新语句中获取表名
//update\\s(.+)set\\s.*
var (
	updateExper     = "(?i)^\\s*update\\s+(\\w+)\\s+set\\s"
	updateRegexp, _ = regexp.Compile(updateExper)
)

// findUpdateTableName 获取语句中表名
// 第一个是符合的整体数据,第二个是表名
func findUpdateTableName(strsql *string) []string {
	matchs := updateRegexp.FindStringSubmatch(*strsql)
	return matchs
}

// 从删除语句中获取表名
// delete\\sfrom\\s(.+)where\\s(.*)
var (
	deleteExper     = "(?i)^\\s*delete\\s+from\\s+(\\w+)\\s+where\\s"
	deleteRegexp, _ = regexp.Compile(deleteExper)
)

// findDeleteTableName 获取语句中表名
// 第一个是符合的整体数据,第二个是表名
func findDeleteTableName(strsql *string) []string {
	matchs := deleteRegexp.FindStringSubmatch(*strsql)
	return matchs
}
*/

// FuncGenerateStringID 默认生成字符串ID的函数.方便自定义扩展
// FuncGenerateStringID Function to generate string ID by default. Convenient for custom extension
var FuncGenerateStringID = func(ctx context.Context) string {
	// 使用 crypto/rand 真随机9位数
	randNum, randErr := rand.Int(rand.Reader, big.NewInt(1000000000))
	if randErr != nil {
		return ""
	}
	// 获取9位数,前置补0,确保9位数
	rand9 := fmt.Sprintf("%09d", randNum)

	// 获取纳秒 按照 年月日时分秒毫秒微秒纳秒 拼接为长度23位的字符串
	pk := time.Now().Format("2006.01.02.15.04.05.000000000")
	pk = strings.ReplaceAll(pk, ".", "")

	// 23位字符串+9位随机数=32位字符串,这样的好处就是可以使用ID进行排序
	pk = pk + rand9
	return pk
}

// FuncWrapFieldTagName 用于包裹字段名, eg. `describe`
var FuncWrapFieldTagName = func(colName string) string {
	// custom: return fmt.Sprintf("`%s`", colName)
	return colName
}

// getFieldTagName 获取模型中定义的数据库的 column tag
func getFieldTagName(field *reflect.StructField) string {
	colName := field.Tag.Get(tagColumnName)
	colName = FuncWrapFieldTagName(colName)
	/*
		if dialect == "kingbase" {
			// kingbase R3 驱动大小写敏感，通常是大写。数据库全的列名部换成双引号括住的大写字符，避免与数据库内置关键词冲突时报错
			colName = strings.ReplaceAll(colName, "\"", "")
			colName = fmt.Sprintf(`"%s"`, strings.ToUpper(colName))
		}
	*/
	return colName
}

// wrapSQLHint 在sql语句中增加hint
func wrapSQLHint(ctx context.Context, sqlstr *string) error {
	// 获取hint
	contextValue := ctx.Value(contextSQLHintValueKey)
	if contextValue == nil { // 如果没有设置hint
		return nil
	}
	hint, ok := contextValue.(string)
	if !ok {
		return errors.New("->wrapSQLHint-->contextValue转换string失败")
	}
	if hint == "" {
		return nil
	}
	sqlByte := []byte(*sqlstr)
	// 获取第一个单词
	_, start, end, err := firstOneWord(0, &sqlByte)
	if err != nil {
		return err
	}
	if start == -1 || end == -1 { // 未取到字符串
		return nil
	}
	var sqlBuilder strings.Builder
	sqlBuilder.Grow(stringBuilderGrowLen)
	sqlBuilder.WriteString((*sqlstr)[:end])
	sqlBuilder.WriteByte(' ')
	sqlBuilder.WriteString(hint)
	sqlBuilder.WriteString((*sqlstr)[end:])
	*sqlstr = sqlBuilder.String()
	return nil
}

// reBindSQL 包装基础的SQL语句,根据数据库类型,调整SQL变量符号,例如?,? $1,$2这样的
// reBindSQL Pack basic SQL statements, adjust the SQL variable symbols according to the database type, such as?,? $1,$2
func reBindSQL(dialect string, sqlstr *string, args *[]interface{}) (*string, *[]interface{}, error) {
	argsNum := len(*args)
	if argsNum < 1 { // 没有参数,不需要处理,也不判断参数数量了,数据库会报错提示
		return sqlstr, args, nil
	}
	// 重新记录参数值
	// Re-record the parameter value
	newValues := make([]interface{}, 0)
	// 记录sql参数值的下标,例如 $1 @p1 ,从1开始
	sqlParamIndex := 1

	// 新的sql
	// new sql
	var newSQLStr strings.Builder
	// newSQLStr.Grow(len(*sqlstr))
	newSQLStr.Grow(stringBuilderGrowLen)
	i := -1
	for _, v := range []byte(*sqlstr) {
		if v != '?' { // 如果不是?问号
			newSQLStr.WriteByte(v)
			continue
		}
		i = i + 1
		if i >= argsNum { // 占位符数量比参数值多,不使用 strings.Count函数,避免多次操作字符串
			return nil, nil, fmt.Errorf("sql语句中参数和值数量不一致,-->zormErrorExecSQL:%s,-->zormErrorSQLValues:%v", *sqlstr, *args)
		}
		v := (*args)[i]
		// 反射获取参数的值
		valueOf := reflect.ValueOf(v)
		// 获取类型
		kind := valueOf.Kind()
		// 如果参数是个指针类型
		// If the parameter is a pointer type
		if kind == reflect.Ptr { // 如果是指针 ｜ If it is a pointer
			valueOf = valueOf.Elem()
			kind = valueOf.Kind()
		}
		typeOf := valueOf.Type()
		// 参数值长度,默认是1,其他取值数组长度
		valueLen := 1

		// 如果不是数组或者slice
		// If it is not an array or slice
		if !(kind == reflect.Array || kind == reflect.Slice) {
			// 记录新值
			// Record new value.
			newValues = append(newValues, v)
		} else if typeOf == reflect.TypeOf([]byte{}) {
			// 记录新值
			// Record new value
			newValues = append(newValues, v)
		} else {
			// 如果不是字符串类型的值,无法取长度,这个是个bug,先注释了
			// 获取数组类型参数值的长度
			// If it is not a string type value, the length cannot be taken, this is a bug, first comment
			// Get the length of the array type parameter value
			valueLen = valueOf.Len()
			// 数组类型的参数长度小于1,认为是有异常的参数
			// The parameter length of the array type is less than 1, which is considered to be an abnormal parameter
			if valueLen < 1 {
				return nil, nil, errors.New("->reBindSQL()语句:" + *sqlstr + ",第" + strconv.Itoa(i+1) + "个参数,类型是Array或者Slice,值的长度为0,请检查sql参数有效性")
			} else if valueLen == 1 { // 如果数组里只有一个参数,认为是单个参数
				v = valueOf.Index(0).Interface()
				newValues = append(newValues, v)
			}

		}

		switch dialect {
		case "mysql", "sqlite", "dm", "gbase", "clickhouse", "db2":
			wrapParamSQL("?", valueLen, &sqlParamIndex, &newSQLStr, &valueOf, &newValues, false, false)
		case "postgresql", "kingbase": // postgresql,kingbase
			wrapParamSQL("$", valueLen, &sqlParamIndex, &newSQLStr, &valueOf, &newValues, true, false)
		case "mssql": // mssql
			wrapParamSQL("@p", valueLen, &sqlParamIndex, &newSQLStr, &valueOf, &newValues, true, false)
		case "oracle", "shentong": // oracle,神通
			wrapParamSQL(":", valueLen, &sqlParamIndex, &newSQLStr, &valueOf, &newValues, true, false)
		case "tdengine": // tdengine,重新处理 字符类型的参数 '?'
			wrapParamSQL("?", valueLen, &sqlParamIndex, &newSQLStr, &valueOf, &newValues, false, true)
		default: // 其他情况,还是使用 ? | In other cases, or use  ?
			newSQLStr.WriteByte('?')
		}

	}

	//?号占位符的数量和参数不一致,不使用 strings.Count函数,避免多次操作字符串
	if (i + 1) != argsNum {
		return nil, nil, fmt.Errorf("sql语句中参数和值数量不一致,-->zormErrorExecSQL:%s,-->zormErrorSQLValues:%v", *sqlstr, *args)
	}
	sqlstring := newSQLStr.String()
	return &sqlstring, &newValues, nil
}

// reUpdateFinderSQL 根据数据类型更新 手动编写的 UpdateFinder的语句,用于处理数据库兼容,例如 clickhouse的 UPDATE 和 DELETE
func reUpdateSQL(dialect string, sqlstr *string) error {
	if dialect != "clickhouse" { // 目前只处理clickhouse
		return nil
	}
	// 处理clickhouse的特殊更新语法
	sqlByte := []byte(*sqlstr)
	// 获取第一个单词
	firstWord, start, end, err := firstOneWord(0, &sqlByte)
	if err != nil {
		return err
	}
	if start == -1 || end == -1 { // 未取到字符串
		return nil
	}
	// SQL语句的构造器
	// SQL statement constructor
	var sqlBuilder strings.Builder
	sqlBuilder.Grow(stringBuilderGrowLen)
	sqlBuilder.WriteString((*sqlstr)[:start])
	sqlBuilder.WriteString("ALTER TABLE ")
	firstWord = strings.ToUpper(firstWord)
	tableName := ""
	if firstWord == "UPDATE" { // 更新  update tableName set
		tableName, _, end, err = firstOneWord(end, &sqlByte)
		if err != nil {
			return err
		}
		// 拿到 set
		_, start, end, err = firstOneWord(end, &sqlByte)

	} else if firstWord == "DELETE" { // 删除 delete from tableName
		// 拿到from
		_, _, end, err = firstOneWord(end, &sqlByte)
		if err != nil {
			return err
		}
		// 拿到 tableName
		tableName, start, end, err = firstOneWord(end, &sqlByte)
	} else { // 只处理UPDATE 和 DELETE 语法
		return nil
	}
	if err != nil {
		return err
	}
	if start == -1 || end == -1 { // 获取的位置异常
		return errors.New("->reUpdateSQL中clickhouse语法异常,请检查sql语句是否标准,-->zormErrorExecSQL:" + *sqlstr)
	}
	sqlBuilder.WriteString(tableName)
	sqlBuilder.WriteByte(' ')
	sqlBuilder.WriteString(firstWord)
	// sqlBuilder.WriteByte(' ')
	sqlBuilder.WriteString((*sqlstr)[end:])
	*sqlstr = sqlBuilder.String()
	return nil
}

// wrapAutoIncrementInsertSQL 包装自增的自增主键的插入sql
func wrapAutoIncrementInsertSQL(pkColumnName string, sqlstr *string, dialect string, values *[]interface{}) (*int64, *int64) {
	// oracle 12c+ 支持IDENTITY属性的自增列,因为分页也要求12c+的语法,所以数据库就IDENTITY创建自增吧
	// 处理序列产生的自增主键,例如oracle,postgresql等
	var lastInsertID, zormSQLOutReturningID *int64
	var sqlBuilder strings.Builder
	// sqlBuilder.Grow(len(*sqlstr) + len(pkColumnName) + 40)
	sqlBuilder.Grow(stringBuilderGrowLen)
	sqlBuilder.WriteString(*sqlstr)
	switch dialect {
	case "postgresql", "kingbase":
		var p int64 = 0
		lastInsertID = &p
		// sqlstr = sqlstr + " RETURNING " + pkColumnName
		sqlBuilder.WriteString(" RETURNING ")
		sqlBuilder.WriteString(pkColumnName)
	case "oracle", "shentong":
		var p int64 = 0
		zormSQLOutReturningID = &p
		// sqlstr = sqlstr + " RETURNING " + pkColumnName + " INTO :zormSQLOutReturningID "
		sqlBuilder.WriteString(" RETURNING ")
		sqlBuilder.WriteString(pkColumnName)
		sqlBuilder.WriteString(" INTO :zormSQLOutReturningID ")
		v := sql.Named("zormSQLOutReturningID", sql.Out{Dest: zormSQLOutReturningID})
		*values = append(*values, v)
	}

	*sqlstr = sqlBuilder.String()
	return lastInsertID, zormSQLOutReturningID
}

// getConfigFromConnection 从dbConnection中获取数据库方言,如果没有,从FuncReadWriteStrategy获取dbDao,获取dbdao.config.Dialect
func getConfigFromConnection(ctx context.Context, dbConnection *dataBaseConnection, rwType int) (*DataSourceConfig, error) {
	var config *DataSourceConfig
	// dbConnection为nil,使用defaultDao
	// dbConnection is nil, use default Dao
	if dbConnection == nil {
		dbdao, err := FuncReadWriteStrategy(ctx, rwType)
		if err != nil {
			return nil, err
		}
		config = dbdao.config
	} else {
		config = dbConnection.config
	}
	return config, nil
}

// wrapParamSQL 包装SQL语句
// symbols(占位符) valueLen(参数长度) sqlParamIndexPtr(参数的下标指针,数组会改变值) newSQLStr(SQL字符串Builder) valueOf(参数值的反射对象) hasParamIndex(是否拼接参数下标 $1 $2) isTDengine(TDengine数据库需要单独处理字符串类型)
func wrapParamSQL(symbols string, valueLen int, sqlParamIndexPtr *int, newSQLStr *strings.Builder, valueOf *reflect.Value, newValues *[]interface{}, hasParamIndex bool, isTDengine bool) {
	sqlParamIndex := *sqlParamIndexPtr
	if valueLen == 1 {
		if isTDengine && valueOf.Kind() == reflect.String { // 处理tdengine的字符串类型
			symbols = "'?'"
		}
		newSQLStr.WriteString(symbols)

		if hasParamIndex {
			newSQLStr.WriteString(strconv.Itoa(sqlParamIndex))
		}

	} else if valueLen > 1 { // 如果值是数组
		for j := 0; j < valueLen; j++ {
			valuej := (*valueOf).Index(j)
			if isTDengine && valuej.Kind() == reflect.String { // 处理tdengine的字符串类型
				symbols = "'?'"
			}
			if j == 0 { // 第一个
				newSQLStr.WriteString(symbols)
			} else {
				newSQLStr.WriteByte(',')
				newSQLStr.WriteString(symbols)
			}
			if hasParamIndex {
				newSQLStr.WriteString(strconv.Itoa(sqlParamIndex + j))
			}
			sliceValue := valuej.Interface()
			*newValues = append(*newValues, sliceValue)
		}
	}
	*sqlParamIndexPtr = *sqlParamIndexPtr + valueLen
}

// firstOneWord 从指定下标,获取第一个单词,不包含前后空格,并返回开始下标和结束下标,如果找不到合法的字符串,返回-1
func firstOneWord(index int, strByte *[]byte) (string, int, int, error) {
	start := -1
	end := -1
	byteLen := len(*strByte)
	if index < 0 {
		return "", start, end, errors.New("->firstOneWord索引小于0")
	}
	if index > byteLen { // 如果索引大于长度
		return "", start, end, errors.New("->firstOneWord索引大于字符串长度")
	}
	var newStr strings.Builder
	newStr.Grow(10)
	for ; index < byteLen; index++ {
		v := (*strByte)[index]
		if v == '(' || v == ')' { // 不处理括号
			continue
		}
		if start == -1 && v != ' ' { // 不是空格
			start = index
		}
		if start == -1 && v == ' ' { // 空格
			continue
		}
		if start >= 0 && v != ' ' { // 需要的字符
			newStr.WriteByte(v)
		} else { // 遇到空格结束记录
			end = index
			break
		}
	}
	if start >= 0 && end == -1 { // 记录到结尾,不是空格结束
		end = byteLen
	}

	return newStr.String(), start, end, nil
}
