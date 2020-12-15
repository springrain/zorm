package zorm

import (
	"database/sql"
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gitee.com/chunanyong/gouuid"
)

//包装基础的SQL语句
func wrapSQL(dbType string, sqlstr string) (string, error) {

	//根据数据库类型,调整SQL变量符号,例如?,? $1,$2这样的
	sqlstr = rebind(dbType, sqlstr)
	return sqlstr, nil
}

//包装分页的SQL语句
func wrapPageSQL(dbType string, sqlstr string, page *Page) (string, error) {
	//查询order by 的位置.为了保持各个数据库之间的分页语句兼容,要求都要有order by,不然迁移数据库时的风险就很大了.
	locOrderBy := findOrderByIndex(sqlstr)
	if len(locOrderBy) <= 0 { //如果没有 order by
		return "", errors.New("分页语句必须有 order by")
	}
	var sqlbuilder strings.Builder
	sqlbuilder.WriteString(sqlstr)
	if dbType == "mysql" { //MySQL数据库
		sqlbuilder.WriteString(" LIMIT ")
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize * (page.PageNo - 1)))
		sqlbuilder.WriteString(",")
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize))

	} else if dbType == "postgresql" { //postgresql
		sqlbuilder.WriteString(" LIMIT ")
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize))
		sqlbuilder.WriteString(" OFFSET ")
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize * (page.PageNo - 1)))
	} else if dbType == "mssql" { //sqlserver 2012+
		sqlbuilder.WriteString(" OFFSET ")
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize * (page.PageNo - 1)))
		sqlbuilder.WriteString(" ROWS FETCH NEXT ")
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize))
		sqlbuilder.WriteString(" ROWS ONLY ")

	} else if dbType == "oracle" { //oracle 12c+
		sqlbuilder.WriteString(" OFFSET ")
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize * (page.PageNo - 1)))
		sqlbuilder.WriteString(" ROWS FETCH NEXT ")
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize))
		sqlbuilder.WriteString(" ROWS ONLY ")
	} else if dbType == "sqlite" { //sqlite3
		sqlbuilder.WriteString(" LIMIT ")
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize * (page.PageNo - 1)))
		sqlbuilder.WriteString(",")
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize))
	} else if dbType == "dm" { //达梦数据库
		sqlbuilder.WriteString(" LIMIT ")
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize * (page.PageNo - 1)))
		sqlbuilder.WriteString(",")
		sqlbuilder.WriteString(strconv.Itoa(page.PageSize))
	} else if dbType == "db2" { //db2

		//先不写啦
		//bug(springrain) 还需要其他的数据库分页语句
	}
	sqlstr = sqlbuilder.String()
	return wrapSQL(dbType, sqlstr)
}

//包装保存Struct语句.返回语句,是否自增,错误信息
//数组传递,如果外部方法有调用append的逻辑,传递指针,因为append会破坏指针引用
func wrapInsertStructSQL(dbType string, typeOf reflect.Type, entity IEntityStruct, columns *[]reflect.StructField, values *[]interface{}) (string, bool, error) {
	sqlstr, autoIncrement, err := wrapInsertStructSQLNORebuild(dbType, typeOf, entity, columns, values)
	savesql, err := wrapSQL(dbType, sqlstr)
	return savesql, autoIncrement, err
}

