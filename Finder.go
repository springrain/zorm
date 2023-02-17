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
	"errors"
	"strings"
)

// Finder 查询数据库的载体,所有的sql语句都要通过Finder执行.
// Finder To query the database carrier, all SQL statements must be executed through Finder
type Finder struct {
	// 拼接SQL
	// Splicing SQL.
	sqlBuilder strings.Builder
	// SQL的参数值
	// SQL parameter values.
	values []interface{}
	// 注入检查,默认true 不允许SQL注入的 ' 单引号
	// Injection check, default true does not allow SQL injection  single quote
	InjectionCheck bool
	// CountFinder 自定义的查询总条数'Finder',使用指针默认为nil.主要是为了在'group by'等复杂情况下,为了性能,手动编写总条数语句
	// CountFinder The total number of custom queries is'Finder', and the pointer is nil by default. It is mainly used to manually write the total number of statements for performance in complex situations such as'group by'
	CountFinder *Finder
	// 是否自动查询总条数,默认true.同时需要Page不为nil,才查询总条数
	// Whether to automatically query the total number of entries, the default is true. At the same time, the Page is not nil to query the total number of entries
	SelectTotalCount bool
	// SQL语句
	// SQL statement
	sqlstr string
}

// NewFinder  初始化一个Finder,生成一个空的Finder
// NewFinder Initialize a Finder and generate an empty Finder
func NewFinder() *Finder {
	finder := Finder{}
	finder.sqlBuilder.Grow(stringBuilderGrowLen)
	finder.SelectTotalCount = true
	finder.InjectionCheck = true
	// slice扩容会生成新的slice,最后要值复制接收.问:为什么cap是3?答:经验
	finder.values = make([]interface{}, 0, 3)
	return &finder
}

// NewSelectFinder 根据表名初始化查询的Finder,strs 只取第一个字符串,用数组类型是为了可以不传入,默认为 * | Finder that initializes the query based on the table name
// NewSelectFinder("tableName") SELECT * FROM tableName
// NewSelectFinder("tableName", "id,name") SELECT id,name FROM tableName
func NewSelectFinder(tableName string, strs ...string) *Finder {
	strsLen := len(strs)
	if strsLen > 1 { // 不支持多个参数
		return nil
	}
	finder := NewFinder()
	finder.sqlBuilder.WriteString("SELECT ")
	if strsLen == 1 { // 只取值第一个字符串
		finder.sqlBuilder.WriteString(strs[0])
	} else {
		finder.sqlBuilder.WriteByte('*')
	}
	finder.sqlBuilder.WriteString(" FROM ")
	finder.sqlBuilder.WriteString(tableName)
	return finder
}

// NewUpdateFinder 根据表名初始化更新的Finder,  UPDATE tableName SET
// NewUpdateFinder Initialize the updated Finder according to the table name, UPDATE tableName SET
func NewUpdateFinder(tableName string) *Finder {
	finder := NewFinder()
	finder.sqlBuilder.WriteString("UPDATE ")
	finder.sqlBuilder.WriteString(tableName)
	finder.sqlBuilder.WriteString(" SET ")
	return finder
}

// NewDeleteFinder 根据表名初始化删除的'Finder',  DELETE FROM tableName
// NewDeleteFinder Finder for initial deletion based on table name. DELETE FROM tableName
func NewDeleteFinder(tableName string) *Finder {
	finder := NewFinder()
	finder.sqlBuilder.WriteString("DELETE FROM ")
	finder.sqlBuilder.WriteString(tableName)
	// 所有的 WHERE 都不加,规则统一,好记
	// No WHERE is added, the rules are unified, easy to remember
	// finder.sqlBuilder.WriteString(" WHERE ")
	return finder
}

// Append 添加SQL和参数的值,第一个参数是语句,后面的参数[可选]是参数的值,顺序要正确
// 例如: finder.Append(" and id=? and name=? ",23123,"abc")
// 只拼接SQL,例如: finder.Append(" and name=123 ")
// Append:Add SQL and parameter values, the first parameter is the statement, and the following parameter (optional) is the value of the parameter, in the correct order
// E.g:  finder.Append(" and id=? and name=? ",23123,"abc")
// Only splice SQL, E.g: finder.Append(" and name=123 ")
func (finder *Finder) Append(s string, values ...interface{}) *Finder {
	// 不要自己构建finder,使用NewFinder()方法
	// Don't build finder by yourself, use NewFinder() method
	if finder == nil || finder.values == nil {
		return nil
	}

	if s != "" {
		if finder.sqlstr != "" {
			finder.sqlstr = ""
		}
		// 默认加一个空格,避免手误两个字符串连接再一起
		// A space is added by default to avoid hand mistakes when connecting two strings together
		finder.sqlBuilder.WriteByte(' ')

		finder.sqlBuilder.WriteString(s)

	}
	if values == nil || len(values) < 1 {
		return finder
	}

	finder.values = append(finder.values, values...)
	return finder
}

// AppendFinder 添加另一个Finder finder.AppendFinder(f)
// AppendFinder Add another Finder . finder.AppendFinder(f)
func (finder *Finder) AppendFinder(f *Finder) (*Finder, error) {
	if finder == nil {
		return finder, errors.New("->finder-->AppendFinder()finder对象为nil")
	}
	if f == nil {
		return finder, errors.New("->finder-->AppendFinder()参数是nil")
	}

	// 不要自己构建finder,使用NewFinder()方法
	// Don't build finder by yourself, use NewFinder() method
	if finder.values == nil {
		return finder, errors.New("->finder-->AppendFinder()不要自己构建finder,使用NewFinder()方法")
	}

	// 添加f的SQL
	// SQL to add f
	sqlstr, err := f.GetSQL()
	if err != nil {
		return finder, err
	}
	finder.sqlstr = ""
	finder.sqlBuilder.WriteString(sqlstr)
	// 添加f的值
	// Add the value of f
	finder.values = append(finder.values, f.values...)
	return finder, nil
}

// GetSQL 返回Finder封装的SQL语句
// GetSQL Return the SQL statement encapsulated by the Finder
func (finder *Finder) GetSQL() (string, error) {
	// 不要自己构建finder,使用NewFinder方法
	// Don't build finder by yourself, use NewFinder method
	if finder == nil || finder.values == nil {
		return "", errors.New("->finder-->GetSQL()不要自己构建finder,使用NewFinder()方法")
	}
	if len(finder.sqlstr) > 0 {
		return finder.sqlstr, nil
	}
	sqlstr := finder.sqlBuilder.String()
	// 包含单引号,属于非法字符串
	// Contains single quotes, which are illegal strings
	if finder.InjectionCheck && (strings.Contains(sqlstr, "'")) {
		return "", errors.New(`->finder-->GetSQL()SQL语句请不要直接拼接字符串参数,容易注入!!!请使用问号占位符,例如 finder.Append("and id=?","stringId"),如果必须拼接字符串,请设置 finder.InjectionCheck = false `)
	}
	finder.sqlstr = sqlstr
	return sqlstr, nil
}
