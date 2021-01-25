package zorm

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/gob"
	"errors"
	"fmt"
	"go/ast"
	"reflect"
	"strings"
	"sync"
)

//allowBaseTypeMap 允许基础类型查询,用于查询单个基础类型字段,例如 select id from t_user 查询返回的是字符串类型
/*
var allowBaseTypeMap = map[reflect.Kind]bool{
	reflect.String: true,

	reflect.Int:   true,
	reflect.Int8:  true,
	reflect.Int16: true,
	reflect.Int32: true,
	reflect.Int64: true,

	reflect.Uint:   true,
	reflect.Uint8:  true,
	reflect.Uint16: true,
	reflect.Uint32: true,
	reflect.Uint64: true,

	reflect.Float32: true,
	reflect.Float64: true,
}
*/

const (
	//tag标签的名称
	tagColumnName = "column"

	//输出字段 缓存的前缀
	exportPrefix = "_exportStructFields_"
	//私有字段 缓存的前缀
	privatePrefix = "_privateStructFields_"
	//数据库列名 缓存的前缀
	dbColumnNamePrefix = "_dbColumnName_"

	//field对应的column的tag值 缓存的前缀
	structFieldTagPrefix = "_structFieldTag_"
	//数据库主键  缓存的前缀
	dbPKNamePrefix = "_dbPKName_"
)

// 用于缓存反射的信息,sync.Map内部处理了并发锁
//var cacheStructFieldInfoMap *sync.Map = &sync.Map{}
var cacheStructFieldInfoMap = make(map[string]map[string]reflect.StructField)

//用于缓存field对应的column的tag值
//var cacheStructFieldTagInfoMap = make(map[string]map[string]string)

