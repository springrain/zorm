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

const (
	// tag标签的名称
	tagColumnName = "column"

	// 输出字段 缓存的前缀
	exportPrefix = "_exportStructFields_"
	// 私有字段 缓存的前缀
	privatePrefix = "_privateStructFields_"
	// 数据库列名 缓存的前缀
	dbColumnNamePrefix = "_dbColumnName_"

	// 数据库所有列名,经过排序 缓存的前缀
	dbColumnNameSlicePrefix = "_dbColumnNameSlice_"

	// field对应的column的tag值 缓存的前缀
	// structFieldTagPrefix = "_structFieldTag_"
	// 数据库主键  缓存的前缀
	// dbPKNamePrefix = "_dbPKName_"
)

// cacheStructFieldInfoMap 用于缓存反射的信息,sync.Map内部处理了并发锁
var cacheStructFieldInfoMap *sync.Map = &sync.Map{}

// var cacheStructFieldInfoMap = make(map[string]map[string]reflect.StructField)

// 用于缓存field对应的column的tag值
// var cacheStructFieldTagInfoMap = make(map[string]map[string]string)

// structFieldInfo 获取StructField的信息.只对struct或者*struct判断,如果是指针,返回指针下实际的struct类型
// 第一个返回值是可以输出的字段(首字母大写),第二个是不能输出的字段(首字母小写)
func structFieldInfo(typeOf *reflect.Type) error {
	if typeOf == nil {
		return errors.New("->structFieldInfo数据为空")
	}

	entityName := (*typeOf).String()

	// 缓存的key
	// 所有输出的属性,包含数据库字段,key是struct属性的名称,不区分大小写
	exportCacheKey := exportPrefix + entityName
	// 所有私有变量的属性,key是struct属性的名称,不区分大小写
	privateCacheKey := privatePrefix + entityName
	// 所有数据库的属性,key是数据库的字段名称,不区分大小写
	dbColumnCacheKey := dbColumnNamePrefix + entityName
	// 所有数据库字段名称的slice,经过排序,不区分大小写
	dbColumnNameSliceCacheKey := dbColumnNameSlicePrefix + entityName

	// structFieldTagCacheKey := structFieldTagPrefix + entityName
	// dbPKNameCacheKey := dbPKNamePrefix + entityName
	// 缓存的数据库主键值
	_, exportOk := cacheStructFieldInfoMap.Load(exportCacheKey)
	//_, exportOk := cacheStructFieldInfoMap[exportCacheKey]
	//如果存在值,认为缓存中有所有的信息,不再处理
	if exportOk {
		return nil
	}
	// 获取字段长度
	fieldNum := (*typeOf).NumField()
	// 如果没有字段
	if fieldNum < 1 {
		return errors.New("->structFieldInfo-->NumField entity没有属性")
	}

	// 声明所有字段的载体
	var allFieldMap *sync.Map = &sync.Map{}
	// anonymous := make([]reflect.StructField, 0)

	// 缓存的数据
	exportStructFieldMap := make(map[string]reflect.StructField)
	privateStructFieldMap := make(map[string]reflect.StructField)
	dbColumnFieldMap := make(map[string]reflect.StructField)

	// structFieldTagMap := make(map[string]string)
	dbColumnFieldNameSlice := make([]string, 0)

	// 遍历sync.Map,要求输入一个func作为参数
	// 这个函数的入参、出参的类型都已经固定,不能修改
	// 可以在函数体内编写自己的代码,调用map中的k,v
	// var funcMapKV func(k, v interface{}) bool
	funcMapKV := func(k, v interface{}) bool {
		field := v.(reflect.StructField)
		fieldName := field.Name
		if ast.IsExported(fieldName) { // 如果是可以输出的,不区分大小写
			exportStructFieldMap[strings.ToLower(fieldName)] = field
			// 如果是数据库字段
			tagColumnValue := field.Tag.Get(tagColumnName)
			if len(tagColumnValue) > 0 {
				// dbColumnFieldMap[tagColumnValue] = field
				// 使用数据库字段的小写,处理oracle和达梦数据库的sql返回值大写
				tagColumnValueLower := strings.ToLower(tagColumnValue)
				dbColumnFieldMap[tagColumnValueLower] = field
				dbColumnFieldNameSlice = append(dbColumnFieldNameSlice, tagColumnValueLower)
				// structFieldTagMap[fieldName] = tagColumnValue
			}

		} else { // 私有属性
			privateStructFieldMap[strings.ToLower(fieldName)] = field
		}

		return true
	}
	// 并发锁,用于处理slice并发append
	var lock sync.Mutex
	// funcRecursiveAnonymous 递归调用struct的匿名属性,就近覆盖属性
	var funcRecursiveAnonymous func(allFieldMap *sync.Map, anonymous *reflect.StructField)
	funcRecursiveAnonymous = func(allFieldMap *sync.Map, anonymous *reflect.StructField) {
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
				funcRecursiveAnonymous(allFieldMap, &anonymousField)
			} else if _, ok := allFieldMap.Load(anonymousField.Name); !ok { // 普通命名字段,而且没有记录过
				allFieldMap.Store(anonymousField.Name, anonymousField)
				lock.Lock()
				funcMapKV(anonymousField.Name, anonymousField)
				lock.Unlock()
			}
		}
	}

	// 遍历所有字段,记录匿名属性
	for i := 0; i < fieldNum; i++ {
		field := (*typeOf).Field(i)
		if field.Anonymous { // 如果是匿名的
			funcRecursiveAnonymous(allFieldMap, &field)
		} else if _, ok := allFieldMap.Load(field.Name); !ok { // 普通命名字段,而且没有记录过
			allFieldMap.Store(field.Name, field)
			lock.Lock()
			funcMapKV(field.Name, field)
			lock.Unlock()
		}
	}

	// allFieldMap.Range(f)

	// 加入缓存
	cacheStructFieldInfoMap.Store(exportCacheKey, exportStructFieldMap)
	cacheStructFieldInfoMap.Store(privateCacheKey, privateStructFieldMap)
	cacheStructFieldInfoMap.Store(dbColumnCacheKey, dbColumnFieldMap)
	// cacheStructFieldInfoMap[exportCacheKey] = exportStructFieldMap
	// cacheStructFieldInfoMap[privateCacheKey] = privateStructFieldMap
	// cacheStructFieldInfoMap[dbColumnCacheKey] = dbColumnFieldMap

	// cacheStructFieldTagInfoMap[structFieldTagCacheKey] = structFieldTagMap

	// 不按照字母顺序,按照反射获取的Struct属性顺序,生成insert语句和update语句
	// sort.Strings(dbColumnFieldNameSlice)
	cacheStructFieldInfoMap.Store(dbColumnNameSliceCacheKey, dbColumnFieldNameSlice)

	return nil
}

