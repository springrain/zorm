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
	"strings"
)

//customDriverValueMap 用于配置数据库字段类型的处理关系,key是数据库字段类型的字符串,例如 TEXT
var customDriverValueMap = make(map[string]ICustomDriverValueConver)
var cdvmLen int
var iscdvm bool

//ICustomDriverValueConver 自定义类型转化接口,用于解决 类似达梦 text --> dm.DmClob --> string类型接收的问题
type ICustomDriverValueConver interface {
	//GetDriverValue 根据数据库列类型,实体类属性类型,Finder对象,返回driver.Value的实例
	//如果无法获取到structFieldType,例如Map查询,会传入nil
	GetDriverValue(ctx context.Context, columnType *sql.ColumnType) (driver.Value, error)

	//ConverDriverValue 数据库列类型,实体类属性类型,GetDriverValue返回的driver.Value的临时接收值,Finder对象
	//如果无法获取到structFieldType,例如Map查询,会传入nil
	//返回符合接收类型值的指针,指针,指针!!!!
	ConverDriverValue(ctx context.Context, columnType *sql.ColumnType, tempDriverValue driver.Value) (interface{}, error)
}
type driverValueInfo struct {
	customDriverValueConver ICustomDriverValueConver
	columnType              *sql.ColumnType
	tempDriverValue         interface{}
}

//RegisterCustomDriverValueConver 注册自定义的字段处理逻辑,用于驱动无法直接转换的场景,例如达梦的 TEXT 无法直接转化成 string
//一般是放到init方法里进行注册
func RegisterCustomDriverValueConver(columnType string, customDriverValueConver ICustomDriverValueConver) error {
	if len(columnType) < 1 {
		return errors.New("->RegisterCustomDriverValueConver-->columnType为空")
	}
	customDriverValueMap[strings.ToUpper(columnType)] = customDriverValueConver
	cdvmLen = len(customDriverValueMap)

	iscdvm = true
	return nil
}

/**

//实现ICustomDriverValueConver接口,扩展自定义类型,例如 达梦数据库text类型,映射出来的是dm.DmClob类型,无法使用string类型直接接收
type CustomDMText struct{}
//GetDriverValue 根据数据库列类型,实体类属性类型,Finder对象,返回driver.Value的实例
//如果无法获取到structFieldType,例如Map查询,会传入nil
//如果返回值为nil,接口扩展逻辑无效,使用原生的方式接收数据库字段值
func (dmtext CustomDMText) GetDriverValue(ctx context.Context, columnType *sql.ColumnType, structFieldType reflect.Type, finder *zorm.Finder) (driver.Value, error) {
	return &dm.DmClob{}, nil
}
//ConverDriverValue 数据库列类型,实体类属性类型,GetDriverValue返回的driver.Value的临时接收值,Finder对象
//如果无法获取到structFieldType,例如Map查询,会传入nil
//返回符合接收类型值的指针,指针,指针!!!!
func (dmtext CustomDMText) ConverDriverValue(ctx context.Context, columnType *sql.ColumnType, structFieldType reflect.Type, tempDriverValue driver.Value, finder *zorm.Finder) (interface{}, error) {
	//类型转换
	dmClob, isok := tempDriverValue.(*dm.DmClob)
	if !isok {
		return tempDriverValue, errors.New("->ConverDriverValue-->转换至*dm.DmClob类型失败")
	}

	//获取长度
	dmlen, errLength := dmClob.GetLength()
	if errLength != nil {
		return dmClob, errLength
	}

	//int64转成int类型
	strInt64 := strconv.FormatInt(dmlen, 10)
	dmlenInt, errAtoi := strconv.Atoi(strInt64)
	if errAtoi != nil {
		return dmClob, errAtoi
	}

	//读取字符串
	str, errReadString := dmClob.ReadString(1, dmlenInt)
	return &str, errReadString
}
//RegisterCustomDriverValueConver 注册自定义的字段处理逻辑,用于驱动无法直接转换的场景,例如达梦的 TEXT 无法直接转化成 string
//一般是放到init方法里进行注册
zorm.RegisterCustomDriverValueConver("TEXT", CustomDMText{})

**/
