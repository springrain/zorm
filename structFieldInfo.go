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
	"errors"
	"fmt"
	"go/ast"
	"reflect"
	"strings"
	"sync"
)

// tagColumnName tag标签的名称
const tagColumnName = "column"

// fieldColumnCache 查询字段缓存,包含每个字段的必要信息
type fieldColumnCache struct {
	// 数据库列类型
	columnType *sql.ColumnType
	columnName string
	columnTag  string // tag里的列名,可以用FuncWrapFieldTagName处理带方言的列名
	// 列名(小写),避免重复调用 Name() 和 ToLower()
	columnNameLower string
	// 对应的结构体字段,可能为nil
	structField *reflect.StructField
	// 字段是否为指针类型
	isPtr bool
	// 数据库类型名(大写),避免重复调用 DatabaseTypeName() 和 ToUpper()
	databaseTypeName string
	// 字段名,如果structField不为nil
	fieldName string
	// 带方言前缀的数据库类型名,避免重复拼接字符串
	dialectDatabaseTypeName string
}

// entityStructCache 实体类结构体缓存,包含实体类的字段和数据库列的映射信息
type entityStructCache struct {
	fields []*fieldColumnCache
	// struct属性小写做key
	fieldMap map[string]*fieldColumnCache
	columns  []*fieldColumnCache
	// 数据库字段小写做key
	columnMap map[string]*fieldColumnCache
	insertSQL string
	valuesSQL string
	//updateSQL     string
	deleteSQL     string
	pkField       *fieldColumnCache // 主键字段
	pkType        string            // 主键类型
	pkSequence    string            // 主键序列名称
	autoIncrement int               // 自增类型  0(不自增),1(普通自增),2(序列自增)
}

// cacheEntityStructMap 用于缓存反射的信息,sync.Map内部处理了并发锁
var cacheEntityStructMap *sync.Map = &sync.Map{}

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
	// if !(typeOf.Kind() == reflect.Struct || allowBaseTypeMap[typeOf.Kind()]) { //如果不是指针
	//	return nil, errors.New("checkEntityKind必须是*struct类型或者基础类型的指针")
	// }
	return &typeOf, nil
}

