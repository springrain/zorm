package zorm

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

//TestInsert 02.测试保存Struct对象
func TestString(t *testing.T) {
	teststr := "456"
	args := make([]interface{}, 0)
	args = append(args, "123")
	args = append(args, &teststr)
	args = append(args, 789)
	sql := "SELECT * from table where id=? and name=? and age=?"
	sqlptr := &sql
	var err error
	sqlptr, err = reTDengineSQL("tdengine2", sqlptr, args)
	fmt.Println(*sqlptr)

	//标记测试失败
	if err != nil {
		t.Errorf("错误:%v", err)
	}
}

func TestPage(t *testing.T) {

	sql := "SELECT * from table where id=? and name=? and age=?"

	sqlptr, err := wrapPageSQL("oracle", sql, NewPage())

	fmt.Println(sqlptr)

	//标记测试失败
	if err != nil {
		t.Errorf("错误:%v", err)
	}
}

func TestInfo(t *testing.T) {

	//获取接收值的对象
	typeOf := reflect.TypeOf(newDemoStruct())
	//获取到类型的字段缓存
	//Get the type field cache
	getDBColumnExportFieldMap(&typeOf)
}

//demoStructTableName 表名常量,方便直接调用
const demoStructTableName = "t_demo"

// demoStruct 例子
type demoStruct struct {
	//引入默认的struct,隔离IEntityStruct的方法改动
	EntityStruct

	//Id 主键
	Id string `column:"id"`

	//UserName 姓名
	UserName string `column:"userName"`

	//Password 密码
	Password string `column:"password"`

	//CreateTime <no value>
	CreateTime time.Time `column:"createTime"`

	//Active 是否有效(0否,1是)
	//Active int `column:"active"`

	//------------------数据库字段结束,自定义字段写在下面---------------//
	//如果查询的字段在column tag中没有找到,就会根据名称(不区分大小写,支持 _ 下划线转驼峰)映射到struct的属性上

	//模拟自定义的字段Active
	Active int
}

//GetTableName 获取表名称
//IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *demoStruct) GetTableName() string {
	return demoStructTableName
}

//GetPKColumnName 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称
//不支持联合主键,变通认为无主键,业务控制实现(艰难取舍)
//如果没有主键,也需要实现这个方法, return "" 即可
//IEntityStruct 接口的方法,实体类需要实现!!!
func (entity *demoStruct) GetPKColumnName() string {
	//如果没有主键
	//return ""
	return "id"
}

//newDemoStruct 创建一个默认对象
func newDemoStruct() demoStruct {
	demo := demoStruct{
		//如果Id=="",保存时zorm会调用zorm.FuncGenerateStringID(),默认时间戳+随机数,也可以自己定义实现方式,例如 zorm.FuncGenerateStringID=funcmyId
		Id:         FuncGenerateStringID(),
		UserName:   "defaultUserName",
		Password:   "defaultPassword",
		Active:     1,
		CreateTime: time.Now(),
	}
	return demo
}
