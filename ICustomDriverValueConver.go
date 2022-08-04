package zorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"reflect"
)

//CustomDriverValueMap 用于配置driver.Value和对应的处理关系,key是 drier.Value 的字符串,例如 *dm.DmClob
//一般是放到init方法里进行添加
var CustomDriverValueMap = make(map[string]ICustomDriverValueConver)

//ICustomDriverValueConver 自定义类型转化接口,用于解决 类似达梦 text --> dm.DmClob --> string类型接收的问题
type ICustomDriverValueConver interface {
	//GetDriverValue 根据数据库列类型,实体类属性类型,Finder对象,返回driver.Value的实例
	//如果无法获取到structFieldType,例如Map查询,会传入nil
	//如果返回值为nil,接口扩展逻辑无效,使用原生的方式接收数据库字段值
	GetDriverValue(ctx context.Context, columnType *sql.ColumnType, structFieldType *reflect.Type, finder *Finder) (driver.Value, error)

	//ConverDriverValue 数据库列类型,实体类属性类型,GetDriverValue返回的driver.Value的临时接收值,Finder对象
	//如果无法获取到structFieldType,例如Map查询,会传入nil
	//返回符合接收类型值的指针,指针,指针!!!!
	ConverDriverValue(ctx context.Context, columnType *sql.ColumnType, structFieldType *reflect.Type, tempDriverValue driver.Value, finder *Finder) (interface{}, error)
}
type driverValueInfo struct {
	converFunc      ICustomDriverValueConver
	columnType      *sql.ColumnType
	tempDriverValue interface{}
}

/**

//实现CustomDriverValueConver接口,扩展自定义类型,例如 达梦数据库text类型,映射出来的是dm.DmClob类型,无法使用string类型直接接收
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
		return tempDriverValue, errors.New("转换至*dm.DmClob类型失败")
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
//CustomDriverValueMap 用于配置driver.Value和对应的处理关系,key是 drier.Value 的字符串,例如 *dm.DmClob
//一般是放到init方法里进行添加
zorm.CustomDriverValueMap["*dm.DmClob"] = CustomDMText{}

**/