// sqlRowsValues 包装接收sqlRows的Values数组,反射rows屏蔽数据库null值,兼容单个字段查询和Struct映射
// 当读取数据库的值为NULL时,由于基本类型不支持为NULL,通过反射将未知driver.Value改为interface{},不再映射到struct实体类
// 感谢@fastabler提交的pr fix:converting NULL to int is unsupported
// oneColumnScanner 只有一个字段,而且可以直接Scan,例如string或者[]string,不需要反射StructType进行处理
func sqlRowsValues(ctx context.Context, valueOf *reflect.Value, typeOf *reflect.Type, rows *sql.Rows, driverValue *reflect.Value, fieldCache []*fieldColumnCache, columnTypeToCache map[*sql.ColumnType]*fieldColumnCache, entity interface{}) error {
	if entity == nil && (valueOf == nil || valueOf.IsNil()) {
		return errors.New("->sqlRowsValues-->接收值的entity参数为nil")
	}

	var valueOfElem reflect.Value
	if entity == nil && valueOf != nil {
		valueOfElem = valueOf.Elem()
	}

	ctLen := len(fieldCache)
	// 声明载体数组,用于存放struct的属性指针
	// Declare a carrier array to store the attribute pointer of the struct
	values := make([]interface{}, ctLen)
	// 记录需要类型转换的字段信息
	var fieldTempDriverValueMap map[*sql.ColumnType]*driverValueInfo
	if iscdvm {
		fieldTempDriverValueMap = make(map[*sql.ColumnType]*driverValueInfo)
	}
	var err error
	var customDriverValueConver ICustomDriverValueConver
	var converOK bool

	for i, cache := range fieldCache {
		columnType := cache.columnType
		if iscdvm {
			// 使用缓存的databaseTypeName,避免重复调用 DatabaseTypeName() 和 ToUpper()
			// 根据接收的类型,获取到类型转换的接口实现,优先匹配指定的数据库类型
			if cache.dialectDatabaseTypeName != "" {
				customDriverValueConver, converOK = customDriverValueMap[cache.dialectDatabaseTypeName]
			}
			if !converOK {
				customDriverValueConver, converOK = customDriverValueMap[cache.databaseTypeName]
			}
		}
		dv := driverValue.Index(i)
		// if dv.IsValid() && dv.InterfaceData()[0] == 0 {
		if dv.IsValid() && dv.IsNil() { // 该字段的数据库值是null,取默认值 | The database value of this field is null, no further processing is required, use the default value
			values[i] = new(interface{})
			continue
		} else if converOK { // 如果是需要转换的字段
			// 获取字段类型
			var structFieldType *reflect.Type
			if entity != nil { // 查询一个字段,并且可以直接接收
				structFieldType = typeOf
			} else { // 如果是struct类型
				if cache.structField != nil { // 存在这个字段
					vtype := cache.structField.Type
					structFieldType = &vtype
				}
			}
			tempDriverValue, err := customDriverValueConver.GetDriverValue(ctx, columnType, structFieldType)
			if err != nil {
				return err
			}
			if tempDriverValue == nil {
				return errors.New("->sqlRowsValues-->customDriverValueConver.GetDriverValue返回的driver.Value不能为nil")
			}
			values[i] = tempDriverValue

			// 如果需要类型转换
			dvinfo := driverValueInfo{}
			dvinfo.customDriverValueConver = customDriverValueConver
			// dvinfo.columnType = columnType
			dvinfo.structFieldType = structFieldType
			dvinfo.tempDriverValue = tempDriverValue
			fieldTempDriverValueMap[columnType] = &dvinfo
			continue

		} else if entity != nil { // 查询一个字段,并且可以直接接收
			values[i] = entity
			continue
		} else {
			if cache.structField == nil { // 如果不存在这个字段
				values[i] = new(interface{})
			} else {
				var v interface{}
				// 字段的反射值
				fieldValue := valueOfElem.FieldByIndex(cache.structField.Index)
				if cache.isPtr { // 如果是指针类型
					// 反射new一个对应类型的指针
					newValue := reflect.New(cache.structField.Type.Elem())
					// 反射赋值到字段值
					fieldValue.Set(newValue)
					// 获取字段值
					v = fieldValue.Interface()
				} else {
					v = fieldValue.Addr().Interface()
				}
				// v := new(interface{})
				values[i] = v
			}
		}

	}
	err = rows.Scan(values...)
	if err != nil {
		return err
	}
	if len(fieldTempDriverValueMap) < 1 {
		return err
	}

	// 循环需要替换的值
	for columnType, driverValueInfo := range fieldTempDriverValueMap {
		// 根据列名,字段类型,新值 返回符合接收类型值的指针,返回值是个指针,指针,指针!!!!
		// typeOf := fieldValue.Type()
		rightValue, errConverDriverValue := driverValueInfo.customDriverValueConver.ConverDriverValue(ctx, columnType, driverValueInfo.tempDriverValue, driverValueInfo.structFieldType)
		if errConverDriverValue != nil {
			errConverDriverValue = fmt.Errorf("->sqlRowsValues-->customDriverValueConver.ConverDriverValue错误:%w", errConverDriverValue)
			FuncLogError(ctx, errConverDriverValue)
			return errConverDriverValue
		}
		if entity != nil { // 查询一个字段,并且可以直接接收
			// entity = rightValue
			// valueOfElem.Set(reflect.ValueOf(rightValue).Elem())
			reflect.ValueOf(entity).Elem().Set(reflect.ValueOf(rightValue).Elem())
			continue
		} else { // 如果是Struct类型接收
			// 使用映射进行O(1)查找
			if cache, ok := columnTypeToCache[columnType]; ok && cache.structField != nil {
				// 字段的反射值
				fieldValue := valueOfElem.FieldByIndex(cache.structField.Index)
				// 给字段赋值
				fieldValue.Set(reflect.ValueOf(rightValue).Elem())
			}
		}

	}

	return err
}

// buildSelectFieldColumnCache 构建查询字段缓存
// buildSelectFieldColumnCache builds a cache of query fields
func buildSelectFieldColumnCache(columnTypes []*sql.ColumnType, entityCache *entityStructCache, dialect string) ([]*fieldColumnCache, map[*sql.ColumnType]*fieldColumnCache, error) {
	if columnTypes == nil {
		return nil, nil, errors.New("->buildSelectFieldColumnCache-->columnTypes不能为nil")
	}

	cache := make([]*fieldColumnCache, len(columnTypes))
	// 创建columnType到cache的映射,用于O(1)查找
	columnTypeToCache := make(map[*sql.ColumnType]*fieldColumnCache, len(columnTypes))

	for i, columnType := range columnTypes {
		//field, err := getStructFieldByColumnType(columnType, dbColumnFieldMap, exportFieldMap)
		columnName := strings.ToLower(columnType.Name())
		field, ok := entityCache.columnMap[columnName]
		if !ok {
			field, ok = entityCache.fieldMap[columnName] // 尝试用struct属性名查找
		}
		if !ok { // 尝试驼峰命名转换(去除下划线)
			cname := strings.ReplaceAll(columnName, "_", "")
			field, ok = entityCache.fieldMap[cname]
		}
		if field == nil {
			return nil, nil, errors.New("->buildSelectFieldColumnCache-->columnType.Name()找不到对应的字段")
		}

		databaseTypeName := strings.ToUpper(columnType.DatabaseTypeName())
		cacheItem := &fieldColumnCache{
			columnType:       columnType,
			structField:      field.structField,
			databaseTypeName: databaseTypeName,
			columnNameLower:  field.columnNameLower,
			isPtr:            field.isPtr,
			fieldName:        field.fieldName,
		}

		// 预计算带方言前缀的数据库类型名
		if dialect != "" {
			cacheItem.dialectDatabaseTypeName = dialect + "." + databaseTypeName
		}

		cache[i] = cacheItem
		columnTypeToCache[columnType] = cacheItem
	}

	return cache, columnTypeToCache, nil
}

