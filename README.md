## 介绍
go(golang)轻量级ORM,零依赖,支持达梦(dm),金仓(kingbase),神通(shentong),南大通用(gbase),mysql,postgresql,oracle,mssql,sqlite,clickhouse数据库.  
源码地址:https://gitee.com/chunanyong/zorm  
作者博客:[https://www.jiagou.com](https://www.jiagou.com)  

``` 
go get gitee.com/chunanyong/zorm 
```  
* 基于原生sql语句编写,是[springrain](https://gitee.com/chunanyong/springrain)的精简和优化.
* [代码生成器](https://gitee.com/zhou-a-xing/wsgt)  
* 代码精简,主体2500行,零依赖4000行,注释详细,方便定制修改.  
* <font color=red>支持事务传播,这是zorm诞生的主要原因</font>
* 支持mysql,postgresql,oracle,mssql,sqlite,clickhouse,dm(达梦),kingbase(金仓),shentong(神通),gbase(南通)
* 支持多库和读写分离
* 更新性能zorm,gorm,xorm相当. 读取性能zorm比gorm,xorm快一倍
* 不支持联合主键,变通认为无主键,业务控制实现(艰难取舍)  
* 支持使用seata-golang实现分布式事务,支持全局托管模式,不需要proxy代理,不修改业务代码,实现分布式事务    
* 支持clickhouse,更新,删除语句使用SQL92标准语法.clickhouse-go官方驱动不支持批量insert语法,建议使用https://github.com/mailru/go-clickhouse

zorm生产环境使用参考: [UserStructService.go](https://gitee.com/chunanyong/readygo/tree/master/permission/permservice)  

## 支持国产数据库  
### 达梦(dm)  
配置zorm.DataSourceConfig的 DriverName:dm ,DBType:dm  
达梦数据库驱动: [https://gitee.com/chunanyong/dm](https://gitee.com/chunanyong/dm)  
达梦的text类型会映射为dm.DmClob,string不能接收,需要实现zorm.CustomDriverValueConver接口,自定义扩展处理  

### 人大金仓(kingbase)  
配置zorm.DataSourceConfig的 DriverName:kingbase ,DBType:kingbase    
金仓驱动说明: [https://help.kingbase.com.cn/doc-view-8108.html](https://help.kingbase.com.cn/doc-view-8108.html)  
金仓kingbase 8核心是基于postgresql 9.6,可以使用 [https://github.com/lib/pq](https://github.com/lib/pq) 进行测试,生产环境建议使用官方驱动.    
注意修改 data/kingbase.conf中 ```ora_input_emptystr_isnull = false```,因为golang没有null值,一般数据库都是not null,golang的string默认是'',如果这个设置为true,数据库就会把值设置为null,和字段属性not null 冲突,因此报错.   

### 神州通用(shentong)  
建议使用官方驱动,配置zorm.DataSourceConfig的 DriverName:aci ,DBType:shentong  

### 南大通用(gbase)
~~暂时还未找到官方golang驱动,配置zorm.DataSourceConfig的 DriverName:gbase ,DBType:gbase~~  
暂时先使用odbc驱动,DriverName:odbc ,DBType:gbase



## 测试用例  
https://gitee.com/chunanyong/readygo/blob/master/test/testzorm/BaseDao_test.go  

```go  
// zorm 使用原生的sql语句,没有对sql语法做限制.语句使用Finder作为载体
// 占位符统一使用?,zorm会根据数据库类型,自动替换占位符,例如postgresql数据库把?替换成$1,$2...
// zorm使用 ctx context.Context 参数实现事务传播,ctx从web层传递进来即可,例如gin的c.Request.Context()
// zorm的事务操作需要显示使用zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {})开启
``` 



## 数据库脚本和实体类
https://gitee.com/chunanyong/readygo/blob/master/test/testzorm/demoStruct.go

生成实体类或手动编写,建议使用代码生成器 https://gitee.com/zhou-a-xing/wsgt
```go 

package testzorm

import (
	"time"

	"gitee.com/chunanyong/zorm"
)

//建表语句

/*

DROP TABLE IF EXISTS `t_demo`;
CREATE TABLE `t_demo`  (
  `id` varchar(50)  NOT NULL COMMENT '主键',
  `userName` varchar(30)  NOT NULL COMMENT '姓名',
  `password` varchar(50)  NOT NULL COMMENT '密码',
  `createTime` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
  `active` int  COMMENT '是否有效(0否,1是)',
  PRIMARY KEY (`id`)
) ENGINE = InnoDB CHARACTER SET = utf8mb4  COMMENT = '例子' ;

*/

//demoStructTableName 表名常量,方便直接调用
const demoStructTableName = "t_demo"

// demoStruct 例子
type demoStruct struct {
	//引入默认的struct,隔离IEntityStruct的方法改动
	zorm.EntityStruct

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
		//如果Id=="",保存时zorm会调用zorm.FuncGenerateStringID(),默认UUID字符串,也可以自己定义实现方式,例如 zorm.FuncGenerateStringID=funcmyId
		Id:         zorm.FuncGenerateStringID(),
		UserName:   "defaultUserName",
		Password:   "defaultPassword",
		Active:     1,
		CreateTime: time.Now(),
	}
	return demo
}


```

## 测试用例即文档

```go  

// testzorm 使用原生的sql语句,没有对sql语法做限制.语句使用Finder作为载体
// 占位符统一使用?,zorm会根据数据库类型,自动替换占位符,例如postgresql数据库把?替换成$1,$2...
// zorm使用 ctx context.Context 参数实现事务传播,ctx从web层传递进来即可,例如gin的c.Request.Context()
// zorm的事务操作需要显示使用zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {})开启
package testzorm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gitee.com/chunanyong/zorm"

	//00.引入数据库驱动
	_ "github.com/go-sql-driver/mysql"
)

//dbDao 代表一个数据库,如果有多个数据库,就对应声明多个DBDao
var dbDao *zorm.DBDao

// ctx默认应该有 web层传入,例如gin的c.Request.Context().这里只是模拟
var ctx = context.Background()

//01.初始化DBDao
func init() {

	//自定义zorm日志输出
	//zorm.LogCallDepth = 4 //日志调用的层级
	//zorm.FuncLogError = myFuncLogError //记录异常日志的函数
	//zorm.FuncLogPanic = myFuncLogPanic //记录panic日志,默认使用ZormErrorLog实现
	//zorm.FuncPrintSQL = myFuncPrintSQL //打印sql的函数

	//自定义日志输出格式,把FuncPrintSQL函数重新赋值
	//log.SetFlags(log.LstdFlags)
	//zorm.FuncPrintSQL = zorm.FuncPrintSQL

	//dbDaoConfig 数据库的配置.这里只是模拟,生产应该是读取配置配置文件,构造DataSourceConfig
	dbDaoConfig := zorm.DataSourceConfig{
		//DSN 数据库的连接字符串
		DSN: "root:root@tcp(127.0.0.1:3306)/readygo?charset=utf8&parseTime=true",
		//数据库驱动名称:mysql,postgres,oci8,sqlserver,sqlite3,clickhouse,dm,kingbase,aci 和DBType对应,处理数据库有多个驱动
		DriverName: "mysql",
		//数据库类型(方言判断依据):mysql,postgresql,oracle,mssql,sqlite,clickhouse,dm,kingbase,shentong 和 DriverName 对应,处理数据库有多个驱动
		DBType: "mysql",
		//MaxOpenConns 数据库最大连接数 默认50
		MaxOpenConns: 50,
		//MaxIdleConns 数据库最大空闲连接数 默认50
		MaxIdleConns: 50,
		//ConnMaxLifetimeSecond 连接存活秒时间. 默认600(10分钟)后连接被销毁重建.避免数据库主动断开连接,造成死连接.MySQL默认wait_timeout 28800秒(8小时)
		ConnMaxLifetimeSecond: 600,
		//PrintSQL 打印SQL.会使用FuncPrintSQL记录SQL
		PrintSQL: true,
		//DefaultTxOptions 事务隔离级别的默认配置,默认为nil
		//DefaultTxOptions: nil,
		//DefaultTxOptions: &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false},

		//FuncSeataGlobalTransaction seata-golang分布式的适配函数,返回ISeataGlobalTransaction接口的实现
	    FuncSeataGlobalTransaction func(ctx context.Context) (ISeataGlobalTransaction, context.Context, error)
	}

	// 根据dbDaoConfig创建dbDao, 一个数据库只执行一次,第一个执行的数据库为 defaultDao,后续zorm.xxx方法,默认使用的就是defaultDao
	dbDao, _ = zorm.NewDBDao(&dbDaoConfig)
}

//TestInsert 02.测试保存Struct对象
func TestInsert(t *testing.T) {

	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚
	//如果全局DefaultTxOptions配置不满足需求,可以在zorm.Transaction事务方法前设置事务的隔离级别,
	//例如 ctx, _ := dbDao.BindContextTxOptions(ctx, &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}),如果txOptions为nil,使用全局DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		//创建一个demo对象
		demo := newDemoStruct()

		//保存对象,参数是对象指针.如果主键是自增,会赋值到对象的主键属性
		_, err := zorm.Insert(ctx, &demo)

		//如果返回的err不是nil,事务就会回滚
		return nil, err
	})
	//标记测试失败
	if err != nil {
		t.Errorf("错误:%v", err)
	}
}

//TestInsertSlice 03.测试批量保存Struct对象的Slice
//如果是自增主键,无法对Struct对象里的主键属性赋值
func TestInsertSlice(t *testing.T) {

	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚
	//如果全局DefaultTxOptions配置不满足需求,可以在zorm.Transaction事务方法前设置事务的隔离级别,
	//例如 ctx, _ := dbDao.BindContextTxOptions(ctx, &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}),如果txOptions为nil,使用全局DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

		//slice存放的类型是zorm.IEntityStruct!!!,golang目前没有泛型,使用IEntityStruct接口,兼容Struct实体类
		demoSlice := make([]zorm.IEntityStruct, 0)

		//创建对象1
		demo1 := newDemoStruct()
		demo1.UserName = "demo1"
		//创建对象2
		demo2 := newDemoStruct()
		demo2.UserName = "demo2"

		demoSlice = append(demoSlice, &demo1, &demo2)

		//批量保存对象,如果主键是自增,无法保存自增的ID到对象里.
		_, err := zorm.InsertSlice(ctx, demoSlice)

		//如果返回的err不是nil,事务就会回滚
		return nil, err
	})
	//标记测试失败
	if err != nil {
		t.Errorf("错误:%v", err)
	}
}

//TestInsertEntityMap 04.测试保存EntityMap对象,用于不方便使用struct的场景,使用Map作为载体
func TestInsertEntityMap(t *testing.T) {

	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚
	//如果全局DefaultTxOptions配置不满足需求,可以在zorm.Transaction事务方法前设置事务的隔离级别,
	//例如 ctx, _ := dbDao.BindContextTxOptions(ctx, &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}),如果txOptions为nil,使用全局DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		//创建一个EntityMap,需要传入表名
		entityMap := zorm.NewEntityMap(demoStructTableName)
		//设置主键名称
		entityMap.PkColumnName = "id"
		//如果是自增序列,设置序列的值
		//entityMap.PkSequence = "mySequence"

		//Set 设置数据库的字段值
		//如果主键是自增或者序列,不要entityMap.Set主键的值
		entityMap.Set("id", zorm.FuncGenerateStringID())
		entityMap.Set("userName", "entityMap-userName")
		entityMap.Set("password", "entityMap-password")
		entityMap.Set("createTime", time.Now())
		entityMap.Set("active", 1)

		//执行
		_, err := zorm.InsertEntityMap(ctx, entityMap)

		//如果返回的err不是nil,事务就会回滚
		return nil, err
	})
	//标记测试失败
	if err != nil {
		t.Errorf("错误:%v", err)
	}
}

//TestQueryRow 05.测试查询一个struct对象
func TestQueryRow(t *testing.T) {

	//声明一个对象的指针,用于承载返回的数据
	demo := &demoStruct{}

	//构造查询用的finder
	finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo
	//finder = zorm.NewSelectFinder(demoStructTableName, "id,userName") // select id,userName from t_demo
	//finder = zorm.NewFinder().Append("SELECT * FROM " + demoStructTableName) // select * from t_demo

	//finder.Append 第一个参数是语句,后面的参数是对应的值,值的顺序要正确.语句统一使用?,zorm会处理数据库的差异
	finder.Append("WHERE id=? and active in(?)", "20210630163227149563000042432429", []int{0, 1})

	//执行查询
	_, err := zorm.QueryRow(ctx, finder, demo)

	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}
	//打印结果
	fmt.Println(demo)
}

//TestQueryRowMap 06.测试查询map接收结果,用于不太适合struct的场景,比较灵活
func TestQueryRowMap(t *testing.T) {

	//构造查询用的finder
	finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo
	//finder.Append 第一个参数是语句,后面的参数是对应的值,值的顺序要正确.语句统一使用?,zorm会处理数据库的差异
	finder.Append("WHERE id=? and active in(?)", "20210630163227149563000042432429", []int{0, 1})
	//执行查询
	resultMap, err := zorm.QueryRowMap(ctx, finder)

	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}
	//打印结果
	fmt.Println(resultMap)
}

//TestQuery 07.测试查询对象列表
func TestQuery(t *testing.T) {
	//创建用于接收结果的slice
	list := make([]*demoStruct, 0)

	//构造查询用的finder
	finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo
	//创建分页对象,查询完成后,page对象可以直接给前端分页组件使用
	page := zorm.NewPage()
	page.PageNo = 2   //查询第1页,默认是1
	page.PageSize = 2 //每页20条,默认是20

	//执行查询
	err := zorm.Query(ctx, finder, &list, page)
	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}
	//打印结果
	fmt.Println("总条数:", page.TotalCount, "  列表:", list)
}

//TestQueryMap 08.测试查询map列表,用于不方便使用struct的场景,一条记录是一个map对象
func TestQueryMap(t *testing.T) {
	//构造查询用的finder
	finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo

	//创建分页对象,查询完成后,page对象可以直接给前端分页组件使用
	page := zorm.NewPage()
	page.PageNo = 1   //查询第1页,默认是1
	page.PageSize = 2 //每页20条,默认是20
	//执行查询
	listMap, err := zorm.QueryMap(ctx, finder, page)
	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}
	//打印结果
	fmt.Println("总条数:", page.TotalCount, "  列表:", listMap)
}

//TestUpdateNotZeroValue 09.更新struct对象,只更新不为零值的字段.主键必须有值
func TestUpdateNotZeroValue(t *testing.T) {

	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚
	//如果全局DefaultTxOptions配置不满足需求,可以在zorm.Transaction事务方法前设置事务的隔离级别,
	//例如 ctx, _ := dbDao.BindContextTxOptions(ctx, &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}),如果txOptions为nil,使用全局DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		//声明一个对象的指针,用于更新数据
		demo := &demoStruct{}
		demo.Id = "20210630163227149563000042432429"
		demo.UserName = "UpdateNotZeroValue"

		//更新 "sql":"UPDATE t_demo SET userName=? WHERE id=?","args":["UpdateNotZeroValue","41b2aa4f-379a-4319-8af9-08472b6e514e"]
		_, err := zorm.UpdateNotZeroValue(ctx, demo)

		//如果返回的err不是nil,事务就会回滚
		return nil, err
	})
	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}

}

//TestUpdate 10.更新struct对象,更新所有字段.主键必须有值
func TestUpdate(t *testing.T) {

	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚
	//如果全局DefaultTxOptions配置不满足需求,可以在zorm.Transaction事务方法前设置事务的隔离级别,
	//例如 ctx, _ := dbDao.BindContextTxOptions(ctx, &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}),如果txOptions为nil,使用全局DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

		//声明一个对象的指针,用于更新数据
		demo := &demoStruct{}
		demo.Id = "20210630163227149563000042432429"
		demo.UserName = "TestUpdate"

		_, err := zorm.Update(ctx, demo)

		//如果返回的err不是nil,事务就会回滚
		return nil, err
	})
	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}
}

//TestUpdateFinder 11.通过finder更新,zorm最灵活的方式,可以编写任何更新语句,甚至手动编写insert语句
func TestUpdateFinder(t *testing.T) {
	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚
	//如果全局DefaultTxOptions配置不满足需求,可以在zorm.Transaction事务方法前设置事务的隔离级别,
	//例如 ctx, _ := dbDao.BindContextTxOptions(ctx, &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}),如果txOptions为nil,使用全局DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		finder := zorm.NewUpdateFinder(demoStructTableName) // UPDATE t_demo SET
		//finder = zorm.NewDeleteFinder(demoStructTableName)  // DELETE FROM t_demo
		//finder = zorm.NewFinder().Append("UPDATE").Append(demoStructTableName).Append("SET") // UPDATE t_demo SET
		finder.Append("userName=?,active=?", "TestUpdateFinder", 1).Append("WHERE id=?", "20210630163227149563000042432429")

		//更新 "sql":"UPDATE t_demo SET  userName=?,active=? WHERE id=?","args":["TestUpdateFinder",1,"41b2aa4f-379a-4319-8af9-08472b6e514e"]
		_, err := zorm.UpdateFinder(ctx, finder)

		//如果返回的err不是nil,事务就会回滚
		return nil, err
	})
	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}

}

//TestUpdateEntityMap 12.更新一个EntityMap,主键必须有值
func TestUpdateEntityMap(t *testing.T) {
	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚
	//如果全局DefaultTxOptions配置不满足需求,可以在zorm.Transaction事务方法前设置事务的隔离级别,
	//例如 ctx, _ := dbDao.BindContextTxOptions(ctx, &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}),如果txOptions为nil,使用全局DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		//创建一个EntityMap,需要传入表名
		entityMap := zorm.NewEntityMap(demoStructTableName)
		//设置主键名称
		entityMap.PkColumnName = "id"
		//Set 设置数据库的字段值,主键必须有值
		entityMap.Set("id", "20210630163227149563000042432429")
		entityMap.Set("userName", "TestUpdateEntityMap")
		//更新 "sql":"UPDATE t_demo SET userName=? WHERE id=?","args":["TestUpdateEntityMap","41b2aa4f-379a-4319-8af9-08472b6e514e"]
		_, err := zorm.UpdateEntityMap(ctx, entityMap)

		//如果返回的err不是nil,事务就会回滚
		return nil, err
	})
	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}

}

//TestDelete 13.删除一个struct对象,主键必须有值
func TestDelete(t *testing.T) {
	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚
	//如果全局DefaultTxOptions配置不满足需求,可以在zorm.Transaction事务方法前设置事务的隔离级别,
	//例如 ctx, _ := dbDao.BindContextTxOptions(ctx, &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}),如果txOptions为nil,使用全局DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		demo := &demoStruct{}
		demo.Id = "20210630163227149563000042432429"

		//删除 "sql":"DELETE FROM t_demo WHERE id=?","args":["ae9987ac-0467-4fe2-a260-516c89292684"]
		_, err := zorm.Delete(ctx, demo)

		//如果返回的err不是nil,事务就会回滚
		return nil, err
	})
	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}

}

//TestProc 14.测试调用存储过程
func TestProc(t *testing.T) {
	demo := &demoStruct{}
	finder := zorm.NewFinder().Append("call testproc(?) ", "u_10001")
	zorm.QueryRow(ctx, finder, demo)
	fmt.Println(demo)
}

//TestFunc 15.测试调用自定义函数
func TestFunc(t *testing.T) {
	userName := ""
	finder := zorm.NewFinder().Append("select testfunc(?) ", "u_10001")
	zorm.QueryRow(ctx, finder, &userName)
	fmt.Println(userName)
}

//TestOther 16.其他的一些说明.非常感谢您能看到这一行
func TestOther(t *testing.T) {

	//场景1.多个数据库.通过对应数据库的dbDao,调用BindContextDBConnection函数,把这个数据库的连接绑定到返回的ctx上,然后把ctx传递到zorm的函数即可.
	newCtx, err := dbDao.BindContextDBConnection(ctx)
	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}

	finder := zorm.NewSelectFinder(demoStructTableName)
	//把新产生的newCtx传递到zorm的函数
	list, _ := zorm.QueryMap(newCtx, finder, nil)
	fmt.Println(list)

	//场景2.单个数据库的读写分离.设置读写分离的策略函数.
	zorm.FuncReadWriteStrategy = myReadWriteStrategy

	//场景3.如果是多个数据库,每个数据库还读写分离,按照 场景1 处理

}

//单个数据库的读写分离的策略 rwType=0 read,rwType=1 write
func myReadWriteStrategy(rwType int) *zorm.DBDao {
	//根据自己的业务场景,返回需要的读写dao,每次需要数据库的连接的时候,会调用这个函数
	return dbDao
}

//---------------------------------//

//实现CustomDriverValueConver接口,扩展自定义类型,例如 达梦数据库text类型,映射出来的是dm.DmClob类型,无法使用string类型直接接收
type CustomDMText struct{}
//GetDriverValue 根据数据库列类型,实体类属性类型,Finder对象,返回driver.Value的实例
//如果无法获取到structFieldType,例如Map查询,会传入nil
//如果返回值为nil,接口扩展逻辑无效,使用原生的方式接收数据库字段值
func (dmtext CustomDMText) GetDriverValue(columnType *sql.ColumnType, structFieldType reflect.Type, finder *zorm.Finder) (driver.Value, error) {
	return &dm.DmClob{}, nil
}
//ConverDriverValue 数据库列类型,实体类属性类型,GetDriverValue返回的driver.Value的临时接收值,Finder对象
//如果无法获取到structFieldType,例如Map查询,会传入nil
//返回符合接收类型值的指针,指针,指针!!!!
func (dmtext CustomDMText) ConverDriverValue(columnType *sql.ColumnType, structFieldType reflect.Type, tempDriverValue driver.Value, finder *zorm.Finder) (interface{}, error) {
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

```  
##  分布式事务
基于seata-golang实现分布式事务. 
### proxy模式 
```golang
//DataSourceConfig 配置  DefaultTxOptions
//DefaultTxOptions: &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false},

//注册mysql驱动
import (
	"github.com/opentrx/mysql/v2"
	"github.com/opentrx/seata-golang/v2/pkg/client"
	"github.com/opentrx/seata-golang/v2/pkg/client/config"
	"github.com/opentrx/seata-golang/v2/pkg/client/rm"
	"github.com/opentrx/seata-golang/v2/pkg/client/tm"
	"github.com/opentrx/seata-golang/v2/pkg/util/log"
	seataContext "github.com/opentrx/seata-golang/v2/pkg/client/base/context"
)

//读取配置文件
configPath := os.Getenv("ConfigPath")
conf := config.InitConfiguration(configPath)
//设置日志
log.Init(conf.Log.LogPath, conf.Log.LogLevel)
//初始化配置
client.Init(conf)
//注册mysql驱动
rm.RegisterTransactionServiceServer(mysql.GetDataSourceManager())
mysql.RegisterResource(config.GetATConfig().DSN)

//后续正常初始化zorm

//tm注册事务服务,参照官方例子
tm.Implement(svc.ProxySvc)

//................//

//ctx传递事务 XID

//通过seataContext获取XID
rootContext := ctx.(*seataContext.RootContext)

//获取XID.可以通过gin的header传递,或者其他方式传递
 xid:=rootContext.GetXID()

// xid绑定到ctx,后续正常使用zorm传递ctx即可
ctx =context.WithValue(ctx,mysql.XID,xid)
```

### 全局托管模式

```golang

//tm注册事务服务,参照官方例子. 不使用proxy模式,全局托管分布式事务
//tm.Implement(svc.ProxySvc)


// ZormSeataGlobalTransaction 包装seata的*tm.DefaultGlobalTransaction,实现zorm.ISeataGlobalTransaction接口
type ZormSeataGlobalTransaction struct {
	*tm.DefaultGlobalTransaction
}

// zorm.FuncSeataGlobalTransaction zorm的全局事务函数,配置zorm.DataSourceConfig.FuncSeataGlobalTransaction=MyFuncSeataGlobalTransaction
func MyFuncSeataGlobalTransaction(ctx context.Context) (zorm.ISeataGlobalTransaction, context.Context, error) {
	//获取seata的rootContext
	rootContext := seataContext.NewRootContext(ctx)
	//创建seata事务
	seataTx := tm.GetCurrentOrCreate(rootContext)
	//使用zorm.ISeataGlobalTransaction接口对象包装seata事务,隔离seata-golang依赖
	seataGlobalTransaction := ZormSeataGlobalTransaction{seataTx}

	return seataGlobalTransaction, rootContext, nil
}

//实现zorm.ISeataGlobalTransaction接口
func (gtx ZormSeataGlobalTransaction) SeataBegin(ctx context.Context) error {
	rootContext := ctx.(*seataContext.RootContext)
	return gtx.BeginWithTimeout(int32(6000), rootContext)
}

func (gtx ZormSeataGlobalTransaction) SeataCommit(ctx context.Context) error {
	rootContext := ctx.(*seataContext.RootContext)
	return gtx.Commit(rootContext)
}

func (gtx ZormSeataGlobalTransaction) SeataRollback(ctx context.Context) error {
	rootContext := ctx.(*seataContext.RootContext)
	return gtx.SeataRollback(rootContext)
}

func (gtx ZormSeataGlobalTransaction) SeataTransactionXID(ctx context.Context) string {
	rootContext := ctx.(*seataContext.RootContext)
	return rootContext.GetXID()
}
```

##  性能压测

   测试代码:https://github.com/alphayan/goormbenchmark

   指标说明
   总时间,平均每次纳秒数,平均每次分配的内存,平均每次分配内存次数

   更新性能zorm,gorm,xorm相当.读取性能zorm比gorm,xorm快一倍  

```
2000 times - Insert
      zorm:     9.05s      4524909 ns/op    2146 B/op     33 allocs/op
      gorm:     9.60s      4800617 ns/op    5407 B/op    119 allocs/op
      xorm:    12.63s      6315205 ns/op    2365 B/op     56 allocs/op

    2000 times - BulkInsert 100 row
      xorm:    23.89s     11945333 ns/op  253812 B/op   4250 allocs/op
      gorm:     Don't support bulk insert - https://github.com/jinzhu/gorm/issues/255
      zorm:     Don't support bulk insert

    2000 times - Update
      xorm:     0.39s       195846 ns/op    2529 B/op     87 allocs/op
      zorm:     0.51s       253577 ns/op    2232 B/op     32 allocs/op
      gorm:     0.73s       366905 ns/op    9157 B/op    226 allocs/op

  2000 times - Read
      zorm:     0.28s       141890 ns/op    1616 B/op     43 allocs/op
      gorm:     0.45s       223720 ns/op    5931 B/op    138 allocs/op
      xorm:     0.55s       276055 ns/op    8648 B/op    227 allocs/op

  2000 times - MultiRead limit 1000
      zorm:    13.93s      6967146 ns/op  694286 B/op  23054 allocs/op
      gorm:    26.40s     13201878 ns/op 2392826 B/op  57031 allocs/op
      xorm:    30.77s     15382967 ns/op 1637098 B/op  72088 allocs/op
```