//包装保存Struct语句.返回语句,没有rebuild,返回原始的SQL,是否自增,错误信息
//数组传递,如果外部方法有调用append的逻辑,传递指针,因为append会破坏指针引用
func wrapInsertStructSQLNORebuild(dbType string, typeOf reflect.Type, entity IEntityStruct, columns *[]reflect.StructField, values *[]interface{}) (string, bool, error) {

	//是否自增,默认false
	autoIncrement := false
	//SQL语句的构造器
	var sqlBuilder strings.Builder
	sqlBuilder.WriteString("INSERT INTO ")
	sqlBuilder.WriteString(entity.GetTableName())
	sqlBuilder.WriteString("(")

	//SQL语句中,VALUES(?,?,...)语句的构造器
	var valueSQLBuilder strings.Builder
	valueSQLBuilder.WriteString(" VALUES (")
	//主键的名称
	pkFieldName, e := entityPKFieldName(entity, typeOf)
	if e != nil {
		return "", autoIncrement, e
	}
	for i := 0; i < len(*columns); i++ {
		field := (*columns)[i]
		if field.Name == pkFieldName { //如果是主键
			pkKind := field.Type.Kind()

			if !(pkKind == reflect.String || pkKind == reflect.Int || pkKind == reflect.Int8 || pkKind == reflect.Int16 || pkKind == reflect.Int32 || pkKind == reflect.Int64) { //只支持字符串和int类型的主键
				return "", autoIncrement, errors.New("不支持的主键类型")
			}
			//主键的值
			pkValue := (*values)[i]
			if len(entity.GetPkSequence()) > 0 { //如果是主键序列
				//拼接字符串
				//sqlBuilder.WriteString(getStructFieldTagColumnValue(typeOf, field.Name))
				sqlBuilder.WriteString(field.Tag.Get(tagColumnName))
				sqlBuilder.WriteString(",")
				valueSQLBuilder.WriteString(entity.GetPkSequence())
				valueSQLBuilder.WriteString(",")
				//去掉这一列,后续不再处理
				*columns = append((*columns)[:i], (*columns)[i+1:]...)
				*values = append((*values)[:i], (*values)[i+1:]...)
				i = i - 1
				continue

			} else if (pkKind == reflect.String) && (pkValue.(string) == "") { //主键是字符串类型,并且值为"",赋值id
				//生成主键字符串
				id := FuncGenerateStringID()
				(*values)[i] = id
				//给对象主键赋值
				v := reflect.ValueOf(entity).Elem()
				v.FieldByName(field.Name).Set(reflect.ValueOf(id))
				//如果是数字类型,并且值为0,认为是数据库自增,从数组中删除掉主键的信息,让数据库自己生成
			} else if (pkKind == reflect.Int || pkKind == reflect.Int8 || pkKind == reflect.Int16 || pkKind == reflect.Int32 || pkKind == reflect.Int64) && (pkValue.(int) == 0) {
				//标记是自增主键
				autoIncrement = true
				//去掉这一列,后续不再处理
				*columns = append((*columns)[:i], (*columns)[i+1:]...)
				*values = append((*values)[:i], (*values)[i+1:]...)
				i = i - 1
				continue
			}
		}
		//拼接字符串
		//sqlBuilder.WriteString(getStructFieldTagColumnValue(typeOf, field.Name))
		sqlBuilder.WriteString(field.Tag.Get(tagColumnName))
		sqlBuilder.WriteString(",")
		valueSQLBuilder.WriteString("?,")

	}
	//去掉字符串最后的 , 号
	sqlstr := sqlBuilder.String()
	if len(sqlstr) > 0 {
		sqlstr = sqlstr[:len(sqlstr)-1]
	}
	valuestr := valueSQLBuilder.String()
	if len(valuestr) > 0 {
		valuestr = valuestr[:len(valuestr)-1]
	}
	sqlstr = sqlstr + ")" + valuestr + ")"
	//savesql, err := wrapSQL(dbType, sqlstr)
	return sqlstr, autoIncrement, nil

}