// buildEmptySelectFieldColumnCache 构建空的查询字段缓存(用于单字段查询)
// buildEmptySelectFieldColumnCache builds an empty query field cache (used for single field queries)
func buildEmptySelectFieldColumnCache(columnTypes []*sql.ColumnType, dialect string) ([]*fieldColumnCache, map[*sql.ColumnType]*fieldColumnCache) {
	if columnTypes == nil {
		return nil, nil
	}

	cache := make([]*fieldColumnCache, len(columnTypes))
	// 创建columnType到cache的映射,用于O(1)查找
	columnTypeToCache := make(map[*sql.ColumnType]*fieldColumnCache, len(columnTypes))

	for i, columnType := range columnTypes {
		databaseTypeName := strings.ToUpper(columnType.DatabaseTypeName())
		cacheItem := &fieldColumnCache{
			columnType:       columnType,
			databaseTypeName: databaseTypeName,
		}
		// 预计算带方言前缀的数据库类型名
		if dialect != "" {
			cacheItem.dialectDatabaseTypeName = dialect + "." + databaseTypeName
		}
		cache[i] = cacheItem
		columnTypeToCache[columnType] = cacheItem
	}

	return cache, columnTypeToCache
}

// funcCreateEntityStructCache 创建实体类字段缓存
func funcCreateEntityStructCache(ctx context.Context, entityCache *entityStructCache, field reflect.StructField) bool {
	fieldName := field.Name
	if !ast.IsExported(fieldName) { // 私有字段不处理
		return true
	}
	fieldNameLower := strings.ToLower(fieldName)
	selectFieldCache := &fieldColumnCache{}
	selectFieldCache.structField = &field

	selectFieldCache.isPtr = field.Type.Kind() == reflect.Ptr
	selectFieldCache.fieldName = fieldName

	//记录FieldCache
	entityCache.fieldMap[fieldNameLower] = selectFieldCache
	entityCache.fields = append(entityCache.fields, selectFieldCache)

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
		selectFieldCache.columnName = columnName
		selectFieldCache.columnNameLower = strings.ToLower(columnName)

		selectFieldCache.columnTag = columnTag
		// @TODO 这里需要考虑已经在column tag中添加了包裹符,最好是把包裹符号放到Config中,取消FuncWrapFieldTagName函数
		if FuncWrapFieldTagName != nil {
			selectFieldCache.columnTag = FuncWrapFieldTagName(ctx, selectFieldCache.structField, columnTag)
		}

		entityCache.columns = append(entityCache.columns, selectFieldCache)
		entityCache.columnMap[selectFieldCache.columnNameLower] = selectFieldCache
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
		if anonymousField.Anonymous { // 匿名struct里自身又有匿名struct
			funcRecursiveAnonymous(ctx, entityCache, &anonymousField)
		} else if _, ok := entityCache.fieldMap[strings.ToLower(anonymousField.Name)]; !ok { // 普通命名字段,而且没有记录过
			funcCreateEntityStructCache(ctx, entityCache, anonymousField)
		}
	}
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
	typeOf := valueOf.Type()

	entityCache, err := getStructTypeOfCache(ctx, typeOf, config)
	if err != nil {
		return nil, fmt.Errorf("->getEntityStructCache-->getStructTypeOfCache错误:%w", err)
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
	// 处理主键自增
	sequence := entity.GetPkSequence()
	if sequence != "" { // 序列自增 Sequence increment
		entityCache.pkSequence = sequence
		entityCache.autoIncrement = 2
	}
	pkColumnName := strings.ToLower(entity.GetPKColumnName())
	pkFieldCache, pkOK := entityCache.columnMap[pkColumnName]
	if pkOK {
		entityCache.pkField = pkFieldCache
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
	//entityCache.updateSQL = "UPDATE " + entity.GetTableName() + " SET "

	// 包装insert SQL语句
	err = wrapInsertSQL(ctx, entityCache, config)
	if err != nil {
		return nil, err
	}
	entityCache.deleteSQL, err = wrapDeleteSQL(ctx, entity)
	if err != nil {
		return nil, err
	}

	return entityCache, nil
}

// getStructTypeOfCache 获取Struct实体类的结构体缓存,可以是普通的Struct
func getStructTypeOfCache(ctx context.Context, typeOf reflect.Type, config *DataSourceConfig) (*entityStructCache, error) {
	// pkgPath + _ + pkgName(因为单独用这个不保证唯一)
	//key := fmt.Sprintf("%s_%s_%s", config.Dialect, (*typeOf).PkgPath(), (*typeOf).String())
	// 不同方言的缓存分开存储
	var keyBuilder strings.Builder
	keyBuilder.Grow(len(config.Dialect) + len(typeOf.PkgPath()) + len(typeOf.String()) + 3)
	keyBuilder.WriteString(config.Dialect)
	keyBuilder.WriteByte('_')
	keyBuilder.WriteString(typeOf.PkgPath())
	keyBuilder.WriteByte('_')
	keyBuilder.WriteString(typeOf.String())
	key := keyBuilder.String()

	// 缓存的数据库主键值
	entityCacheLoad, cacheOK := cacheEntityStructMap.Load(key)

	// 如果存在值,认为缓存中有所有的信息,不再处理
	if cacheOK {
		return entityCacheLoad.(*entityStructCache), nil
	}
	// 获取字段长度
	fieldNum := typeOf.NumField()
	// 如果没有字段
	if fieldNum < 1 {
		return nil, errors.New("->getEntityStructCache-->NumField entity没有属性")
	}
	entityCache := &entityStructCache{}
	entityCache.fields = make([]*fieldColumnCache, 0, fieldNum)
	entityCache.fieldMap = make(map[string]*fieldColumnCache)
	entityCache.columns = make([]*fieldColumnCache, 0, fieldNum)
	entityCache.columnMap = make(map[string]*fieldColumnCache)

	// 遍历所有字段,记录匿名属性
	for i := 0; i < fieldNum; i++ {
		field := typeOf.Field(i)
		if field.Anonymous { // 如果是匿名的
			funcRecursiveAnonymous(ctx, entityCache, &field)
		} else if _, ok := entityCache.fieldMap[strings.ToLower(field.Name)]; !ok { // 普通命名字段,而且没有记录过
			funcCreateEntityStructCache(ctx, entityCache, field)
		}
	}

	// 记录到缓存中
	cacheEntityStructMap.Store(key, entityCache)
	return entityCache, nil
}

// insertEntityFieldValues 获取实体类的字段值数组
func insertEntityFieldValues(ctx context.Context, entity IEntityStruct, entityCache *entityStructCache, useDefaultValue bool, values *[]interface{}) error {
	// 获取实体类的反射,指针下的struct
	valueOf := reflect.ValueOf(entity).Elem()

	// 默认值的map,只对Insert Struct有效
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
		isZero := fv.IsZero()
		if column.columnNameLower == entityCache.pkField.columnNameLower && isZero && entityCache.pkType == "string" { // 主键是字符串类型,并且值为"",赋值id
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
				mustUpdateColsMap[strings.ToLower(entity.GetPKColumnName())] = true
			}
		}
	} else { // update 更新全部字段时,需要处理onlyUpdateCols
		onlyUpdateCols := ctx.Value(contextOnlyUpdateColsValueKey)
		if onlyUpdateCols != nil { // 指定了仅更新的列
			onlyUpdateColsMap = onlyUpdateCols.(map[string]bool)
			if onlyUpdateColsMap != nil {
				// 添加主键
				onlyUpdateColsMap[strings.ToLower(entity.GetPKColumnName())] = true
			}
		}
	}
	j := 0
	// 遍历所有数据库字段名,小写的
	for _, column := range entityCache.columns {

		if column.columnNameLower == entityCache.pkField.columnNameLower { // 主键不更新
			continue
		}

		// 指定仅更新的列,当前列不用更新
		if onlyUpdateColsMap != nil && (!onlyUpdateColsMap[column.columnNameLower]) {
			continue
		}

		var value interface{}
		fv := valueOf.FieldByIndex(column.structField.Index)
		// 必须更新的字段
		isMustUpdate := false
		if mustUpdateColsMap != nil {
			isMustUpdate = mustUpdateColsMap[column.columnNameLower]
		}
		isZero := fv.IsZero()
		if onlyUpdateNotZero && !isMustUpdate && isZero { // 如果只更新不为零值的,并且不是mustUpdateCols
			continue
			// 重点说明:仅用于Insert和InsertSlice Struct,对Update和UpdateNotZeroValue无效
		} else if column.isPtr { // 如果是指针类型
			if !fv.IsNil() { // 如果不是nil值
				value = fv.Elem().Interface()
			} else {
				value = nil
			}
		} else {
			value = fv.Interface()
		}
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
