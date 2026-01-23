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
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// tagColumnName tag标签的名称
const tagColumnName = "column"

// entityStructCacheMap 用于缓存entity和struct反射的信息,sync.Map内部处理了并发锁
var entityStructCacheMap = sync.Map{}

// fieldColumnCache 字段和数据库列的缓存
type fieldColumnCache struct {
	// columnName 数据库的原始列名,不带包裹符号
	columnName string
	// columnTag tag里的列名,可以用FuncWrapFieldTagName处理带方言的列名
	columnTag string
	// columnNameLower 列名(小写),避免重复调用 Name() 和 ToLower()
	columnNameLower string
	// structField 对应的结构体字段,可能为nil
	structField *reflect.StructField
	// fieldName 字段名,如果structField不为nil
	fieldName string
	// isPtr 字段是否为指针类型
	isPtr bool
	// isPK 是否是主键字段
	isPK bool

	// 以下属性仅用在查询时的映射上,每个查询都是单独的struct对象

	// columnType 数据库列类型
	columnType *sql.ColumnType
	// databaseTypeName 数据库类型名(大写),避免重复调用 DatabaseTypeName() 和 ToUpper()
	databaseTypeName string
	// dialectDatabaseTypeName 带方言前缀的数据库类型名,避免重复拼接字符串,例如 mysql.VARCHAR
	dialectDatabaseTypeName string
	// customDriverValueConver 自定义类型转换接口,缓存避免每行每列的map查找
	customDriverValueConver ICustomDriverValueConver
	// tempDriverValue 记录customDriverValueConver.GetDriverValue临时的值
	tempDriverValue driver.Value
}

// entityStructCache entity和struct结构体缓存,包含实体类的字段和数据库列的映射信息
type entityStructCache struct {
	// fields 所有的公开字段
	fields []*fieldColumnCache
	// fieldMap struct属性小写做key,value是字段缓存
	fieldMap map[string]*fieldColumnCache
	// columns field对应的数据库字段
	columns []*fieldColumnCache
	// columnMap 数据库字段小写做key
	columnMap map[string]*fieldColumnCache
	// insertSQL 插入的SQL语句
	insertSQL string
	// valuesSQL 值的SQL语句,例如 (?,?,?)
	valuesSQL string
	// deleteSQL 删除的SQL语句
	deleteSQL string
	// pkField 主键字段
	pkField *fieldColumnCache // 主键字段
	// pkType 主键类型: string,int,int64
	pkType string // 主键类型
	// pkSequence 主键序列名称
	pkSequence string // 主键序列名称
	// autoIncrement 自增类型  0(不自增),1(普通自增),2(序列自增)
	autoIncrement int
}