//获取StructField的信息.只对struct或者*struct判断,如果是指针,返回指针下实际的struct类型.
//第一个返回值是可以输出的字段(首字母大写),第二个是不能输出的字段(首字母小写)
func structFieldInfo(typeOf reflect.Type) error {

	if typeOf == nil {
		return errors.New("structFieldInfo数据为空")
	}

	entityName := typeOf.String()

	//缓存的key
	exportCacheKey := exportPrefix + entityName
	privateCacheKey := privatePrefix + entityName
	dbColumnCacheKey := dbColumnNamePrefix + entityName
	//structFieldTagCacheKey := structFieldTagPrefix + entityName
	//dbPKNameCacheKey := dbPKNamePrefix + entityName
	//缓存的数据库主键值
	//_, exportOk := cacheStructFieldInfoMap.Load(exportCacheKey)
	_, exportOk := cacheStructFieldInfoMap[exportCacheKey]
	//如果存在值,认为缓存中有所有的信息,不再处理
	if exportOk {
		return nil
	}
	//获取字段长度
	fieldNum := typeOf.NumField()
	//如果没有字段
	if fieldNum < 1 {
		return errors.New("structFieldInfo-->NumField entity没有属性")
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
	//structFieldTagMap := make(map[string]string)

	//遍历sync.Map,要求输入一个func作为参数
	//这个函数的入参、出参的类型都已经固定,不能修改
	//可以在函数体内编写自己的代码,调用map中的k,v
	f := func(k, v interface{}) bool {
		// fmt.Println(k, ":", v)
		field := v.(reflect.StructField)
		fieldName := field.Name
		if ast.IsExported(fieldName) { //如果是可以输出的
			exportStructFieldMap[fieldName] = field
			//如果是数据库字段
			tagColumnValue := field.Tag.Get(tagColumnName)
			if len(tagColumnValue) > 0 {
				//dbColumnFieldMap[tagColumnValue] = field
				//使用数据库字段的小写,处理oracle和达梦数据库的sql返回值大写
				dbColumnFieldMap[strings.ToLower(tagColumnValue)] = field
				//structFieldTagMap[fieldName] = tagColumnValue
			}

		} else { //私有属性
			privateStructFieldMap[fieldName] = field
		}

		return true
	}
	allFieldMap.Range(f)

	//加入缓存
	//cacheStructFieldInfoMap.Store(exportCacheKey, exportStructFieldMap)
	//cacheStructFieldInfoMap.Store(privateCacheKey, privateStructFieldMap)
	//cacheStructFieldInfoMap.Store(dbColumnCacheKey, dbColumnFieldMap)

	cacheStructFieldInfoMap[exportCacheKey] = exportStructFieldMap
	cacheStructFieldInfoMap[privateCacheKey] = privateStructFieldMap
	cacheStructFieldInfoMap[dbColumnCacheKey] = dbColumnFieldMap
	//cacheStructFieldTagInfoMap[structFieldTagCacheKey] = structFieldTagMap
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
	f, ok := dbMap[strings.ToLower(columnName)]
	if ok { //给主键赋值
		valueOf.FieldByName(f.Name).Set(reflect.ValueOf(value))
	}
	return nil

}

//获取指定字段的值
func structFieldValue(s interface{}, fieldName string) (interface{}, error) {

	if s == nil || len(fieldName) < 1 {
		return nil, errors.New("structFieldValue数据为空")
	}
	//entity的s类型
	valueOf := reflect.ValueOf(s)

	kind := valueOf.Kind()
	if !(kind == reflect.Ptr || kind == reflect.Struct) {
		return nil, errors.New("structFieldValue必须是Struct或者*Struct类型")
	}

	if kind == reflect.Ptr {
		//获取指针下的Struct类型
		valueOf = valueOf.Elem()
		if valueOf.Kind() != reflect.Struct {
			return nil, errors.New("structFieldValue必须是Struct或者*Struct类型")
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

	//dbColumnFieldMap, dbOk := cacheStructFieldInfoMap.Load(dbColumnNamePrefix + entityName)
	dbColumnFieldMap, dbOk := cacheStructFieldInfoMap[dbColumnNamePrefix+entityName]
	if !dbOk { //缓存不存在
		//获取实体类的输出字段和私有 字段
		err := structFieldInfo(typeOf)
		if err != nil {
			return nil, err
		}
		//dbColumnFieldMap, dbOk = cacheStructFieldInfoMap.Load(dbColumnNamePrefix + entityName)
		dbColumnFieldMap, dbOk = cacheStructFieldInfoMap[dbColumnNamePrefix+entityName]
	}

	/*
		dbMap, efOK := dbColumnFieldMap.(map[string]reflect.StructField)
		if !efOK {
			return nil, errors.New("缓存数据库字段异常")
		}
		return dbMap, nil
	*/
	return dbColumnFieldMap, nil
}

/*
//获取 fileName 属性 中 tag column的值
func getStructFieldTagColumnValue(typeOf reflect.Type, fieldName string) string {
	entityName := typeOf.String()
	structFieldTagMap, dbOk := cacheStructFieldTagInfoMap[structFieldTagPrefix+entityName]
	if !dbOk { //缓存不存在
		//获取实体类的输出字段和私有 字段
		err := structFieldInfo(typeOf)
		if err != nil {
			return ""
		}
		structFieldTagMap, dbOk = cacheStructFieldTagInfoMap[structFieldTagPrefix+entityName]
	}

	return structFieldTagMap[fieldName]
}
*/

//根据保存的对象,返回插入的语句,需要插入的字段,字段的值.
func columnAndValue(entity interface{}) (reflect.Type, []reflect.StructField, []interface{}, error) {
	typeOf, checkerr := checkEntityKind(entity)
	if checkerr != nil {
		return typeOf, nil, nil, checkerr
	}
	// 获取实体类的反射,指针下的struct
	valueOf := reflect.ValueOf(entity).Elem()
	//reflect.Indirect

	//先从本地缓存中查找
	//typeOf := reflect.TypeOf(entity).Elem()

	dbMap, err := getDBColumnFieldMap(typeOf)
	if err != nil {
		return typeOf, nil, nil, err
	}

	//实体类公开字段的长度
	fLen := len(dbMap)
	//接收列的数组,这里是做一个副本,避免外部更改掉原始的列信息
	columns := make([]reflect.StructField, 0, fLen)
	//接收值的数组
	values := make([]interface{}, 0, fLen)

	//遍历所有数据库属性
	for _, field := range dbMap {
		//获取字段类型的Kind
		//	fieldKind := field.Type.Kind()
		//if !allowTypeMap[fieldKind] { //不允许的类型
		//	continue
		//}

		columns = append(columns, field)
		//FieldByName方法返回的是reflect.Value类型,调用Interface()方法,返回原始类型的数据值.字段不会重名,不使用FieldByIndex()函数
		value := valueOf.FieldByName(field.Name).Interface()

		/*
			if value != nil { //如果不是nil
				timeValue, ok := value.(time.Time)
				if ok && timeValue.IsZero() { //如果是日期零时,需要设置一个初始值1970-01-01 00:00:01,兼容数据库
					value = defaultZeroTime
				}
			}
		*/

		//添加到记录值的数组
		values = append(values, value)

	}

	//缓存数据库的列

	return typeOf, columns, values, nil

}

//获取实体类主键属性名称
func entityPKFieldName(entity IEntityStruct, typeOf reflect.Type) (string, error) {

	//检查是否是指针对象
	//typeOf, checkerr := checkEntityKind(entity)
	//if checkerr != nil {
	//	return "", checkerr
	//}

	//缓存的key,TypeOf和ValueOf的String()方法,返回值不一样
	//typeOf := reflect.TypeOf(entity).Elem()

	dbMap, err := getDBColumnFieldMap(typeOf)
	if err != nil {
		return "", err
	}
	field := dbMap[strings.ToLower(entity.GetPKColumnName())]
	return field.Name, nil

}

//检查entity类型必须是*struct类型或者基础类型的指针
func checkEntityKind(entity interface{}) (reflect.Type, error) {
	if entity == nil {
		return nil, errors.New("checkEntityKind参数不能为空,必须是*struct类型或者基础类型的指针")
	}
	typeOf := reflect.TypeOf(entity)
	if typeOf.Kind() != reflect.Ptr { //如果不是指针
		return nil, errors.New("checkEntityKind必须是*struct类型或者基础类型的指针")
	}
	typeOf = typeOf.Elem()
	//if !(typeOf.Kind() == reflect.Struct || allowBaseTypeMap[typeOf.Kind()]) { //如果不是指针
	//	return nil, errors.New("checkEntityKind必须是*struct类型或者基础类型的指针")
	//}
	return typeOf, nil
}

// sqlRowsValues 包装接收sqlRows的Values数组,反射rows屏蔽数据库null值
// fix:converting NULL to int is unsupported
// 当读取数据库的值为NULL时,由于基本类型不支持为NULL,通过反射将未知driver.Value改为interface{},不再映射到struct实体类
// 感谢@fastabler提交的pr
func sqlRowsValues(rows *sql.Rows, driverValue reflect.Value, columnTypes []*sql.ColumnType, dbColumnFieldMap map[string]reflect.StructField, valueOf reflect.Value) error {
	//声明载体数组,用于存放struct的属性指针
	//Declare a carrier array to store the attribute pointer of the struct
	values := make([]interface{}, len(columnTypes))

	fieldTempValueMap := make(map[reflect.Value]*driverValueInfo)

	//反射获取 []driver.Value的值
	//driverValue := reflect.Indirect(reflect.ValueOf(rows))
	//driverValue = driverValue.FieldByName("lastcols")
	//遍历数据库的列名
	//Traverse the database column names
	for i, columnType := range columnTypes {
		column := columnType.Name()
		//从缓存中获取列名的field字段
		//Get the field field of the column name from the cache
		field, fok := dbColumnFieldMap[strings.ToLower(column)]
		//如果列名不存在,就初始化一个空值
		//If the column name does not exist, initialize a null value
		if !fok {
			values[i] = new(interface{})
			continue
		}
		dv := driverValue.Index(i)
		if dv.IsValid() && dv.InterfaceData()[0] == 0 { // 该字段的数据库值是null,取默认值
			values[i] = new(interface{})
		} else {

			fieldValue := valueOf.FieldByName(field.Name)

			//根据接收的类型,获取到设置的转换函数
			converFunc, converOK := CustomDriverValueMap[dv.Elem().Type().String()]
			var tempDriverValue driver.Value
			var errGetDriverValue error
			if converOK {
				tempDriverValue, errGetDriverValue = converFunc.GetDriverValue(columnType, fieldValue.Type())
				if errGetDriverValue != nil {
					errGetDriverValue = fmt.Errorf("QuerySlice-->conver.GetDriverValue异常:%w", errGetDriverValue)
					FuncLogError(errGetDriverValue)
					return errGetDriverValue
				}

				if tempDriverValue != nil { //返回值不为nil
					values[i] = tempDriverValue
					dvinfo := driverValueInfo{}
					dvinfo.converFunc = converFunc
					dvinfo.columnType = columnType
					dvinfo.tempDriverValue = tempDriverValue
					fieldTempValueMap[fieldValue] = &dvinfo
					continue
				}
			}

			//获取struct的属性值的指针地址,字段不会重名,不使用FieldByIndex()函数
			//Get the pointer address of the attribute value of the struct,the field will not have the same name, and the Field By Index() function is not used
			value := fieldValue.Addr().Interface()
			//把指针地址放到数组
			//Put the pointer address into the array
			values[i] = value
		}
	}
	//scan赋值.是一个指针数组,已经根据struct的属性类型初始化了,sql驱动能感知到参数类型,所以可以直接赋值给struct的指针.这样struct的属性就有值了
	//Scan assignment. It is an array of pointers that has been initialized according to the attribute type of the struct.The sql driver can perceive the parameter type,so it can be directly assigned to the pointer of the struct. In this way, the attributes of the struct have values
	scanerr := rows.Scan(values...)
	if scanerr != nil {
		return scanerr
	}

	//循环需要替换的值
	for fieldValue, driverValueInfo := range fieldTempValueMap {
		rightValue, errConverDriverValue := driverValueInfo.converFunc.ConverDriverValue(driverValueInfo.columnType, fieldValue.Type(), driverValueInfo.tempDriverValue)
		if errConverDriverValue != nil {
			errConverDriverValue = fmt.Errorf("QuerySlice-->conver.ConverDriverValue异常:%w", errConverDriverValue)
			FuncLogError(errConverDriverValue)
			return errConverDriverValue
		}
		fieldValue.Set(reflect.ValueOf(rightValue).Elem())
	}

	return scanerr
}

/*

// sqlRowsValuesFast 包装接收sqlRows的Values数组,快速模式,数据库表不能有null值
// Deprecated: 暂时不用
func sqlRowsValuesFast(rows *sql.Rows, columns []string, dbColumnFieldMap map[string]reflect.StructField, valueOf reflect.Value) error {
	//声明载体数组,用于存放struct的属性指针
	values := make([]interface{}, len(columns))

	//遍历数据库的列名
	for i, column := range columns {
		//从缓存中获取列名的field字段
		field, fok := dbColumnFieldMap[column]
		if !fok { //如果列名不存在,就初始化一个空值
			values[i] = new(interface{})
			continue
		}

		//获取struct的属性值的指针地址,字段不会重名,不使用FieldByIndex()函数
		value := valueOf.FieldByName(field.Name).Addr().Interface()
		//把指针地址放到数组
		values[i] = value
	}
	//scan赋值.是一个指针数组,已经根据struct的属性类型初始化了,sql驱动能感知到参数类型,所以可以直接赋值给struct的指针.这样struct的属性就有值了
	scanerr := rows.Scan(values...)
	if scanerr != nil {
		scanerr = fmt.Errorf("rows.Scan异常:%w", scanerr)
		FuncLogError(scanerr)
		return scanerr
	}
	return nil
}

// sqlRowsValues 包装接收sqlRows的Values数组
//  基础类型使用sql.Nullxxx替换,放到sqlNullMap[field.name]*sql.Nullxxx,用于接受数据库的值,用于处理数据库为null的情况,然后再重新替换回去
// Deprecated: 暂时不用
func sqlRowsValues2(rows *sql.Rows, columns []string, dbColumnFieldMap map[string]reflect.StructField, valueOf reflect.Value) error {
	//声明载体数组,用于存放struct的属性指针
	values := make([]interface{}, len(columns))

	//基础类型使用sql.Nullxxx替换,放到sqlNullMap[field.name]*sql.Nullxxx,用于接受数据库的值,用于处理数据库为null的情况,然后再重新替换回去
	sqlNullMap := make(map[string]interface{})

	//遍历数据库的列名
	for i, column := range columns {
		//从缓存中获取列名的field字段
		field, fok := dbColumnFieldMap[column]
		if !fok { //如果列名不存在,就初始化一个空值
			values[i] = new(interface{})
			continue
		}
		//values 中的值
		var value interface{}
		//struct中的字段属性名称
		//fieldName := field.Name

		//fmt.Println(fieldName, "----", field.Type, "--------", field.Type.Kind(), "++++", field.Type.String())
		ftypeString := field.Type.String()
		fkind := field.Type.Kind()
		//判断字段类型
		switch fkind {
		case reflect.String:
			value = &sql.NullString{}
			sqlNullMap[column] = value
		case reflect.Int8, reflect.Int16, reflect.Int, reflect.Int32:
			value = &sql.NullInt32{}
			sqlNullMap[column] = value
		case reflect.Int64:
			value = &sql.NullInt64{}
			sqlNullMap[column] = value
		case reflect.Float32, reflect.Float64:
			value = &sql.NullFloat64{}
			sqlNullMap[column] = value
		case reflect.Bool:
			value = &sql.NullBool{}
			sqlNullMap[column] = value
		case reflect.Struct:
			if ftypeString == "time.Time" {
				value = &sql.NullTime{}
				sqlNullMap[column] = value
			} else if ftypeString == "decimal.Decimal" {
				value = &decimal.NullDecimal{}
				sqlNullMap[column] = value
			} else {
				//获取struct的属性值的指针地址,字段不会重名,不使用FieldByIndex()函数
				value = valueOf.FieldByName(field.Name).Addr().Interface()
			}
			sqlNullMap[column] = value
		default:
			//获取struct的属性值的指针地址,字段不会重名,不使用FieldByIndex()函数
			value = valueOf.FieldByName(field.Name).Addr().Interface()
		}

		//把指针地址放到数组
		values[i] = value
	}
	//scan赋值.是一个指针数组,已经根据struct的属性类型初始化了,sql驱动能感知到参数类型,所以可以直接赋值给struct的指针.这样struct的属性就有值了
	scanerr := rows.Scan(values...)
	if scanerr != nil {
		return scanerr
	}

	//循环需要处理的字段
	for column, value := range sqlNullMap {
		//从缓存中获取列名的field字段
		field, fok := dbColumnFieldMap[column]
		if !fok { //如果列名不存在,就初始化一个空值
			continue
		}
		ftypeString := field.Type.String()
		fkind := field.Type.Kind()
		fieldName := field.Name
		//判断字段类型
		switch fkind {
		case reflect.String:
			vptr, ok := value.(*sql.NullString)
			if vptr.Valid && ok {
				valueOf.FieldByName(fieldName).SetString(vptr.String)
			}
		case reflect.Int8, reflect.Int16, reflect.Int, reflect.Int32:
			vptr, ok := value.(*sql.NullInt32)
			if vptr.Valid && ok {
				valueOf.FieldByName(fieldName).Set(reflect.ValueOf(int(vptr.Int32)))
			}
		case reflect.Int64:
			vptr, ok := value.(*sql.NullInt64)
			if vptr.Valid && ok {
				valueOf.FieldByName(fieldName).SetInt(vptr.Int64)
			}
		case reflect.Float32:
			vptr, ok := value.(*sql.NullFloat64)
			if vptr.Valid && ok {
				valueOf.FieldByName(fieldName).Set(reflect.ValueOf(float32(vptr.Float64)))
			}
		case reflect.Float64:
			vptr, ok := value.(*sql.NullFloat64)
			if vptr.Valid && ok {
				valueOf.FieldByName(fieldName).SetFloat(vptr.Float64)
			}
		case reflect.Bool:
			vptr, ok := value.(*sql.NullBool)
			if vptr.Valid && ok {
				valueOf.FieldByName(fieldName).SetBool(vptr.Bool)
			}
		case reflect.Struct:
			if ftypeString == "time.Time" {
				vptr, ok := value.(*sql.NullTime)
				if vptr.Valid && ok {
					valueOf.FieldByName(fieldName).Set(reflect.ValueOf(vptr.Time))
				}
			} else if ftypeString == "decimal.Decimal" {
				vptr, ok := value.(*decimal.NullDecimal)
				if vptr.Valid && ok {
					valueOf.FieldByName(fieldName).Set(reflect.ValueOf(vptr.Decimal))
				}

			}

		}

	}

	return scanerr
}
*/

//CustomDriverValueMap 用于配置driver.Value和对应的处理关系,key是 drier.Value 的字符串,例如 *dm.DmClob
//一般是放到init方法里进行添加
var CustomDriverValueMap = make(map[string]CustomDriverValueConver)

//CustomDriverValueConver 自定义类型转化接口,用于解决 类似达梦 text --> dm.DmClob --> string类型接收的问题
type CustomDriverValueConver interface {
	//GetDriverValue 根据数据库列类型和实体类字段类型,返回driver.Value的实例.如果返回值为nil,则不做类型替换,使用默认方式
	GetDriverValue(columnType *sql.ColumnType, structFieldType reflect.Type) (driver.Value, error)

	//ConverDriverValue 数据库列类型,实体类字段类型,GetDriverValue返回的driver.Value新值, 返回符合接收类型值的指针,指针,指针!!!!
	ConverDriverValue(columnType *sql.ColumnType, structFieldType reflect.Type, tempDriverValue driver.Value) (interface{}, error)
}
type driverValueInfo struct {
	converFunc      CustomDriverValueConver
	columnType      *sql.ColumnType
	tempDriverValue interface{}
}

/**

//实现CustomDriverValueConver接口,扩展自定义类型,例如 达梦数据库text类型,映射出来的是dm.DmClob类型,无法使用string类型直接接收
type CustomDMText struct{}
//GetDriverValue 根据数据库列类型和实体类字段类型,返回driver.Value的实例.如果返回值为nil,则不做类型替换,使用默认方式
func (dmtext CustomDMText) GetDriverValue(columnType *sql.ColumnType, structFieldType reflect.Type) (driver.Value, error) {
	return &dm.DmClob{}, nil
}
//ConverDriverValue 数据库列类型,实体类字段类型,GetDriverValue返回的driver.Value新值, 返回符合接收类型值的指针,指针,指针!!!!
func (dmtext CustomDMText) ConverDriverValue(columnType *sql.ColumnType, structFieldType reflect.Type, tempDriverValue driver.Value) (interface{}, error) {
	dmClob, _ := tempDriverValue.(*dm.DmClob)
	dmlen, _ := dmClob.GetLength()
	strInt64 := strconv.FormatInt(dmlen, 10)
	dmlenInt, _ := strconv.Atoi(strInt64)
	str, _ := dmClob.ReadString(1, dmlenInt)
	return &str, nil
}
//CustomDriverValueMap 用于配置driver.Value和对应的处理关系,key是 drier.Value 的字符串,例如 *dm.DmClob
//一般是放到init方法里进行添加
zorm.CustomDriverValueMap["*dm.DmClob"] = CustomDMText{}

**/
