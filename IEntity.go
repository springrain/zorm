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

// IEntityStruct "struct"实体类的接口,所有的struct实体类都要实现这个接口
// IEntityStruct The interface of the "struct" entity class, all struct entity classes must implement this interface
type IEntityStruct interface {
	// 获取表名称
	// Get the table name.
	GetTableName() string

	// 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称
	// Get the primary key field name of the database table. Because it is compatible with Map, it can only be the field name of the database
	GetPKColumnName() string

	// GetPkSequence 主键序列
	// GetPkSequence Primary key sequence
	GetPkSequence() string
}

// IEntityMap 使用Map保存数据,用于不方便使用struct的场景,如果主键是自增或者序列,不要"entityMap.Set"主键的值
// IEntityMap Use Map to save data for scenarios where it is not convenient to use struct
// If the primary key is auto-increment or sequence, do not "entity Map.Set" the value of the primary key
type IEntityMap interface {
	// 获取表名称
	// Get the table name
	GetTableName() string

	// 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称.
	// Get the primary key field name of the database table. Because it is compatible with Map, it can only be the field name of the database.
	GetPKColumnName() string

	// GetEntityMapPkSequence 主键序列,不能使用GetPkSequence方法名,避免默认实现了IEntityStruct接口
	// GetEntityMapPkSequence primary key sequence, you cannot use the GetPkSequence method name, to avoid the default implementation of IEntityStruct interface
	GetEntityMapPkSequence() string

	// GetDBFieldMap 针对Map类型,记录数据库字段
	// GetDBFieldMap For Map type, record database fields.
	GetDBFieldMap() map[string]interface{}

	// GetDBFieldMapKey 按照Set的先后顺序记录key值,也就是数据库字段,用于SQL排序
	// GetDBFieldMapKey records the key value, that is, the database field, in the order of the Set, which is used for SQL sorting
	GetDBFieldMapKey() []string
	// 设置数据库字段的值
	// Set the value of a database field.
	Set(key string, value interface{}) map[string]interface{}
}

// EntityStruct "IBaseEntity" 的基础实现,所有的实体类都匿名注入.这样就类似实现继承了,如果接口增加方法,调整这个默认实现即可
// EntityStruct The basic implementation of "IBaseEntity", all entity classes are injected anonymously
// This is similar to implementation inheritance. If the interface adds methods, adjust the default implementation
type EntityStruct struct{}

// 默认数据库的主键列名
// Primary key column name of the default database
const defaultPkName = "id"

//GetTableName 获取表名称,必须有具体的Struct实现,类似java的抽象方法,避免手误忘记写表名.如果有扩展需求,建议使用接口进行扩展,不要默认实现GetTableName
/*
func (entity *EntityStruct) GetTableName() string {
	return ""
}
*/

// GetPKColumnName 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称
// GetPKColumnName Get the primary key field name of the database table
// Because it is compatible with Map, it can only be the field name of the database
func (entity *EntityStruct) GetPKColumnName() string {
	return defaultPkName
}

// var defaultPkSequence = make(map[string]string, 0)

// GetPkSequence 主键序列
// GetPkSequence Primary key sequence
func (entity *EntityStruct) GetPkSequence() string {
	return ""
}

//-------------------------------------------------------------------------//

// EntityMap IEntityMap的基础实现,可以直接使用或者匿名注入
type EntityMap struct {
	// 表名
	tableName string
	// 主键列名
	PkColumnName string
	// 主键序列,如果有值,优先级最高
	PkSequence string
	// 数据库字段,不暴露外部
	dbFieldMap map[string]interface{}
	// 列名,记录顺序
	dbFieldMapKey []string
}

// NewEntityMap 初始化Map,必须传入表名称
func NewEntityMap(tbName string) *EntityMap {
	entityMap := EntityMap{}
	entityMap.dbFieldMap = map[string]interface{}{}
	entityMap.tableName = tbName
	entityMap.PkColumnName = defaultPkName
	entityMap.dbFieldMapKey = make([]string, 0)
	return &entityMap
}

// GetTableName 获取表名称
func (entity *EntityMap) GetTableName() string {
	return entity.tableName
}

// GetPKColumnName 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称
func (entity *EntityMap) GetPKColumnName() string {
	return entity.PkColumnName
}

// GetEntityMapPkSequence 主键序列,不能使用GetPkSequence方法名,避免默认实现了IEntityStruct接口
// GetEntityMapPkSequence primary key sequence, you cannot use the GetPkSequence method name, to avoid the default implementation of IEntityStruct interface
func (entity *EntityMap) GetEntityMapPkSequence() string {
	return entity.PkSequence
}

// GetDBFieldMap 针对Map类型,记录数据库字段
// GetDBFieldMap For Map type, record database fields
func (entity *EntityMap) GetDBFieldMap() map[string]interface{} {
	return entity.dbFieldMap
}

// GetDBFieldMapKey 按照Set的先后顺序记录key值,也就是数据库字段,用于SQL排序
// GetDBFieldMapKey records the key value, that is, the database field, in the order of the Set, which is used for SQL sorting
func (entity *EntityMap) GetDBFieldMapKey() []string {
	return entity.dbFieldMapKey
}

// Set 设置数据库字段
// Set Set database fields
func (entity *EntityMap) Set(key string, value interface{}) map[string]interface{} {
	_, ok := entity.dbFieldMap[key]
	if !ok { // 如果不存在
		entity.dbFieldMapKey = append(entity.dbFieldMapKey, key)
	}
	entity.dbFieldMap[key] = value

	return entity.dbFieldMap
}