//包装批量保存StructSlice语句.返回语句,是否自增,错误信息
//数组传递,如果外部方法有调用append的逻辑,传递指针,因为append会破坏指针引用
func wrapInsertSliceStructSQL(dbType string, typeOf reflect.Type, entityStructSlice []IEntityStruct, columns *[]reflect.StructField, values *[]interface{}) (string, bool, error) {
	sliceLen := len(entityStructSlice)
	if entityStructSlice == nil || sliceLen < 1 {
		return "", false, errors.New("对象数组不能为空")
	}

	//第一个对象,获取第一个Struct对象,用于获取数据库字段,也获取了值
	entity := entityStructSlice[0]

	//先生成一条语句
	sqlstr, autoIncrement, firstErr := wrapInsertStructSQLNORebuild(dbType, typeOf, entity, columns, values)
	if firstErr != nil {
		return sqlstr, autoIncrement, firstErr
	}
	//如果只有一个Struct对象
	if sliceLen == 1 {
		return sqlstr, autoIncrement, firstErr
	}
	//主键的名称
	pkFieldName, e := entityPKFieldName(entity, typeOf)
	if e != nil {
		return sqlstr, autoIncrement, e
	}

	//截取生成的SQL语句中 VALUES 后面的字符串值
	valueIndex := strings.Index(sqlstr, " VALUES (")
	if valueIndex < 1 { //生成的语句异常
		return sqlstr, autoIncrement, errors.New("生成的语句异常")
	}
	//value后面的字符串 例如 (?,?,?),用于循环拼接
	valuestr := sqlstr[valueIndex+8:]
	//SQL语句的构造器
	var insertSliceSQLBuilder strings.Builder
	insertSliceSQLBuilder.WriteString(sqlstr)
	for i := 1; i < sliceLen; i++ {
		//拼接字符串
		insertSliceSQLBuilder.WriteString(",")
		insertSliceSQLBuilder.WriteString(valuestr)

		entityStruct := entityStructSlice[i]
		for j := 0; j < len(*columns); j++ {
			// 获取实体类的反射,指针下的struct
			valueOf := reflect.ValueOf(entityStruct).Elem()
			field := (*columns)[j]
			if field.Name == pkFieldName { //如果是主键
				pkKind := field.Type.Kind()
				//主键的值
				pkValue := valueOf.FieldByName(field.Name).Interface()
				//只处理字符串类型的主键,其他类型,columns中并不包含
				if (pkKind == reflect.String) && (pkValue.(string) == "") { //主键是字符串类型,并且值为"",赋值id
					//生成主键字符串
					id := FuncGenerateStringID()
					*values = append(*values, id)
					//给对象主键赋值
					valueOf.FieldByName(field.Name).Set(reflect.ValueOf(id))
					continue
				}
			}

			//给字段赋值
			*values = append(*values, valueOf.FieldByName(field.Name).Interface())

		}
	}

	//包装sql
	savesql, err := wrapSQL(dbType, insertSliceSQLBuilder.String())
	return savesql, autoIncrement, err

}

//包装更新Struct语句
//数组传递,如果外部方法有调用append的逻辑,传递指针,因为append会破坏指针引用
func wrapUpdateStructSQL(dbType string, typeOf reflect.Type, entity IEntityStruct, columns *[]reflect.StructField, values *[]interface{}, onlyUpdateNotZero bool) (string, error) {

	//SQL语句的构造器
	var sqlBuilder strings.Builder
	sqlBuilder.WriteString("UPDATE ")
	sqlBuilder.WriteString(entity.GetTableName())
	sqlBuilder.WriteString(" SET ")

	//主键的值
	var pkValue interface{}
	//主键的名称
	pkFieldName, e := entityPKFieldName(entity, typeOf)
	if e != nil {
		return "", e
	}

	for i := 0; i < len(*columns); i++ {
		field := (*columns)[i]
		if field.Name == pkFieldName { //如果是主键
			pkValue = (*values)[i]
			//去掉这一列,最后处理主键
			*columns = append((*columns)[:i], (*columns)[i+1:]...)
			*values = append((*values)[:i], (*values)[i+1:]...)
			i = i - 1
			continue
		}

		//如果是默认值字段,删除掉,不更新
		if onlyUpdateNotZero && (reflect.ValueOf((*values)[i]).IsZero()) {
			//去掉这一列,不再处理
			*columns = append((*columns)[:i], (*columns)[i+1:]...)
			*values = append((*values)[:i], (*values)[i+1:]...)
			i = i - 1
			continue

		}
		//sqlBuilder.WriteString(getStructFieldTagColumnValue(typeOf, field.Name))
		sqlBuilder.WriteString(field.Tag.Get(tagColumnName))
		sqlBuilder.WriteString("=?,")

	}
	//主键的值是最后一个
	*values = append(*values, pkValue)
	//去掉字符串最后的 , 号
	sqlstr := sqlBuilder.String()
	sqlstr = sqlstr[:len(sqlstr)-1]

	sqlstr = sqlstr + " WHERE " + entity.GetPKColumnName() + "=?"

	return wrapSQL(dbType, sqlstr)
}

//包装删除Struct语句
func wrapDeleteStructSQL(dbType string, entity IEntityStruct) (string, error) {

	//SQL语句的构造器
	var sqlBuilder strings.Builder
	sqlBuilder.WriteString("DELETE FROM ")
	sqlBuilder.WriteString(entity.GetTableName())
	sqlBuilder.WriteString(" WHERE ")
	sqlBuilder.WriteString(entity.GetPKColumnName())
	sqlBuilder.WriteString("=?")
	sqlstr := sqlBuilder.String()

	return wrapSQL(dbType, sqlstr)

}