// getStructTypeOfCache 获取Struct实体类的结构体缓存,可以是普通的Struct
func getStructTypeOfCache(ctx context.Context, typeOfPtr *reflect.Type, config *DataSourceConfig) (*entityStructCache, error) {
	// pkgPath + _ + pkgName(因为单独用这个不保证唯一)
	//key := fmt.Sprintf("%s_%s_%s", config.Dialect, (*typeOf).PkgPath(), (*typeOf).String())
	typeOf := *typeOfPtr
	pkgPath := typeOf.PkgPath()
	typeOfString := typeOf.String()
	// 不同方言的缓存分开存储
	var keyBuilder strings.Builder
	keyBuilder.Grow(len(config.Dialect) + len(pkgPath) + len(typeOfString) + 3)
	keyBuilder.WriteString(config.Dialect)
	keyBuilder.WriteByte('_')
	keyBuilder.WriteString(pkgPath)
	keyBuilder.WriteByte('_')
	keyBuilder.WriteString(typeOfString)
	key := keyBuilder.String()

	// 缓存的值
	entityCacheLoad, cacheOK := entityStructCacheMap.Load(key)

	// 如果缓存存在值,不再处理
	if cacheOK {
		return entityCacheLoad.(*entityStructCache), nil
	}
	// 获取字段长度
	fieldNum := typeOf.NumField()
	// 如果没有字段
	if fieldNum < 1 {
		return nil, errors.New("->getEntityStructCache-->NumField entity没有属性")
	}
	// 创建实体类字段缓存
	entityCache := &entityStructCache{}
	entityCache.fields = make([]*fieldColumnCache, 0, fieldNum)
	entityCache.fieldMap = make(map[string]*fieldColumnCache)
	entityCache.columns = make([]*fieldColumnCache, 0, fieldNum)
	entityCache.columnMap = make(map[string]*fieldColumnCache)

	// 遍历所有字段,包括匿名属性
	for i := 0; i < fieldNum; i++ {
		field := typeOf.Field(i)
		if field.Anonymous { // 如果是匿名的,调用递归处理
			funcRecursiveAnonymous(ctx, entityCache, &field)
		} else if _, ok := entityCache.fieldMap[strings.ToLower(field.Name)]; !ok { // 普通命名字段,而且没有记录过
			// 创建entityStruct缓存
			funcCreateEntityStructCache(ctx, entityCache, field)
		}
	}

	// 记录到缓存中
	entityStructCacheMap.Store(key, entityCache)
	return entityCache, nil
}

// getEntityStructCache 获取Entity实体类的结构体缓存,必须是实现IEntityStruct接口
func getEntityStructCache(ctx context.Context, entity IEntityStruct, config *DataSourceConfig) (*entityStructCache, error) {
	if entity == nil {
		return nil, errors.New("->getEntityStructCache entity数据为空")
	}
	valueOf := reflect.ValueOf(entity)
	if valueOf.Kind() == reflect.Ptr { // 如果是指针
		valueOf = valueOf.Elem()
	} else {
		return nil, errors.New("->getEntityStructCache entity必须是指针类型")
	}
	// 反射类型
	typeOf := valueOf.Type()
	// 获取Struct类型的缓存
	entityCache, err := getStructTypeOfCache(ctx, &typeOf, config)
	if err != nil {
		return nil, err
	}
	if entityCache == nil {
		return nil, errors.New("->getEntityStructCache-->getStructTypeOfCache返回nil")
	}
	if len(entityCache.columns) < 1 {
		return nil, errors.New("->getEntityStructCache-->没有column信息,请检查struct中 column 的tag")
	}
	if entityCache.insertSQL != "" { // 已经处理过了
		return entityCache, nil
	}

	// @TODO 是否需要考虑并发?

	// 处理主键自增. 第一次插入手动插入(ID=0)会影响后续的autoIncrement判断,因为被缓存了,主键自增默认都是从1开始
	sequence := entity.GetPkSequence()
	if sequence != "" { // 序列自增 Sequence increment
		entityCache.pkSequence = sequence
		entityCache.autoIncrement = 2
	}
	pkColumnName := strings.ToLower(entity.GetPKColumnName())
	pkField, pkOK := entityCache.columnMap[pkColumnName]
	if pkOK {
		pkField.isPK = true
		entityCache.pkField = pkField
		// 获取主键类型 | Get the primary key type.
		pkKind := entityCache.pkField.structField.Type.Kind()
		pktype := ""
		switch pkKind {
		case reflect.String:
			pktype = "string"
		case reflect.Int, reflect.Int32, reflect.Int16, reflect.Int8:
			pktype = "int"
		case reflect.Int64:
			pktype = "int64"
		default:
			return nil, errors.New("->getEntityStructCache-->不支持的主键类型,只支持string,int,int64类型")
		}
		entityCache.pkType = pktype
		pkValueIsZero := valueOf.FieldByIndex(entityCache.pkField.structField.Index).IsZero()
		if pkValueIsZero && entityCache.autoIncrement != 2 && (pktype == "int" || pktype == "int64") { // 主键值是零值,并且不是序列自增
			entityCache.autoIncrement = 1 // 普通自增
		}
	}

	entityCache.insertSQL = "INSERT INTO " + entity.GetTableName()

	// insert SQL语句
	err = wrapInsertSQL(ctx, entityCache, config)
	if err != nil {
		return nil, err
	}
	// delete SQL语句
	entityCache.deleteSQL, err = wrapDeleteSQL(ctx, entity)
	if err != nil {
		return nil, err
	}

	return entityCache, nil
}