// setFieldValueByColumnName 根据数据库的字段名,找到struct映射的字段,并赋值
func setFieldValueByColumnName(entity interface{}, columnName string, value interface{}) error {
	// 先从本地缓存中查找
	typeOf := reflect.TypeOf(entity)
	valueOf := reflect.ValueOf(entity)
	if typeOf.Kind() == reflect.Ptr { // 如果是指针
		typeOf = typeOf.Elem()
		valueOf = valueOf.Elem()
	}

	dbMap, err := getDBColumnFieldMap(&typeOf)
	if err != nil {
		return err
	}
	f, ok := dbMap[strings.ToLower(columnName)]
	if ok { // 给主键赋值
		valueOf.FieldByName(f.Name).Set(reflect.ValueOf(value))
	}
	return nil
}

// structFieldValue 获取指定字段的值
func structFieldValue(s interface{}, fieldName string) (interface{}, error) {
	if s == nil || len(fieldName) < 1 {
		return nil, errors.New("->structFieldValue数据为空")
	}
	// entity的s类型
	valueOf := reflect.ValueOf(s)

	kind := valueOf.Kind()
	if !(kind == reflect.Ptr || kind == reflect.Struct) {
		return nil, errors.New("->structFieldValue必须是Struct或者*Struct类型")
	}

	if kind == reflect.Ptr {
		// 获取指针下的Struct类型
		valueOf = valueOf.Elem()
		if valueOf.Kind() != reflect.Struct {
			return nil, errors.New("->structFieldValue必须是Struct或者*Struct类型")
		}
	}

	// FieldByName方法返回的是reflect.Value类型,调用Interface()方法,返回原始类型的数据值
	value := valueOf.FieldByName(fieldName).Interface()

	return value, nil
}