//包装保存Map语句,Map因为没有字段属性,无法完成Id的类型判断和赋值,需要确保Map的值是完整的.
func wrapSaveMapSQL(dbType string, entity IEntityMap) (string, []interface{}, bool, error) {
	//是否自增,默认false
	autoIncrement := false
	dbFieldMap := entity.GetDBFieldMap()
	if len(dbFieldMap) < 1 {
		return "", nil, autoIncrement, errors.New("GetDBFieldMap()返回值不能为空")
	}
	//SQL对应的参数
	values := []interface{}{}

	//SQL语句的构造器
	var sqlBuilder strings.Builder
	sqlBuilder.WriteString("INSERT INTO ")
	sqlBuilder.WriteString(entity.GetTableName())
	sqlBuilder.WriteString("(")

	//SQL语句中,VALUES(?,?,...)语句的构造器
	var valueSQLBuilder strings.Builder
	valueSQLBuilder.WriteString(" VALUES (")
	//是否Set了主键
	_, hasPK := dbFieldMap[entity.GetPKColumnName()]
	if !hasPK { //如果没有设置主键,认为是自增或者序列
		autoIncrement = true
		if len(entity.GetPkSequence()) > 0 { //如果是序列
			sqlBuilder.WriteString(entity.GetPKColumnName())
			sqlBuilder.WriteString(",")
			valueSQLBuilder.WriteString(entity.GetPkSequence())
			valueSQLBuilder.WriteString(",")
		}
	}

	for k, v := range dbFieldMap {
		//拼接字符串
		sqlBuilder.WriteString(k)
		sqlBuilder.WriteString(",")
		valueSQLBuilder.WriteString("?,")
		values = append(values, v)
	}
	//去掉字符串最后的 , 号
	sqlstr := sqlBuilder.String()
	if len(sqlstr) > 0 {
		sqlstr = sqlstr[:len(sqlstr)-1]
	}
	valuestr := valueSQLBuilder.String()
	if len(valuestr) > 0 {
		valuestr = valuestr[:len(valuestr)-1]
	}
	sqlstr = sqlstr + ")" + valuestr + ")"

	var e error
	sqlstr, e = wrapSQL(dbType, sqlstr)
	if e != nil {
		return "", nil, autoIncrement, e
	}
	return sqlstr, values, autoIncrement, nil
}

//包装Map更新语句,Map因为没有字段属性,无法完成Id的类型判断和赋值,需要确保Map的值是完整的.
func wrapUpdateMapSQL(dbType string, entity IEntityMap) (string, []interface{}, error) {
	dbFieldMap := entity.GetDBFieldMap()
	if len(dbFieldMap) < 1 {
		return "", nil, errors.New("GetDBFieldMap()返回值不能为空")
	}
	//SQL语句的构造器
	var sqlBuilder strings.Builder
	sqlBuilder.WriteString("UPDATE ")
	sqlBuilder.WriteString(entity.GetTableName())
	sqlBuilder.WriteString(" SET ")

	//SQL对应的参数
	values := []interface{}{}
	//主键名称
	var pkValue interface{}

	for k, v := range dbFieldMap {

		if k == entity.GetPKColumnName() { //如果是主键
			pkValue = v
			continue
		}

		//拼接字符串
		sqlBuilder.WriteString(k)
		sqlBuilder.WriteString("=?,")
		values = append(values, v)
	}
	//主键的值是最后一个
	values = append(values, pkValue)
	//去掉字符串最后的 , 号
	sqlstr := sqlBuilder.String()
	sqlstr = sqlstr[:len(sqlstr)-1]

	sqlstr = sqlstr + " WHERE " + entity.GetPKColumnName() + "=?"

	var e error
	sqlstr, e = wrapSQL(dbType, sqlstr)
	if e != nil {
		return "", nil, e
	}
	return sqlstr, values, nil
}

//封装查询语句
func wrapQuerySQL(dbType string, finder *Finder, page *Page) (string, error) {

	//获取到没有page的sql的语句
	sqlstr, err := finder.GetSQL()
	if err != nil {
		return "", err
	}
	if page == nil {
		sqlstr, err = wrapSQL(dbType, sqlstr)
	} else {
		sqlstr, err = wrapPageSQL(dbType, sqlstr, page)
	}

	if err != nil {
		return "", err
	}
	return sqlstr, err
}