// insertEntityFieldValues 获取实体类的字段值数组
func insertEntityFieldValues(ctx context.Context, entity IEntityStruct, entityCache *entityStructCache, useDefaultValue bool, values *[]interface{}) error {
	// 获取实体类的反射,指针下的struct
	valueOf := reflect.ValueOf(entity).Elem()

	// 默认值的map,只对 Insert 和 InsertSlice 有效
	var defaultValueMap map[string]interface{} = nil
	if useDefaultValue {
		defaultValueMap = entity.GetDefaultValue()
	}
	// 遍历所有数据库字段名,小写的
	for _, column := range entityCache.columns {
		var value interface{}
		fv := valueOf.FieldByIndex(column.structField.Index)
		// 默认值
		isDefaultValue := false
		var defaultValue interface{}
		if defaultValueMap != nil {
			defaultValue, isDefaultValue = defaultValueMap[column.fieldName]
		}
		// 判断零值
		isZero := fv.IsZero()

		if column.isPK && isZero && entityCache.pkType == "string" { // 主键是字符串类型,并且值为"",赋值id
			// 生成主键字符串
			// Generate primary key string
			id := FuncGenerateStringID(ctx)
			value = id
			// 给对象主键赋值
			// Assign a value to the primary key of the object
			valueOf.FieldByIndex(entityCache.pkField.structField.Index).Set(reflect.ValueOf(id))
		} else if isDefaultValue && isZero { // 如果有默认值,并且fv是零值,等于默认值
			value = defaultValue
		} else if column.isPtr { // 如果是指针类型
			if !fv.IsNil() { // 如果不是nil值
				value = fv.Elem().Interface()
			} else {
				value = nil
			}
		} else {
			value = fv.Interface()
		}
		// 添加到记录值的数组
		*values = append(*values, value)
	}

	return nil
}