// getDBColumnExportFieldMap 获取实体类的数据库字段,key是数据库的字段名称.同时返回所有的字段属性的map,key是实体类的属性.不区分大小写
func getDBColumnExportFieldMap(typeOf *reflect.Type) (map[string]reflect.StructField, map[string]reflect.StructField, error) {
	dbColumnFieldMap, err := getCacheStructFieldInfoMap(typeOf, dbColumnNamePrefix)
	if err != nil {
		return nil, nil, err
	}
	exportFieldMap, err := getCacheStructFieldInfoMap(typeOf, exportPrefix)
	return dbColumnFieldMap, exportFieldMap, err
}

// getDBColumnFieldMap 获取实体类的数据库字段,key是数据库的字段名称.不区分大小写
func getDBColumnFieldMap(typeOf *reflect.Type) (map[string]reflect.StructField, error) {
	return getCacheStructFieldInfoMap(typeOf, dbColumnNamePrefix)
}

// getDBColumnFieldNameSlice 获取实体类的数据库字段,经过排序,key是数据库的字段名称.不区分大小写,
func getDBColumnFieldNameSlice(typeOf *reflect.Type) ([]string, error) {
	dbColumnFieldSlice, dbmapErr := getCacheStructFieldInfo(typeOf, dbColumnNameSlicePrefix)
	if dbmapErr != nil {
		return nil, fmt.Errorf("->getDBColumnFieldNameSlice-->getCacheStructFieldInfo()取值错误:%w", dbmapErr)
	}
	dbcfSlice, efOK := dbColumnFieldSlice.([]string)
	if !efOK {
		return dbcfSlice, errors.New("->getDBColumnFieldNameSlice-->dbColumnFieldSlice取值转[]string类型异常")
	}
	return dbcfSlice, nil
}

// getCacheStructFieldInfo 根据类型和key,获取缓存的数据字段信息slice,已经排序
func getCacheStructFieldInfo(typeOf *reflect.Type, keyPrefix string) (interface{}, error) {
	if typeOf == nil {
		return nil, errors.New("->getCacheStructFieldInfo-->typeOf不能为空")
	}
	key := keyPrefix + (*typeOf).String()
	dbColumnFieldMap, dbOk := cacheStructFieldInfoMap.Load(key)
	// dbColumnFieldMap, dbOk := cacheStructFieldInfoMap[key]
	if !dbOk { // 缓存不存在
		// 获取实体类的输出字段和私有 字段
		err := structFieldInfo(typeOf)
		if err != nil {
			return nil, err
		}
		dbColumnFieldMap, dbOk = cacheStructFieldInfoMap.Load(key)
		// dbColumnFieldMap, dbOk = cacheStructFieldInfoMap[key]
		if !dbOk {
			return nil, errors.New("->getCacheStructFieldInfo-->cacheStructFieldInfoMap.Load()获取数据库字段dbColumnFieldMap异常")
		}
	}

	return dbColumnFieldMap, nil

	// return dbColumnFieldMap, nil
}

