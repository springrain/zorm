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
	"reflect"
	"strings"
)

// customDriverValueMap 用于配置数据库字段类型的处理关系,key是 Dialect.字段类型,例如 dm.TEXT
var customDriverValueMap = make(map[string]ICustomDriverValueConver)

// iscdvm 是否有自定义的DriverValueMap
var iscdvm bool

// ICustomDriverValueConver 自定义类型转化接口,用于解决 类似达梦 text --> dm.DmClob --> string类型接收的问题
type ICustomDriverValueConver interface {
	// GetDriverValue 根据数据库列类型,返回driver.Value的实例,struct属性类型
	// map接收或者字段不存在,无法获取到structFieldType,会传入nil
	GetDriverValue(ctx context.Context, columnType *sql.ColumnType, structFieldType *reflect.Type) (driver.Value, error)

	// ConverDriverValue 数据库列类型,GetDriverValue返回的driver.Value的临时接收值,struct属性类型
	// map接收或者字段不存在,无法获取到structFieldType,会传入nil
	// 返回符合接收类型值的指针,指针,指针!!!!
	ConverDriverValue(ctx context.Context, columnType *sql.ColumnType, tempDriverValue driver.Value, structFieldType *reflect.Type) (interface{}, error)
}

// RegisterCustomDriverValueConver 注册自定义的字段处理逻辑,用于驱动无法直接转换的场景,例如达梦的 TEXT 无法直接转化成 string
// dialectColumnType 值是 Dialect.字段类型,例如: dm.TEXT
// 一般是放到init方法里进行注册
func RegisterCustomDriverValueConver(dialectColumnType string, customDriverValueConver ICustomDriverValueConver) error {
	if len(dialectColumnType) < 1 {
		return errors.New("->RegisterCustomDriverValueConver-->dialectColumnType为空")
	}
	dialectColumnTypes := strings.Split(dialectColumnType, ".")
	if len(dialectColumnTypes) < 2 {
		customDriverValueMap[strings.ToUpper(dialectColumnType)] = customDriverValueConver
		err := errors.New("->RegisterCustomDriverValueConver-->dialectColumnType 值是 Dialect.字段类型,例如: dm.TEXT ,本次正常运行,请尽快修改")
		FuncLogError(nil, err)
	} else {
		customDriverValueMap[strings.ToLower(dialectColumnTypes[0])+"."+strings.ToUpper(dialectColumnTypes[1])] = customDriverValueConver
	}
	iscdvm = true
	return nil
}

type driverValueInfo struct {
	customDriverValueConver ICustomDriverValueConver
	columnType              *sql.ColumnType
	tempDriverValue         interface{}
	structFieldType         *reflect.Type
}

/**

import (
	// 00.引入数据库驱动
	"gitee.com/chunanyong/dm"
	"io"
)

// CustomDMText 实现ICustomDriverValueConver接口,扩展自定义类型,例如 达梦数据库TEXT类型,映射出来的是dm.DmClob类型,无法使用string类型直接接收
type CustomDMText struct{}

// GetDriverValue 根据数据库列类型,返回driver.Value的实例,struct属性类型
// map接收或者字段不存在,无法获取到structFieldType,会传入nil
func (dmtext CustomDMText) GetDriverValue(ctx context.Context, columnType *sql.ColumnType, structFieldType *reflect.Type) (driver.Value, error) {
	// 如果需要使用structFieldType,需要先判断是否为nil
	// if structFieldType != nil {
	// }

	return &dm.DmClob{}, nil
}

// ConverDriverValue 数据库列类型,GetDriverValue返回的driver.Value的临时接收值,struct属性类型
// map接收或者字段不存在,无法获取到structFieldType,会传入nil
// 返回符合接收类型值的指针,指针,指针!!!!
func (dmtext CustomDMText) ConverDriverValue(ctx context.Context, columnType *sql.ColumnType, tempDriverValue driver.Value, structFieldType *reflect.Type) (interface{}, error) {
	// 如果需要使用structFieldType,需要先判断是否为nil
	// if structFieldType != nil {
	// }

	// 类型转换
	dmClob, isok := tempDriverValue.(*dm.DmClob)
	if !isok {
		return tempDriverValue, errors.New("->ConverDriverValue-->转换至*dm.DmClob类型失败")
	}
	if dmClob == nil || !dmClob.Valid {
		return new(string), nil
	}
	// 获取长度
	dmlen, errLength := dmClob.GetLength()
	if errLength != nil {
		return dmClob, errLength
	}

	// int64转成int类型
	strInt64 := strconv.FormatInt(dmlen, 10)
	dmlenInt, errAtoi := strconv.Atoi(strInt64)
	if errAtoi != nil {
		return dmClob, errAtoi
	}

	// 读取字符串
	str, errReadString := dmClob.ReadString(1, dmlenInt)

	// 处理空字符串或NULL造成的EOF错误
	if errReadString == io.EOF {
		return new(string), nil
	}

	return &str, errReadString
}
// RegisterCustomDriverValueConver 注册自定义的字段处理逻辑,用于驱动无法直接转换的场景,例如达梦的 TEXT 无法直接转化成 string
// 一般是放到init方法里进行注册
func init() {
	// dialectColumnType 值是 Dialect.字段类型 ,例如 dm.TEXT
    zorm.RegisterCustomDriverValueConver("dm.TEXT", CustomDMText{})
}

**/
