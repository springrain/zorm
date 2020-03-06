package zorm

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

//Finder 查询数据库的载体,所有的sql语句都要通过Finder执行.
type Finder struct {
	//拼接SQL
	sqlBuilder strings.Builder
	//SQL的参数值
	values []interface{}
	//注入检查,默认true 不允许SQL注入的 ' 单引号
	InjectionCheck bool
	//CountFinder 自定义的查询总条数Finder,使用指针默认为nil.主要是为了在group by等复杂情况下,为了性能,手动编写总条数语句
	CountFinder *Finder
	//是否自动查询总条数,默认true.同时需要Page不为nil,才查询总条数
	SelectTotalCount bool
	//SQL语句
	sqlstr string
}

//NewFinder 初始化一个Finder,生成一个空的Finder
func NewFinder() *Finder {
	finder := Finder{}
	finder.SelectTotalCount = true
	finder.InjectionCheck = true
	finder.values = make([]interface{}, 0)
	return &finder
}

//NewSelectFinder 根据表名初始化查询的Finder
//NewSelectFinder("tableName") SELECT * FROM tableName
//NewSelectFinder("tableName", "id,name") SELECT id,name FROM tableName
func NewSelectFinder(tableName string, strs ...string) *Finder {
	finder := NewFinder()
	finder.sqlBuilder.WriteString("SELECT ")
	if len(strs) > 0 {
		for _, str := range strs {
			finder.sqlBuilder.WriteString(str)
		}
	} else {
		finder.sqlBuilder.WriteString("*")
	}
	finder.sqlBuilder.WriteString(" FROM ")
	finder.sqlBuilder.WriteString(tableName)
	return finder
}

//NewUpdateFinder 根据表名初始化更新的Finder,  UPDATE tableName SET
func NewUpdateFinder(tableName string) *Finder {
	finder := NewFinder()
	finder.sqlBuilder.WriteString("UPDATE ")
	finder.sqlBuilder.WriteString(tableName)
	finder.sqlBuilder.WriteString(" SET ")
	return finder
}

//NewDeleteFinder 根据表名初始化删除的Finder,  DELETE FROM tableName
func NewDeleteFinder(tableName string) *Finder {
	finder := NewFinder()
	finder.sqlBuilder.WriteString("DELETE FROM ")
	finder.sqlBuilder.WriteString(tableName)
	//所有的 WHERE 都不加,规则统一,好记
	//finder.sqlBuilder.WriteString(" WHERE ")
	return finder
}

//Append 添加SQL和参数的值,第一个参数是语句,后面的参数[可选]是参数的值,顺序要正确.
//例如: finder.Append(" and id=? and name=? ",23123,"abc")
//只拼接SQL,例如: finder.Append(" and name=123 ")
func (finder *Finder) Append(s string, values ...interface{}) *Finder {

	//不要自己构建finder,使用Newxxx方法
	if finder.values == nil {
		return nil
	}

	if len(s) > 0 {
		if len(finder.sqlstr) > 0 {
			finder.sqlstr = ""
		}
		finder.sqlBuilder.WriteString(s)
	}
	if values == nil || len(values) < 1 {
		return finder
	}
	//for _, v := range values {
	//	finder.Values = append(finder.Values, v)
	//}
	finder.values = append(finder.values, values...)
	return finder
}

//AppendFinder 添加另一个Finder finder.AppendFinder(f)
func (finder *Finder) AppendFinder(f *Finder) (*Finder, error) {
	if f == nil {
		return nil, errors.New("参数是nil")
	}

	//不要自己构建finder,使用Newxxx方法
	if finder.values == nil {
		return nil, errors.New("不要自己构建finder,使用Newxxx方法")
	}

	//添加f的SQL
	sqlstr, err := f.GetSQL()
	if err != nil {
		return nil, err
	}
	finder.sqlstr = ""
	finder.sqlBuilder.WriteString(sqlstr)
	//添加f的值
	finder.values = append(finder.values, f.values...)
	return finder, nil
}

//GetSQL 返回Finder封装的SQL语句
func (finder *Finder) GetSQL() (string, error) {
	//不要自己构建finder,使用Newxxx方法
	if finder.values == nil {
		return "", errors.New("不要自己构建finder,使用Newxxx方法")
	}
	if len(finder.sqlstr) > 0 {
		return finder.sqlstr, nil
	}
	sqlstr := finder.sqlBuilder.String()
	finder.sqlstr = sqlstr
	//包含单引号,属于非法字符串
	if finder.InjectionCheck && (strings.Index(sqlstr, "'") >= 0) {
		return sqlstr, errors.New("SQL语句请不要直接拼接字符串参数!!!使用标准的占位符实现,例如  finder.Append(' and id=? and name=? ','123','abc')")
	}

	//处理sql语句中的in,实际就是把数组变量展开,例如 id in(?) ["1","2","3"] 语句变更为 id in (?,?,?) 参数也展开到参数数组里
	//这里认为 slice类型的参数就是in
	if finder.values == nil || len(finder.values) < 1 { //如果没有参数
		return sqlstr, nil
	}

	//?问号切割的数组
	questions := strings.Split(sqlstr, "?")

	//语句中没有?问号
	if len(questions) < 1 {
		return sqlstr, nil
	}

	//重新记录参数值
	newValues := make([]interface{}, 0)
	//新的sql
	var newSQLStr strings.Builder
	//?切割的语句实际长度比?号个数多1,先把第一个语句片段加上,后面就是比参数的索引大1
	newSQLStr.WriteString(questions[0])

	//遍历所有的参数
	for i, v := range finder.values {
		//先拼接?,?号切割之后,?号就丢失了,先补充上
		newSQLStr.WriteString("?")

		valueOf := reflect.ValueOf(v)
		typeOf := reflect.TypeOf(v)
		kind := valueOf.Kind()
		//如果参数是个指针类型
		if kind == reflect.Ptr { //如果是指针
			valueOf = valueOf.Elem()
			typeOf = typeOf.Elem()
			kind = valueOf.Kind()
		}
		//获取数组类型参数值的长度
		sliceLen := valueOf.Len()
		//数组类型的参数长度小于1,认为是有异常的参数
		if sliceLen < 1 {
			return sqlstr, errors.New("语句:" + sqlstr + ",第" + strconv.Itoa(i+1) + "个参数,类型是Array或者Slice,值的长度为0,请检查sql参数有效性")
		}

		//如果不是数组或者slice
		if !(kind == reflect.Array || kind == reflect.Slice) {
			//记录新值
			newValues = append(newValues, v)
			//记录SQL
			newSQLStr.WriteString(questions[i+1])
			continue
		}
		//字节数组是特殊的情况
		if typeOf == reflect.TypeOf([]byte{}) {
			//记录新值
			newValues = append(newValues, v)
			//记录SQL
			newSQLStr.WriteString(questions[i+1])
			continue
		}
		for j := 0; j < sliceLen; j++ {
			//每多一个参数,对应",?" 两个符号.增加的问号长度总计是(sliceLen-1)*2.
			if j >= 1 {
				//记录SQL
				newSQLStr.WriteString(",?")
			}
			//记录新值
			sliceValue := valueOf.Index(j).Interface()
			newValues = append(newValues, sliceValue)
		}
		//记录SQL
		newSQLStr.WriteString(questions[i+1])
	}
	//重新赋值
	finder.sqlstr = newSQLStr.String()
	finder.values = newValues
	return finder.sqlstr, nil
}