// updateEntityFieldValues 获取实体类的字段值数组
func updateEntityFieldValues(ctx context.Context, entity IEntityStruct, entityCache *entityStructCache, onlyUpdateNotZero bool) (*string, *[]interface{}, error) {
	// SQL语句的构造器
	// SQL statement constructor
	var updateSQLBuilder strings.Builder
	updateSQLBuilder.Grow(stringBuilderGrowLen)
	updateSQLBuilder.WriteString("UPDATE ")
	updateSQLBuilder.WriteString(entity.GetTableName())
	updateSQLBuilder.WriteString(" SET ")

	fLen := len(entityCache.columns)
	// 接收值的数组
	values := make([]interface{}, 0, fLen+1)

	// 获取实体类的反射,指针下的struct
	valueOf := reflect.ValueOf(entity).Elem()
	// Update仅更新指定列
	var onlyUpdateColsMap map[string]bool
	// UpdateNotZeroValue 必须更新指定列
	var mustUpdateColsMap map[string]bool

	if onlyUpdateNotZero { // 只更新非零值时,需要处理mustUpdateCols
		mustUpdateCols := ctx.Value(contextMustUpdateColsValueKey)
		if mustUpdateCols != nil { // 指定了仅更新的列
			mustUpdateColsMap = mustUpdateCols.(map[string]bool)
			if mustUpdateColsMap != nil {
				// 添加主键
				mustUpdateColsMap[entityCache.pkField.columnNameLower] = true
			}
		}
	} else { // update 更新全部字段时,需要处理onlyUpdateCols
		onlyUpdateCols := ctx.Value(contextOnlyUpdateColsValueKey)
		if onlyUpdateCols != nil { // 指定了仅更新的列
			onlyUpdateColsMap = onlyUpdateCols.(map[string]bool)
			if onlyUpdateColsMap != nil {
				// 添加主键
				onlyUpdateColsMap[entityCache.pkField.columnNameLower] = true
			}
		}
	}
	// 记录需要更新字段的索引,因为有些字段会跳过,所以不用 i
	j := 0
	// 遍历所有数据库字段名,小写的
	for _, column := range entityCache.columns {

		if column.isPK { // 主键不更新
			continue
		}

		// UpdateNotZeroValue,指定仅更新的列,当前列不用更新
		if onlyUpdateColsMap != nil && (!onlyUpdateColsMap[column.columnNameLower]) {
			continue
		}
		// 记录值
		var value interface{}
		// 反射获取字段的值
		fieldValue := valueOf.FieldByIndex(column.structField.Index)
		// 是否必须更新的字段
		isMustUpdate := false
		if mustUpdateColsMap != nil {
			isMustUpdate = mustUpdateColsMap[column.columnNameLower]
		}
		isZero := fieldValue.IsZero()
		if onlyUpdateNotZero && !isMustUpdate && isZero { // 如果只更新不为零值的,并且不是mustUpdateCols
			continue
		}
		if column.isPtr { // 如果是指针类型
			if fieldValue.IsNil() { // 如果是nil值
				value = nil
			} else {
				value = fieldValue.Elem().Interface()
			}
		} else { // 不是指针
			value = fieldValue.Interface()
		}
		// 添加 , 逗号
		if j > 0 {
			updateSQLBuilder.WriteByte(',')
		}
		j++
		updateSQLBuilder.WriteString(column.columnTag)
		updateSQLBuilder.WriteString("=?")
		// 添加到记录值的数组
		values = append(values, value)
	}
	// 添加组件参数
	updateSQLBuilder.WriteString(" WHERE ")
	updateSQLBuilder.WriteString(entityCache.pkField.columnName)
	updateSQLBuilder.WriteString("=?")
	// 添加主键值
	pkValue := valueOf.FieldByIndex(entityCache.pkField.structField.Index).Interface()
	values = append(values, pkValue)
	updateSQL := updateSQLBuilder.String()
	return &updateSQL, &values, nil
}

