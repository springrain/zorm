package zorm

import (
	"bytes"
	"encoding/gob"
	"errors"
	"go/ast"
	"reflect"
	"sync"
)

const (
	//tag标签的名称
	tagColumnName = "column"

	//输出字段 缓存的前缀
	exportPrefix = "_exportStructFields_"
	//私有字段 缓存的前缀
	privatePrefix = "_privateStructFields_"
	//数据库列名 缓存的前缀
	dbColumnNamePrefix = "_dbColumnName_"
	//数据库主键  缓存的前缀
	dbPKNamePrefix = "_dbPKName_"
)

// 用于缓存反射的信息,sync.Map内部处理了并发锁
var cacheStructFieldInfoMap *sync.Map = &sync.Map{}

//获取StructField的信息.只对struct或者*struct判断,如果是指针,返回指针下实际的struct类型.
//第一个返回值是可以输出的字段(首字母大写),第二个是不能输出的字段(首字母小写)
func structFieldInfo(typeOf reflect.Type) error {

	if typeOf == nil {
		return errors.New("数据为空")
	}

	entityName := typeOf.String()

	//缓存的key
	exportCacheKey := exportPrefix + entityName
	privateCacheKey := privatePrefix + entityName
	dbColumnCacheKey := dbColumnNamePrefix + entityName
	//dbPKNameCacheKey := dbPKNamePrefix + entityName
	//缓存的数据库主键值
	_, exportOk := cacheStructFieldInfoMap.Load(exportCacheKey)
	//如果存在值,认为缓存中有所有的信息,不再处理
	if exportOk {
		return nil
	}
	//获取字段长度
	fieldNum := typeOf.NumField()
	//如果没有字段
	if fieldNum < 1 {
		return errors.New("entity没有属性")
	}

	// 声明所有字段的载体
	var allFieldMap *sync.Map = &sync.Map{}
	anonymous := make([]reflect.StructField, 0)

	//遍历所有字段,记录匿名属性
	for i := 0; i < fieldNum; i++ {
		field := typeOf.Field(i)
		if _, ok := allFieldMap.Load(field.Name); !ok {
			allFieldMap.Store(field.Name, field)
		}
		if field.Anonymous { //如果是匿名的
			anonymous = append(anonymous, field)
		}
	}
	//调用匿名struct的递归方法
	recursiveAnonymousStruct(allFieldMap, anonymous)

	//缓存的数据
	exportStructFieldMap := make(map[string]reflect.StructField)
	privateStructFieldMap := make(map[string]reflect.StructField)
	dbColumnFieldMap := make(map[string]reflect.StructField)

	//遍历sync.Map,要求输入一个func作为参数
	//这个函数的入参、出参的类型都已经固定，不能修改
	//可以在函数体内编写自己的代码,调用map中的k,v
	f := func(k, v interface{}) bool {
		// fmt.Println(k, ":", v)
		field := v.(reflect.StructField)
		fieldName := field.Name
		if ast.IsExported(fieldName) { //如果是可以输出的
			exportStructFieldMap[fieldName] = field
			//如果是数据库字段
			tagColumnName := field.Tag.Get(tagColumnName)
			if len(tagColumnName) > 0 {
				dbColumnFieldMap[tagColumnName] = field
			}

		} else { //私有属性
			privateStructFieldMap[fieldName] = field
		}

		return true
	}
	allFieldMap.Range(f)

	//加入缓存
	cacheStructFieldInfoMap.Store(exportCacheKey, exportStructFieldMap)
	cacheStructFieldInfoMap.Store(privateCacheKey, privateStructFieldMap)
	cacheStructFieldInfoMap.Store(dbColumnCacheKey, dbColumnFieldMap)

	return nil
}

//递归调用struct的匿名属性,就近覆盖属性.
func recursiveAnonymousStruct(allFieldMap *sync.Map, anonymous []reflect.StructField) {

	for i := 0; i < len(anonymous); i++ {
		field := anonymous[i]
		typeOf := field.Type

		if typeOf.Kind() == reflect.Ptr {
			//获取指针下的Struct类型
			typeOf = typeOf.Elem()
		}

		//只处理Struct类型
		if typeOf.Kind() != reflect.Struct {
			continue
		}

		//获取字段长度
		fieldNum := typeOf.NumField()
		//如果没有字段
		if fieldNum < 1 {
			continue
		}

		// 匿名struct里自身又有匿名struct
		anonymousField := make([]reflect.StructField, 0)

		//遍历所有字段
		for i := 0; i < fieldNum; i++ {
			field := typeOf.Field(i)
			if _, ok := allFieldMap.Load(field.Name); ok { //如果存在属性名
				continue
			} else { //不存在属性名,加入到allFieldMap
				allFieldMap.Store(field.Name, field)
			}

			if field.Anonymous { //匿名struct里自身又有匿名struct
				anonymousField = append(anonymousField, field)
			}
		}

		//递归调用匿名struct
		recursiveAnonymousStruct(allFieldMap, anonymousField)

	}

}

//根据数据库的字段名,找到struct映射的字段,并赋值
func setFieldValueByColumnName(entity interface{}, columnName string, value interface{}) error {
	//先从本地缓存中查找
	typeOf := reflect.TypeOf(entity)
	valueOf := reflect.ValueOf(entity)
	if typeOf.Kind() == reflect.Ptr { //如果是指针
		typeOf = typeOf.Elem()
		valueOf = valueOf.Elem()
	}

	dbMap, err := getDBColumnFieldMap(typeOf)
	if err != nil {
		return err
	}
	f, ok := dbMap[columnName]
	if ok { //给主键赋值
		valueOf.FieldByName(f.Name).Set(reflect.ValueOf(value))
	}
	return nil

}

//获取指定字段的值
func structFieldValue(s interface{}, fieldName string) (interface{}, error) {

	if s == nil || len(fieldName) < 1 {
		return nil, errors.New("数据为空")
	}
	//entity的s类型
	valueOf := reflect.ValueOf(s)

	kind := valueOf.Kind()
	if !(kind == reflect.Ptr || kind == reflect.Struct) {
		return nil, errors.New("必须是Struct或者*Struct类型")
	}

	if kind == reflect.Ptr {
		//获取指针下的Struct类型
		valueOf = valueOf.Elem()
		if valueOf.Kind() != reflect.Struct {
			return nil, errors.New("必须是Struct或者*Struct类型")
		}
	}

	//FieldByName方法返回的是reflect.Value类型,调用Interface()方法,返回原始类型的数据值
	value := valueOf.FieldByName(fieldName).Interface()

	return value, nil

}

//深度拷贝对象.golang没有构造函数,反射复制对象时,对象中struct类型的属性无法初始化,指针属性也会收到影响.使用深度对象拷贝
func deepCopy(dst, src interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}

func getDBColumnFieldMap(typeOf reflect.Type) (map[string]reflect.StructField, error) {
	entityName := typeOf.String()

	dbColumnFieldMap, dbOk := cacheStructFieldInfoMap.Load(dbColumnNamePrefix + entityName)
	if !dbOk { //缓存不存在
		//获取实体类的输出字段和私有 字段
		err := structFieldInfo(typeOf)
		if err != nil {
			return nil, err
		}
		dbColumnFieldMap, dbOk = cacheStructFieldInfoMap.Load(dbColumnNamePrefix + entityName)
	}

	dbMap, efOK := dbColumnFieldMap.(map[string]reflect.StructField)
	if !efOK {
		return nil, errors.New("缓存数据库字段异常")
	}
	return dbMap, nil
}