// getCacheStructFieldInfoMap 根据类型和key,获取缓存的字段信息
func getCacheStructFieldInfoMap(typeOf *reflect.Type, keyPrefix string) (map[string]reflect.StructField, error) {
	dbColumnFieldMap, dbmapErr := getCacheStructFieldInfo(typeOf, keyPrefix)
	if dbmapErr != nil {
		return nil, fmt.Errorf("->getCacheStructFieldInfoMap-->getCacheStructFieldInfo()取值错误:%w", dbmapErr)
	}
	dbcfMap, efOK := dbColumnFieldMap.(map[string]reflect.StructField)
	if !efOK {
		return dbcfMap, errors.New("->getCacheStructFieldInfoMap-->dbColumnFieldMap取值转map[string]reflect.StructField类型异常")
	}
	return dbcfMap, nil

	// return dbColumnFieldMap, nil
}

// columnAndValue 根据保存的对象,返回插入的语句,需要插入的字段,字段的值
func columnAndValue(entity interface{}) (reflect.Type, []reflect.StructField, []interface{}, error) {
	typeOf, checkerr := checkEntityKind(entity)
	if checkerr != nil {
		return typeOf, nil, nil, checkerr
	}
	// 获取实体类的反射,指针下的struct
	valueOf := reflect.ValueOf(entity).Elem()
	// reflect.Indirect

	// 先从本地缓存中查找
	// typeOf := reflect.TypeOf(entity).Elem()

	dbMap, err := getDBColumnFieldMap(&typeOf)
	if err != nil {
		return typeOf, nil, nil, err
	}
	dbSlice, err := getDBColumnFieldNameSlice(&typeOf)
	if err != nil {
		return typeOf, nil, nil, err
	}
	// 实体类公开字段的长度
	fLen := len(dbMap)
	// 长度不一致
	if fLen-len(dbSlice) != 0 {
		return typeOf, nil, nil, errors.New("->columnAndValue-->缓存的数据库字段和实体类字段不对应")
	}
	// 接收列的数组,这里是做一个副本,避免外部更改掉原始的列信息
	columns := make([]reflect.StructField, 0, fLen)
	// 接收值的数组
	values := make([]interface{}, 0, fLen)

	// 遍历所有数据库属性
	for _, fieldName := range dbSlice {
		//获取字段类型的Kind
		//	fieldKind := field.Type.Kind()
		//if !allowTypeMap[fieldKind] { //不允许的类型
		//	continue
		//}
		field := dbMap[fieldName]
		columns = append(columns, field)
		// FieldByName方法返回的是reflect.Value类型,调用Interface()方法,返回原始类型的数据值.字段不会重名,不使用FieldByIndex()函数
		value := valueOf.FieldByName(field.Name).Interface()
		// 添加到记录值的数组
		values = append(values, value)

	}

	// 缓存数据库的列
	return typeOf, columns, values, nil
}

// entityPKFieldName 获取实体类主键属性名称
func entityPKFieldName(entity IEntityStruct, typeOf *reflect.Type) (string, error) {
	//检查是否是指针对象
	//typeOf, checkerr := checkEntityKind(entity)
	//if checkerr != nil {
	//	return "", checkerr
	//}

	// 缓存的key,TypeOf和ValueOf的String()方法,返回值不一样
	// typeOf := reflect.TypeOf(entity).Elem()

	dbMap, err := getDBColumnFieldMap(typeOf)
	if err != nil {
		return "", err
	}
	field := dbMap[strings.ToLower(entity.GetPKColumnName())]
	return field.Name, nil
}

// checkEntityKind 检查entity类型必须是*struct类型或者基础类型的指针
func checkEntityKind(entity interface{}) (reflect.Type, error) {
	if entity == nil {
		return nil, errors.New("->checkEntityKind参数不能为空,必须是*struct类型或者基础类型的指针")
	}
	typeOf := reflect.TypeOf(entity)
	if typeOf.Kind() != reflect.Ptr { // 如果不是指针
		return nil, errors.New("->checkEntityKind必须是*struct类型或者基础类型的指针")
	}
	typeOf = typeOf.Elem()
	//if !(typeOf.Kind() == reflect.Struct || allowBaseTypeMap[typeOf.Kind()]) { //如果不是指针
	//	return nil, errors.New("checkEntityKind必须是*struct类型或者基础类型的指针")
	//}
	return typeOf, nil
}