// sqlRowsValues 包装接收sqlRows的Values数组,反射rows屏蔽数据库null值,兼容单个字段查询和Struct映射
// 当读取数据库的值为NULL时,由于基本类型不支持为NULL,通过反射将未知driver.Value改为interface{},不再映射到struct实体类
// 感谢@fastabler提交的pr fix:converting NULL to int is unsupported
// oneColumnScanner 只有一个字段,而且可以直接Scan,例如string或者[]string,不需要反射StructType进行处理
func sqlRowsValues(ctx context.Context, valueOf *reflect.Value, typeOf *reflect.Type, rows *sql.Rows, driverValue *reflect.Value, fieldCaches []*fieldColumnCache, entity interface{}) error {
	if entity == nil && (valueOf == nil || valueOf.IsNil()) {
		return errors.New("->sqlRowsValues-->接收值的entity参数为nil")
	}

	var valueOfElem reflect.Value
	if entity == nil && valueOf != nil {
		valueOfElem = valueOf.Elem()
	}

	ctLen := len(fieldCaches)
	// 声明载体数组,用于存放struct的属性指针
	// Declare a carrier array to store the attribute pointer of the struct
	values := make([]interface{}, ctLen)
	// 记录需要类型转换的字段信息
	var tempDriverValues []*fieldColumnCache
	if iscdvm {
		tempDriverValues = make([]*fieldColumnCache, 0, ctLen)
	}
	// 循环字段
	for i, fieldCache := range fieldCaches {
		if fieldCache == nil { // 数据库字段比实体类的多,实体类无法接收,设置为默认值
			values[i] = new(interface{})
			continue
		}
		columnType := fieldCache.columnType
		dv := driverValue.Index(i)
		// if dv.IsValid() && dv.InterfaceData()[0] == 0 {
		if dv.IsValid() && dv.IsNil() { // 该字段的数据库值是null,取默认值 | The database value of this field is null, no further processing is required, use the default value
			values[i] = new(interface{})
			continue
		}
		if fieldCache.customDriverValueConver != nil { // 如果是需要转换的字段
			// 获取字段类型
			var structFieldType *reflect.Type
			if entity != nil { // 查询一个字段,并且可以直接接收
				structFieldType = typeOf
			} else if fieldCache.structField != nil { // struct类型,存在这个字段
				vtype := fieldCache.structField.Type
				structFieldType = &vtype
			}
			tempDriverValue, err := fieldCache.customDriverValueConver.GetDriverValue(ctx, columnType, structFieldType)
			if err != nil {
				return err
			}
			if tempDriverValue == nil {
				return errors.New("->sqlRowsValues-->customDriverValueConver.GetDriverValue返回的driver.Value不能为nil")
			}
			values[i] = tempDriverValue
			fieldCache.tempDriverValue = tempDriverValue
			tempDriverValues = append(tempDriverValues, fieldCache)
			continue
		}

		// 不需要customDriverValueConver特殊转换
		if entity != nil { // 查询一个字段,并且可以直接接收
			values[i] = entity
			continue
		}
		if fieldCache.structField == nil { // 如果不存在这个字段
			values[i] = new(interface{})
			continue
		}
		// 记录值
		var v interface{}
		// 字段的反射值
		fieldValue := valueOfElem.FieldByIndex(fieldCache.structField.Index)
		if fieldCache.isPtr { // 如果是指针类型
			// 反射new一个对应类型的指针
			newValue := reflect.New(fieldCache.structField.Type.Elem())
			// 反射赋值到字段值
			fieldValue.Set(newValue)
			// 获取字段值
			v = fieldValue.Interface()
		} else {
			v = fieldValue.Addr().Interface()
		}
		values[i] = v
	}
	// Scan赋值values数组,values长度必须和rows保持一致
	err := rows.Scan(values...)
	if err != nil {
		return err
	}
	// 没有特殊类型替换的值
	if len(tempDriverValues) < 1 {
		return nil
	}

	// 有特殊类型的替换值,循环需要替换的值
	for _, fieldCache := range tempDriverValues {
		// 根据列名,字段类型,新值 返回符合接收类型值的指针,返回值是个指针,指针,指针!!!!
		rightValue, errConverDriverValue := fieldCache.customDriverValueConver.ConverDriverValue(ctx, fieldCache.columnType, fieldCache.tempDriverValue, &fieldCache.structField.Type)
		if errConverDriverValue != nil {
			errConverDriverValue = fmt.Errorf("->sqlRowsValues-->customDriverValueConver.ConverDriverValue错误:%w", errConverDriverValue)
			FuncLogError(ctx, errConverDriverValue)
			return errConverDriverValue
		}

		if entity != nil { // 查询一个字段,并且可以直接接收
			reflect.ValueOf(entity).Elem().Set(reflect.ValueOf(rightValue).Elem())
			continue
		}
		// 如果是Struct类型接收
		if fieldCache.structField != nil {
			// 字段的反射值
			fieldValue := valueOfElem.FieldByIndex(fieldCache.structField.Index)
			// 给字段赋值
			fieldValue.Set(reflect.ValueOf(rightValue).Elem())
		}

	}

	return nil
}