//根据数据库类型,调整SQL变量符号,例如?,? $1,$2这样的
func rebind(dbType string, sqlstr string) string {
	if dbType == "mysql" || dbType == "sqlite" {
		return sqlstr
	}

	strs := strings.Split(sqlstr, "?")
	if len(strs) < 1 {
		return sqlstr
	}
	var sqlBuilder strings.Builder
	sqlBuilder.WriteString(strs[0])
	for i := 1; i < len(strs); i++ {
		if dbType == "postgresql" { //postgresql
			sqlBuilder.WriteString("$")
			sqlBuilder.WriteString(strconv.Itoa(i))
		} else if dbType == "mssql" { //mssql
			sqlBuilder.WriteString("@p")
			sqlBuilder.WriteString(strconv.Itoa(i))
		} else if dbType == "oracle" { //oracle
			sqlBuilder.WriteString(":")
			sqlBuilder.WriteString(strconv.Itoa(i))
		} else { //其他情况,还是使用?
			sqlBuilder.WriteString("?")
		}
		sqlBuilder.WriteString(strs[i])
	}
	return sqlBuilder.String()
}

//查询order by在sql中出现的开始位置和结束位置
var orderByExpr = "\\s+(order)\\s+(by)+\\s"
var orderByRegexp, _ = regexp.Compile(orderByExpr)

//查询order by在sql中出现的开始位置和结束位置
func findOrderByIndex(strsql string) []int {
	loc := orderByRegexp.FindStringIndex(strings.ToLower(strsql))
	return loc
}

//查询group by在sql中出现的开始位置和结束位置
var groupByExpr = "\\s+(group)\\s+(by)+\\s"
var groupByRegexp, _ = regexp.Compile(groupByExpr)

//查询group by在sql中出现的开始位置和结束位置
func findGroupByIndex(strsql string) []int {
	loc := groupByRegexp.FindStringIndex(strings.ToLower(strsql))
	return loc
}

//查询 from 在sql中出现的开始位置和结束位置
var fromExpr = "\\s+(from)+\\s"
var fromRegexp, _ = regexp.Compile(fromExpr)

//查询from在sql中出现的开始位置和结束位置
func findFromIndex(strsql string) []int {
	loc := fromRegexp.FindStringIndex(strings.ToLower(strsql))
	return loc
}

//根据数据库的字段类型,转化成golang的类型,不处理sql.Nullxxx类型
func converValueColumnType(v interface{}, columnType *sql.ColumnType) interface{} {

	if v == nil {
		return nil
	}

	//如果是字节数组
	value, ok := v.([]byte)
	if !ok { //转化失败
		return v
	}
	if len(value) < 1 { //值为空,为nil
		return nil
	}

	//获取数据库类型,自己对应golang的基础类型值,不处理sql.Nullxxx类型
	databaseTypeName := strings.ToUpper(columnType.DatabaseTypeName())
	//如果是字符串
	if databaseTypeName == "VARCHAR" || databaseTypeName == "NVARCHAR" || databaseTypeName == "TEXT" {
		return typeConvertString(v)
	} else if databaseTypeName == "TINYINT" { //如果是TINYINT
		return typeConvertInt8(v)
	} else if databaseTypeName == "SMALLINT" { //如果是SMALLINT
		return typeConvertInt16(v)
	} else if databaseTypeName == "INT" { //如果是INT
		return typeConvertInt(v)
	} else if databaseTypeName == "BIGINT" { //如果是BIGINT
		return typeConvertInt64(v)
	} else if databaseTypeName == "FLOAT" { //如果是FLOAT
		return typeConvertFloat32(v)
	} else if databaseTypeName == "DOUBLE" { //如果是DOUBLE
		return typeConvertFloat64(v)
	} else if databaseTypeName == "DECIMAL" { //如果是DECIMAL
		return typeConvertDecimal(v)
	} else if databaseTypeName == "DATETIME" { //如果是DATETIME
		return typeConvertTime(v, "2006-01-02 15:04:05", time.Local)
	} else if databaseTypeName == "TIMESTAMP" { //如果是TIMESTAMP
		return typeConvertTime(v, "2006-01-02 15:04:05.000", time.Local)
	}
	//其他类型以后再写.....

	return nil
}

//FuncGenerateStringID 默认生成字符串ID的函数.方便自定义扩展
var FuncGenerateStringID func() string = generateStringID

//generateStringID 生成主键字符串
func generateStringID() string {
	//pk := strconv.FormatInt(time.Now().UnixNano(), 10)
	pk, errUUID := gouuid.NewV4()
	if errUUID != nil {
		return ""
	}
	return pk.String()
}
