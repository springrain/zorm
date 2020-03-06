package zorm

//IEntityStruct Entity实体类接口,所有实体类必须实现,否则baseDao无法执行.baseDao函数形参只有Finder和IBaseEntity
type IEntityStruct interface {
	//获取表名称
	GetTableName() string
	//获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称.
	GetPKColumnName() string
	//兼容主键序列.如果有值,优先级最高
	GetPkSequence() string
}

//IEntityMap Entity实体类接口,所有实体类必须实现,否则baseDao无法执行.baseDao函数形参只有Finder和IBaseEntity
type IEntityMap interface {
	//获取表名称
	GetTableName() string
	//获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称.
	GetPKColumnName() string
	//针对Map类型,记录数据库字段
	GetDBFieldMap() map[string]interface{}
}

//EntityStruct IBaseEntity 的基础实现,所有的实体类都匿名注入.这样就类似实现继承了,如果接口增加方法,调整这个默认实现即可
type EntityStruct struct {
}

//默认数据库的主键列名
const defaultPkName = "id"

//获取表名称
/*
func (entity *EntityStruct) GetTableName() string {
	return ""
}
*/

//GetPKColumnName 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称.
func (entity *EntityStruct) GetPKColumnName() string {
	return defaultPkName
}

//GetPkSequence Oracle和pgsql没有自增,主键使用序列.优先级高于GetPKColumnName方法
func (entity *EntityStruct) GetPkSequence() string {
	return ""
}

//-------------------------------------------------------------------------//

//EntityMap IBaseEntity的基础实现,所有的实体类都匿名注入.这样就类似实现继承了,如果接口增加方法,调整这个默认实现即可
type EntityMap struct {
	//表名
	tableName string
	//主键列名
	pkColumnName string
	//数据库字段,不暴露外部
	dbFieldMap map[string]interface{}
	//自定义的kv字段,不暴露外部
	transientMap map[string]interface{}
}

//NewEntityMap 初始化Map,必须传入表名称
func NewEntityMap(tbName string) *EntityMap {
	entityMap := EntityMap{}
	entityMap.dbFieldMap = map[string]interface{}{}
	entityMap.transientMap = map[string]interface{}{}
	entityMap.tableName = tbName
	entityMap.pkColumnName = defaultPkName
	return &entityMap
}

//GetTableName 获取表名称
func (entity *EntityMap) GetTableName() string {
	return entity.tableName
}

//SetPKColumnName 设置主键的名称
func (entity *EntityMap) SetPKColumnName(pkName string) {
	entity.pkColumnName = pkName
}

//GetPKColumnName 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称.
func (entity *EntityMap) GetPKColumnName() string {
	return entity.pkColumnName
}

//GetDBFieldMap 针对Map类型,记录数据库字段
func (entity *EntityMap) GetDBFieldMap() map[string]interface{} {
	return entity.dbFieldMap
}

//Set 设置数据库字段
func (entity *EntityMap) Set(key string, value interface{}) map[string]interface{} {
	entity.dbFieldMap[key] = value
	return entity.dbFieldMap
}

//Put 设置非数据库字段
func (entity *EntityMap) Put(key string, value interface{}) map[string]interface{} {
	entity.transientMap[key] = value
	return entity.transientMap
}