// sqlRowsValues 包装接收sqlRows的Values数组,反射rows屏蔽数据库null值,兼容单个字段查询和Struct映射
// fix:converting NULL to int is unsupported
// 当读取数据库的值为NULL时,由于基本类型不支持为NULL,通过反射将未知driver.Value改为interface{},不再映射到struct实体类
// 感谢@fastabler提交的pr
// oneColumnScanner 只有一个字段,而且可以直接Scan,例如string或者[]string,不需要反射StructType进行处理
func sqlRowsValues(ctx context.Context, dialect string, valueOf *reflect.Value, typeOf *reflect.Type, rows *sql.Rows, driverValue *reflect.Value, columnTypes []*sql.ColumnType, entity interface{}, dbColumnFieldMap, exportFieldMap *map[string]reflect.StructField) error {
	if entity == nil && valueOf == nil {
		return errors.New("->sqlRowsValues-->valueOfElem为nil")
	}

	var valueOfElem reflect.Value
	if entity == nil && valueOf != nil {
		valueOfElem = valueOf.Elem()
	}

	ctLen := len(columnTypes)
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

	for i, columnType := range columnTypes {
		if iscdvm {
			databaseTypeName := strings.ToUpper(columnType.DatabaseTypeName())
			// 根据接收的类型,获取到类型转换的接口实现,优先匹配指定的数据库类型
			customDriverValueConver, converOK = customDriverValueMap[dialect+"."+databaseTypeName]
			if !converOK {
				customDriverValueConver, converOK = customDriverValueMap[databaseTypeName]
			}
		}
		dv := driverValue.Index(i)
		if dv.IsValid() && dv.InterfaceData()[0] == 0 { // 该字段的数据库值是null,取默认值
			values[i] = new(interface{})
			continue
		} else if converOK { // 如果是需要转换的字段
			// 获取字段类型
			var structFieldType *reflect.Type
			if entity != nil { // 查询一个字段,并且可以直接接收
				structFieldType = typeOf
			} else { // 如果是struct类型
				field, err := getStructFieldByColumnType(columnType, dbColumnFieldMap, exportFieldMap)
				if err != nil {
					return err
				}
				if field != nil { // 存在这个字段
					vtype := field.Type
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
			field, err := getStructFieldByColumnType(columnType, dbColumnFieldMap, exportFieldMap)
			if err != nil {
				return err
			}
			if field == nil { // 如果不存在这个字段
				values[i] = new(interface{})
			} else {
				// fieldType := refPV.FieldByName(field.Name).Type()
				// v := reflect.New(field.Type).Interface()
				// 字段的反射值
				fieldValue := valueOfElem.FieldByName(field.Name)
				v := fieldValue.Addr().Interface()
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
			field, err := getStructFieldByColumnType(columnType, dbColumnFieldMap, exportFieldMap)
			if err != nil {
				return err
			}
			if field != nil { // 如果存在这个字段
				// 字段的反射值
				fieldValue := valueOfElem.FieldByName(field.Name)
				// 给字段赋值
				fieldValue.Set(reflect.ValueOf(rightValue).Elem())
			}
		}

	}

	return err
}

// getStructFieldByColumnType 根据ColumnType获取StructField对象,兼容驼峰
func getStructFieldByColumnType(columnType *sql.ColumnType, dbColumnFieldMap *map[string]reflect.StructField, exportFieldMap *map[string]reflect.StructField) (*reflect.StructField, error) {
	columnName := strings.ToLower(columnType.Name())
	// columnName := "test"
	// 从缓存中获取列名的field字段
	// Get the field field of the column name from the cache
	field, fok := (*dbColumnFieldMap)[columnName]
	if !fok {
		field, fok = (*exportFieldMap)[columnName]
		if !fok {
			// 尝试驼峰
			cname := strings.ReplaceAll(columnName, "_", "")
			field, fok = (*exportFieldMap)[cname]

		}

	}
	if fok {
		return &field, nil
	}
	return nil, nil
}