// buildSelectFieldColumnCache 构建查询字段缓存
// buildSelectFieldColumnCache builds a cache of query fields
func buildSelectFieldColumnCache(columnTypes []*sql.ColumnType, entityCache *entityStructCache, dialect string) ([]*fieldColumnCache, error) {
	if columnTypes == nil {
		return nil, errors.New("->buildSelectFieldColumnCache-->columnTypes不能为nil")
	}

	fieldCaches := make([]*fieldColumnCache, len(columnTypes))
	// 创建columnType到cache的映射,用于O(1)查找
	//columnTypeToCache := make(map[*sql.ColumnType]*fieldColumnCache, len(columnTypes))

	for i, columnType := range columnTypes {
		//field, err := getStructFieldByColumnType(columnType, dbColumnFieldMap, exportFieldMap)
		columnName := strings.ToLower(columnType.Name())
		field, ok := entityCache.columnMap[columnName]
		if !ok {
			field, ok = entityCache.fieldMap[columnName] // 尝试用struct属性名查找
		}
		if !ok && strings.Contains(columnName, "_") { // 尝试驼峰命名转换(去除下划线)
			cname := strings.ReplaceAll(columnName, "_", "")
			field, ok = entityCache.fieldMap[cname]
		}
		// 数据库字段可能比Struct里多, fieldCache[i] = nil
		if field == nil {
			fieldCaches[i] = nil
			continue
		}

		databaseTypeName := strings.ToUpper(columnType.DatabaseTypeName())
		fieldCache := &fieldColumnCache{
			columnType:       columnType,
			structField:      field.structField,
			databaseTypeName: databaseTypeName,
			columnNameLower:  field.columnNameLower,
			isPtr:            field.isPtr,
			fieldName:        field.fieldName,

			// VARCHAR 和 TEXT 可以同时映射到一个string字段上,所以每次临时获取,不能缓存到field上
			//dialectDatabaseTypeName: field.dialectDatabaseTypeName,
			//customDriverValueConver: field.customDriverValueConver,
		}

		// 缓存带方言前缀的数据库类型名
		if dialect != "" {
			fieldCache.dialectDatabaseTypeName = dialect + "." + databaseTypeName
		}

		// 缓存customDriverValueConver,避免每行每列的map查找
		if iscdvm {
			if fieldCache.customDriverValueConver == nil {
				fieldCache.customDriverValueConver, _ = customDriverValueMap[fieldCache.dialectDatabaseTypeName]
			}
			if fieldCache.customDriverValueConver == nil {
				fieldCache.customDriverValueConver, _ = customDriverValueMap[fieldCache.databaseTypeName]
			}
		}

		fieldCaches[i] = fieldCache
		//columnTypeToCache[columnType] = cacheItem
	}

	return fieldCaches, nil
}

// buildEmptySelectFieldColumnCache 构建空的查询字段缓存(用于单字段查询)
// buildEmptySelectFieldColumnCache builds an empty query field cache (used for single field queries)
func buildEmptySelectFieldColumnCache(columnTypes []*sql.ColumnType, dialect string) []*fieldColumnCache {
	if columnTypes == nil {
		return nil
	}

	fieldCaches := make([]*fieldColumnCache, len(columnTypes))
	// 创建columnType到cache的映射,用于O(1)查找
	// columnTypeToCache := make(map[*sql.ColumnType]*fieldColumnCache, len(columnTypes))

	for i, columnType := range columnTypes {
		databaseTypeName := strings.ToUpper(columnType.DatabaseTypeName())
		fieldCache := &fieldColumnCache{
			columnType:       columnType,
			databaseTypeName: databaseTypeName,
		}
		// 缓存带方言前缀的数据库类型名
		if dialect != "" {
			fieldCache.dialectDatabaseTypeName = dialect + "." + databaseTypeName
		}
		// 缓存customDriverValueConver,避免每行每列的map查找
		if iscdvm {
			if fieldCache.customDriverValueConver == nil {
				fieldCache.customDriverValueConver, _ = customDriverValueMap[fieldCache.dialectDatabaseTypeName]
			}
			if fieldCache.customDriverValueConver == nil {
				fieldCache.customDriverValueConver, _ = customDriverValueMap[fieldCache.databaseTypeName]
			}
			//cacheItem.cdvcStatus = 1 // 已检查
		}
		fieldCaches[i] = fieldCache
		//columnTypeToCache[columnType] = cacheItem
	}

	return fieldCaches
}

// checkEntityKind 检查entity类型必须是*struct类型或者基础类型的指针
func checkEntityKind(entity interface{}) (*reflect.Type, error) {
	if entity == nil {
		return nil, errors.New("->checkEntityKind参数不能为空,必须是*struct类型或者基础类型的指针")
	}
	typeOf := reflect.TypeOf(entity)
	if typeOf.Kind() != reflect.Ptr { // 如果不是指针
		return nil, errors.New("->checkEntityKind必须是*struct类型或者基础类型的指针")
	}
	typeOf = typeOf.Elem()
	return &typeOf, nil
}

// funcCreateEntityStructCache 创建实体类字段缓存
func funcCreateEntityStructCache(ctx context.Context, entityCache *entityStructCache, field reflect.StructField) bool {
	fieldName := field.Name
	if !field.IsExported() { // 私有字段不处理
		return true
	}

	fieldCache := &fieldColumnCache{}
	fieldCache.structField = &field

	fieldCache.isPtr = field.Type.Kind() == reflect.Ptr
	fieldCache.fieldName = fieldName

	//记录FieldCache
	fieldNameLower := strings.ToLower(fieldName)
	entityCache.fieldMap[fieldNameLower] = fieldCache
	entityCache.fields = append(entityCache.fields, fieldCache)

	// 如果是数据库字段
	columnTag := field.Tag.Get(tagColumnName)
	if columnTag != "" {
		// dbColumnFieldMap[tagColumnValue] = field
		// 使用数据库字段的小写,处理oracle和达梦数据库的sql返回值大写
		columnName := columnTag
		//去掉可能的包裹符号
		columnName = strings.Trim(columnName, "`")
		columnName = strings.Trim(columnName, "\"")
		columnName = strings.TrimLeft(columnName, "[")
		columnName = strings.TrimRight(columnName, "]")
		fieldCache.columnName = columnName
		fieldCache.columnNameLower = strings.ToLower(columnName)

		fieldCache.columnTag = columnTag
		// @TODO 这里需要考虑已经在column tag中添加了包裹符,最好是把包裹符号放到Config中,取消FuncWrapFieldTagName函数
		if FuncWrapFieldTagName != nil {
			fieldCache.columnTag = FuncWrapFieldTagName(ctx, fieldCache.structField, columnTag)
		}

		entityCache.columns = append(entityCache.columns, fieldCache)
		entityCache.columnMap[fieldCache.columnNameLower] = fieldCache
	}

	return true
}

// funcRecursiveAnonymous 递归处理匿名struct字段
func funcRecursiveAnonymous(ctx context.Context, entityCache *entityStructCache, anonymous *reflect.StructField) {
	// 字段类型
	anonymousTypeOf := anonymous.Type
	if anonymousTypeOf.Kind() == reflect.Ptr {
		// 获取指针下的Struct类型
		anonymousTypeOf = anonymousTypeOf.Elem()
	}

	// 只处理Struct类型
	if anonymousTypeOf.Kind() != reflect.Struct {
		return
	}

	// 获取字段长度
	fieldNum := anonymousTypeOf.NumField()
	// 如果没有字段
	if fieldNum < 1 {
		return
	}
	// 遍历所有字段
	for i := 0; i < fieldNum; i++ {
		anonymousField := anonymousTypeOf.Field(i)
		if anonymousField.Anonymous { // 匿名struct里自身又有匿名struct,调用递归处理
			funcRecursiveAnonymous(ctx, entityCache, &anonymousField)
		} else if _, ok := entityCache.fieldMap[strings.ToLower(anonymousField.Name)]; !ok { // 普通命名字段,而且没有记录过
			// 创建entityStruct缓存
			funcCreateEntityStructCache(ctx, entityCache, anonymousField)
		}
	}
}
