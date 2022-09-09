## 介绍
![zorm logo](zorm-logo.png)  
Go轻量ORM,零依赖,零侵入分布式事务,支持达梦(dm),金仓(kingbase),神通(shentong),南通(gbase),TDengine,mysql,postgresql,oracle,mssql,sqlite,db2,clickhouse... 

官网: https://zorm.cn  
源码地址: https://gitee.com/chunanyong/zorm        
测试用例: https://gitee.com/wuxiangege/zorm-examples/      

交流QQ群：[727723736]() 添加进入社区群聊,问题交流,技术探讨  
社区微信: [LAUV927]()   

``` 
go get gitee.com/chunanyong/zorm 
```  
* 基于原生sql语句编写,学习成本更低  
* [代码生成器](https://gitee.com/zhou-a-xing/wsgt)    
* 代码精简,主体2500行,零依赖4000行,注释详细,方便定制修改   
* <font color=red>支持事务传播,这是zorm诞生的主要原因</font>
* 支持dm(达梦),kingbase(金仓),shentong(神通),gbase(南通),TDengine,mysql,postgresql,oracle,mssql,sqlite,db2,clickhouse...
* 支持多库和读写分离
* 更新性能zorm,gorm,xorm相当. 读取性能zorm比gorm,xorm快50%
* 不支持联合主键,变通认为无主键,业务控制实现(艰难取舍)  
* 支持seata,hptx,dbpack分布式事务,支持全局事务托管,不修改业务代码,零侵入分布式事务
* 支持clickhouse,更新,删除语句使用SQL92标准语法.clickhouse-go官方驱动不支持批量insert语法,建议使用https://github.com/mailru/go-clickhouse

## 事务传播
事务传播是zorm的核心功能,也是zorm所有方法都有ctx入参的主要原因.    
zorm的事务操作需要显式使用```zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {})```开启,在执行闭包函数前检查事务,如果ctx里有事务就加入事务,如果ctx里没事务就创建新的事务,所以只需要传递同一个ctx对象,就可以实现事务传播.特殊场景如果不想事务同步,就可以声明一个新的ctx对象,做事务隔离.

## 源码仓库说明
我主导的开源项目主库都在gitee,github上留有项目说明,引导跳转到gitee,这样也造成了项目star增长缓慢,毕竟github用户多些.  
**开源没有国界,开发者却有自己的祖国.**   
严格意义上,github是受美国法律管辖的 https://www.infoq.cn/article/SA72SsSeZBpUSH_ZH8XB  
尽我所能,支持国内开源社区,不喜勿喷,谢谢!

## 支持国产数据库  
### 达梦(dm)  
- 配置zorm.DataSourceConfig的 ```DriverName:dm ,Dialect:dm```  
- 达梦数据库驱动: https://gitee.com/chunanyong/dm    
- 达梦的text类型会映射为dm.DmClob,string不能接收,需要实现zorm.ICustomDriverValueConver接口,自定义扩展处理  
- 达梦使用time作为where条件,需要注意时区问题,建议使用字符串代替time, https://eco.dameng.com/community/question/936924eb9861e6d429114e87d6f3a854  

### 人大金仓(kingbase)  
- 配置zorm.DataSourceConfig的 ```DriverName:kingbase ,Dialect:kingbase```    
- 金仓驱动说明: https://help.kingbase.com.cn/doc-view-8108.html    
- 金仓kingbase 8核心是基于postgresql 9.6,可以使用 https://github.com/lib/pq 进行测试,生产环境建议使用官方驱动.    
- 注意修改 data/kingbase.conf中 ```ora_input_emptystr_isnull = false```,因为Go没有null值,一般数据库都是not null,Go的string默认是'',如果这个设置为true,数据库就会把值设置为null,和字段属性not null 冲突,因此报错.   

### 神通(shentong)  
建议使用官方驱动,配置zorm.DataSourceConfig的 ```DriverName:aci ,Dialect:shentong```  

### 南通(gbase)
~~暂时还未找到官方Go驱动,配置zorm.DataSourceConfig的 DriverName:gbase ,Dialect:gbase~~  
暂时先使用odbc驱动,```DriverName:odbc ,Dialect:gbase```

### TDengine  
- 因TDengine驱动不支持事务,需要设置```DisableTransaction=true```
- 配置zorm.DataSourceConfig的 ```DriverName:taosSql或者taosRestful, Dialect:tdengine```  
- 测试用例: https://www.yuque.com/u27016943/nrgi00/dnru3f    

## 数据库脚本和实体类
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
		//如果Id=="",保存时zorm会调用zorm.FuncGenerateStringID(ctx),默认时间戳+随机数,也可以自己定义实现方式,例如 zorm.FuncGenerateStringID=funcmyId
		Id:         zorm.FuncGenerateStringID(ctx),
		UserName:   "defaultUserName",
		Password:   "defaultPassword",
		Active:     1,
		CreateTime: time.Now(),
	}
	return demo
}


```

## 测试用例即文档
测试用例: https://gitee.com/wuxiangege/zorm-examples  

```go  

// testzorm 使用原生的sql语句,没有对sql语法做限制.语句使用Finder作为载体
// 占位符统一使用?,zorm会根据数据库类型,自动替换占位符,例如postgresql数据库把?替换成$1,$2...
// zorm使用 ctx context.Context 参数实现事务传播,ctx从web层传递进来即可,例如gin的c.Request.Context()
// zorm的事务操作需要显式使用zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {})开启
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
	//zorm.FuncLogPanic = myFuncLogPanic //记录panic日志,默认使用defaultLogError实现
	//zorm.FuncPrintSQL = myFuncPrintSQL //打印sql的函数

	//自定义日志输出格式,把FuncPrintSQL函数重新赋值
	//log.SetFlags(log.LstdFlags)
	//zorm.FuncPrintSQL = zorm.FuncPrintSQL

    //自定义主键生成
	//zorm.FuncGenerateStringID=funcmyId

    //Go数据库驱动列表:https://github.com/golang/go/wiki/SQLDrivers

	//dbDaoConfig 数据库的配置.这里只是模拟,生产应该是读取配置配置文件,构造DataSourceConfig
	dbDaoConfig := zorm.DataSourceConfig{
		//DSN 数据库的连接字符串,parseTime=true会自动转换为time格式,默认查询出来的是[]byte数组
		DSN: "root:root@tcp(127.0.0.1:3306)/readygo?charset=utf8&parseTime=true",
		//DriverName 数据库驱动名称:mysql,postgres,oci8,sqlserver,sqlite3,go_ibm_db,clickhouse,dm,kingbase,aci,taosSql|taosRestful 和Dialect对应
		DriverName: "mysql",
		//Dialect 数据库方言:mysql,postgresql,oracle,mssql,sqlite,db2,clickhouse,dm,kingbase,shentong,tdengine 和 DriverName 对应
		Dialect: "mysql",
		//MaxOpenConns 数据库最大连接数 默认50
		MaxOpenConns: 50,
		//MaxIdleConns 数据库最大空闲连接数 默认50
		MaxIdleConns: 50,
		//ConnMaxLifetimeSecond 连接存活秒时间. 默认600(10分钟)后连接被销毁重建.避免数据库主动断开连接,造成死连接.MySQL默认wait_timeout 28800秒(8小时)
		ConnMaxLifetimeSecond: 600,
		//SlowSQLMillis 慢sql的时间阈值,单位毫秒.小于0是禁用SQL语句输出;等于0是只输出SQL语句,不计算执行时间;大于0是计算SQL执行时间,并且>=SlowSQLMillis值
		SlowSQLMillis: 0,
		//DefaultTxOptions 事务隔离级别的默认配置,默认为nil
		//DefaultTxOptions: nil,
		//如果是使用分布式事务,建议使用默认配置
		//DefaultTxOptions: &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false},

		//FuncGlobalTransaction seata/hptx全局分布式事务的适配函数,返回IGlobalTransaction接口的实现
		//业务必须调用 ctx,_=zorm.BindContextEnableGlobalTransaction(ctx) 开启全局分布事务
	    //FuncGlobalTransaction : MyFuncGlobalTransaction,

	    //SQLDB 使用现有的数据库连接,优先级高于DSN
	    //SQLDB : nil,

	    //DisableTransaction 禁用事务,默认false,如果设置了DisableTransaction=true,Transaction方法失效,不再要求有事务,为了处理某些数据库不支持事务,比如TDengine
	    //禁用事务应该有驱动伪造事务API,不应该有orm实现,clickhouse的驱动就是这样做的
	    //DisableTransaction :false,
	}

	// 根据dbDaoConfig创建dbDao, 一个数据库只执行一次,第一个执行的数据库为 defaultDao,后续zorm.xxx方法,默认使用的就是defaultDao
	dbDao, _ = zorm.NewDBDao(&dbDaoConfig)
}

//TestInsert 02.测试保存Struct对象
func TestInsert(t *testing.T) {

	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚.如果设置了DisableTransaction=true,Transaction方法失效,不再要求有事务
    //如果zorm.DataSourceConfig.DefaultTxOptions配置不满足需求,可以在zorm.Transaction事务方法前设置事务的隔离级别
	//例如 ctx, _ := dbDao.BindContextTxOptions(ctx, &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}),如果txOptions为nil,使用zorm.DataSourceConfig.DefaultTxOptions
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

	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚.如果设置了DisableTransaction=true,Transaction方法失效,不再要求有事务
    //如果zorm.DataSourceConfig.DefaultTxOptions配置不满足需求,可以在zorm.Transaction事务方法前设置事务的隔离级别
	//例如 ctx, _ := dbDao.BindContextTxOptions(ctx, &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}),如果txOptions为nil,使用zorm.DataSourceConfig.DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

		//slice存放的类型是zorm.IEntityStruct!!!使用IEntityStruct接口,兼容Struct实体类
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

	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚.如果设置了DisableTransaction=true,Transaction方法失效,不再要求有事务
    //如果zorm.DataSourceConfig.DefaultTxOptions配置不满足需求,可以在zorm.Transaction事务方法前设置事务的隔离级别
	//例如 ctx, _ := dbDao.BindContextTxOptions(ctx, &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}),如果txOptions为nil,使用zorm.DataSourceConfig.DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		//创建一个EntityMap,需要传入表名
		entityMap := zorm.NewEntityMap(demoStructTableName)
		//设置主键名称
		entityMap.PkColumnName = "id"
		//如果是自增序列,设置序列的值
		//entityMap.PkSequence = "mySequence"

		//Set 设置数据库的字段值
		//如果主键是自增或者序列,不要entityMap.Set主键的值
		entityMap.Set("id", zorm.FuncGenerateStringID(ctx))
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

    //只查询一个字段,需要使用这个字段的类型进行接收,目前不支持整个struct对象接收
	//声明一个对象的指针,用于承载返回的数据
	demo := &demoStruct{}

	//构造查询用的finder
	//finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo
	//finder = zorm.NewSelectFinder(demoStructTableName, "id,userName") // select id,userName from t_demo
	finder := zorm.NewFinder().Append("SELECT * FROM " + demoStructTableName) // select * from t_demo
	//finder默认启用了sql注入检查,禁止语句中拼接 ' 单引号,可以设置 finder.InjectionCheck = false 解开限制

	//finder.Append 第一个参数是语句,后面的参数是对应的值,值的顺序要正确.语句统一使用?,zorm会处理数据库的差异
	//in (?) 参数必须有()括号,不能 in ?
	finder.Append("WHERE id=? and active in(?)", "20210630163227149563000042432429", []int{0, 1})

	//执行查询,has为true表示数据库有数据
	has, err := zorm.QueryRow(ctx, finder, demo)

	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}
	//打印结果
	fmt.Println(demo)
}

//TestQueryRowMap 06.测试查询map接收结果,用于不太适合struct的场景,比较灵活
func TestQueryRowMap(t *testing.T) {

	//构造查询用的finder
	//finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo
	finder := zorm.NewFinder().Append("SELECT * FROM " + demoStructTableName) // select * from t_demo
	//finder.Append 第一个参数是语句,后面的参数是对应的值,值的顺序要正确.语句统一使用?,zorm会处理数据库的差异
	//in (?) 参数必须有()括号,不能 in ?
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

	//只查询一个字段,需要使用这个字段的类型进行接收,目前不支持整个struct对象接收
	//创建用于接收结果的slice
	list := make([]*demoStruct, 0)

	//构造查询用的finder
	//finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo
	finder := zorm.NewFinder().Append("SELECT * FROM " + demoStructTableName) // select * from t_demo
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
	//finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo
    finder := zorm.NewFinder().Append("SELECT * FROM " + demoStructTableName) // select * from t_demo
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

	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚.如果设置了DisableTransaction=true,Transaction方法失效,不再要求有事务
    //如果zorm.DataSourceConfig.DefaultTxOptions配置不满足需求,可以在zorm.Transaction事务方法前设置事务的隔离级别
	//例如 ctx, _ := dbDao.BindContextTxOptions(ctx, &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}),如果txOptions为nil,使用zorm.DataSourceConfig.DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		//声明一个对象的指针,用于更新数据
		demo := &demoStruct{}
		demo.Id = "20210630163227149563000042432429"
		demo.UserName = "UpdateNotZeroValue"

		//更新 "sql":"UPDATE t_demo SET userName=? WHERE id=?","args":["UpdateNotZeroValue","20210630163227149563000042432429"]
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

	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚.如果设置了DisableTransaction=true,Transaction方法失效,不再要求有事务
    //如果zorm.DataSourceConfig.DefaultTxOptions配置不满足需求,可以在zorm.Transaction事务方法前设置事务的隔离级别
	//例如 ctx, _ := dbDao.BindContextTxOptions(ctx, &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}),如果txOptions为nil,使用zorm.DataSourceConfig.DefaultTxOptions
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
	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚.如果设置了DisableTransaction=true,Transaction方法失效,不再要求有事务
    //如果zorm.DataSourceConfig.DefaultTxOptions配置不满足需求,可以在zorm.Transaction事务方法前设置事务的隔离级别
	//例如 ctx, _ := dbDao.BindContextTxOptions(ctx, &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}),如果txOptions为nil,使用zorm.DataSourceConfig.DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		//finder := zorm.NewUpdateFinder(demoStructTableName) // UPDATE t_demo SET
		//finder = zorm.NewDeleteFinder(demoStructTableName)  // DELETE FROM t_demo
		finder := zorm.NewFinder().Append("UPDATE").Append(demoStructTableName).Append("SET") // UPDATE t_demo SET
		finder.Append("userName=?,active=?", "TestUpdateFinder", 1).Append("WHERE id=?", "20210630163227149563000042432429")

		//更新 "sql":"UPDATE t_demo SET  userName=?,active=? WHERE id=?","args":["TestUpdateFinder",1,"20210630163227149563000042432429"]
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
	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚.如果设置了DisableTransaction=true,Transaction方法失效,不再要求有事务
    //如果zorm.DataSourceConfig.DefaultTxOptions配置不满足需求,可以在zorm.Transaction事务方法前设置事务的隔离级别
	//例如 ctx, _ := dbDao.BindContextTxOptions(ctx, &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}),如果txOptions为nil,使用zorm.DataSourceConfig.DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		//创建一个EntityMap,需要传入表名
		entityMap := zorm.NewEntityMap(demoStructTableName)
		//设置主键名称
		entityMap.PkColumnName = "id"
		//Set 设置数据库的字段值,主键必须有值
		entityMap.Set("id", "20210630163227149563000042432429")
		entityMap.Set("userName", "TestUpdateEntityMap")
		//更新 "sql":"UPDATE t_demo SET userName=? WHERE id=?","args":["TestUpdateEntityMap","20210630163227149563000042432429"]
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
	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚.如果设置了DisableTransaction=true,Transaction方法失效,不再要求有事务
    //如果zorm.DataSourceConfig.DefaultTxOptions配置不满足需求,可以在zorm.Transaction事务方法前设置事务的隔离级别
	//例如 ctx, _ := dbDao.BindContextTxOptions(ctx, &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}),如果txOptions为nil,使用zorm.DataSourceConfig.DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		demo := &demoStruct{}
		demo.Id = "20210630163227149563000042432429"

		//删除 "sql":"DELETE FROM t_demo WHERE id=?","args":["20210630163227149563000042432429"]
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

	finder := zorm.NewFinder().Append("SELECT * FROM " + demoStructTableName) // select * from t_demo
	//把新产生的newCtx传递到zorm的函数
	list, _ := zorm.QueryMap(newCtx, finder, nil)
	fmt.Println(list)

	//场景2.单个数据库的读写分离.设置读写分离的策略函数.
	zorm.FuncReadWriteStrategy = myReadWriteStrategy

	//场景3.如果是多个数据库,每个数据库还读写分离,按照 场景1 处理

}

//单个数据库的读写分离的策略 rwType=0 read,rwType=1 write
func myReadWriteStrategy(ctx context.Context, rwType int) (*zorm.DBDao,error) {
	//根据自己的业务场景,返回需要的读写dao,每次需要数据库的连接的时候,会调用这个函数
	return dbDao, nil
}

//---------------------------------//

//实现ICustomDriverValueConver接口,扩展自定义类型,例如 达梦数据库text类型,映射出来的是dm.DmClob类型,无法使用string类型直接接收
type CustomDMText struct{}
//GetDriverValue 根据数据库列类型,实体类属性类型,Finder对象,返回driver.Value的实例
//如果无法获取到structFieldType,例如Map查询,会传入nil
//如果返回值为nil,接口扩展逻辑无效,使用原生的方式接收数据库字段值
func (dmtext CustomDMText) GetDriverValue(ctx context.Context, columnType *sql.ColumnType, structFieldType *reflect.Type, finder *zorm.Finder) (driver.Value, error) {
	return &dm.DmClob{}, nil
}
//ConverDriverValue 数据库列类型,实体类属性类型,GetDriverValue返回的driver.Value的临时接收值,Finder对象
//如果无法获取到structFieldType,例如Map查询,会传入nil
//返回符合接收类型值的指针,指针,指针!!!!
func (dmtext CustomDMText) ConverDriverValue(ctx context.Context, columnType *sql.ColumnType, structFieldType *reflect.Type, tempDriverValue driver.Value, finder *zorm.Finder) (interface{}, error) {
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
### 基于seata/hptx实现分布式事务  
#### seata proxy模式 
```golang
//DataSourceConfig 配置  DefaultTxOptions
//DefaultTxOptions: &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false},

// 引入seata-golang V1版本的依赖包
import (
	"github.com/opentrx/mysql"
	"github.com/transaction-wg/seata-golang/pkg/client"
	"github.com/transaction-wg/seata-golang/pkg/client/config"
	"github.com/transaction-wg/seata-golang/pkg/client/rm"
	"github.com/transaction-wg/seata-golang/pkg/client/tm"
	gtxContext "github.com/transaction-wg/seata-golang/pkg/client/context"
)

//配置文件路径
var configPath = "./conf/client.yml"

func main() {

	//初始化配置
	conf := config.InitConf(configPath)
	//初始化RPC客户端
	client.NewRpcClient()
	//注册mysql驱动
	mysql.InitDataResourceManager()
	mysql.RegisterResource(config.GetATConfig().DSN)
	//sqlDB, err := sql.Open("mysql", config.GetATConfig().DSN)


	//后续正常初始化zorm,一定要放到seata mysql 初始化后面!!!

	//................//
	//tm注册事务服务,参照官方例子.(事务托管主要是去掉proxy,对业务零侵入)
	tm.Implement(svc.ProxySvc)
	//................//


	//获取seata的rootContext
	//rootContext := gtxContext.NewRootContext(ctx)
	//rootContext := ctx.(*gtxContext.RootContext)

	//创建seata事务
	//globalTx := tm.GetCurrentOrCreate(rootContext)

	//开始事务
	//globalTx.BeginWithTimeoutAndName(int32(6000), "事务名称", rootContext)

	//事务开启之后获取XID.可以通过gin的header传递,或者其他方式传递
	//xid:=rootContext.GetXID()

	// 如果使用的gin框架,获取到ctx
	// ctx := c.Request.Context()

	// 接受传递过来的XID,绑定到本地ctx
	//ctx =context.WithValue(ctx,mysql.XID,xid)


}
```

#### hptx proxy模式 

hptx已合并[@小口天](https://gitee.com/wuxiangege)的pr, [在hptx代理模式下的zorm使用示例](https://github.com/CECTC/hptx-samples/tree/main/http_proxy_zorm)  

```golang
//DataSourceConfig 配置  DefaultTxOptions
//DefaultTxOptions: &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false},

// 引入hptx 依赖包
import (
	"github.com/cectc/hptx"
	"github.com/cectc/hptx/pkg/config"
	"github.com/cectc/hptx/pkg/resource"
	"github.com/cectc/mysql"
	"github.com/cectc/hptx/pkg/tm"

	gtxContext "github.com/cectc/hptx/pkg/base/context"
)

//配置文件路径
var configPath = "./conf/config.yml"

func main() {

	//初始化配置
	hptx.InitFromFile(configPath)
	
	//注册mysql驱动
    mysql.RegisterResource(config.GetATConfig().DSN)
	resource.InitATBranchResource(mysql.GetDataSourceManager())
	//sqlDB, err := sql.Open("mysql", config.GetATConfig().DSN)


	//后续正常初始化zorm,一定要放到hptx mysql 初始化后面!!!

	//................//
	//tm注册事务服务,参照官方例子.(事务托管主要是去掉proxy,对业务零侵入)
	tm.Implement(svc.ProxySvc)
	//................//


	//获取hptx的rootContext
	//rootContext := gtxContext.NewRootContext(ctx)
	//rootContext := ctx.(*gtxContext.RootContext)

	//创建hptx事务
	//globalTx := tm.GetCurrentOrCreate(rootContext)

	//开始事务
	//globalTx.BeginWithTimeoutAndName(int32(6000), "事务名称", rootContext)

	//事务开启之后获取XID.可以通过gin的header传递,或者其他方式传递
	//xid:=rootContext.GetXID()

	// 如果使用的gin框架,获取到ctx
	// ctx := c.Request.Context()

	// 接受传递过来的XID,绑定到本地ctx
	//ctx =context.WithValue(ctx,mysql.XID,xid)
}
```




#### seata/hptx 事务托管模式

```seata-golang``` 和 ```hptx```实现方式一致,只是实现包不同

hptx已合并[@小口天](https://gitee.com/wuxiangege)的pr, [zorm事务托管hptx示例](https://github.com/CECTC/hptx-samples/tree/main/http_zorm)   

```golang

//不使用proxy代理模式,zorm实现事务托管,不修改业务代码,零侵入实现分布式事务
//tm.Implement(svc.ProxySvc)

//必须手动开启分布式事务,必须放到本地事务开启之前调用
ctx,_ = zorm.BindContextEnableGlobalTransaction(ctx)
// 分布式事务示例代码
_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

    // 获取当前分布式事务的XID.不用考虑怎么来的,如果是分布式事务环境,会自动设置值
    // xid := ctx.Value("XID").(string)

	// 把xid传递到第三方应用
	// req.Header.Set("XID", xid)

	// 如果返回的err不是nil,本地事务和分布式事务就会回滚
	return nil, err
})

///----------第三方应用-------///

// 第三方应用开启事务前,ctx需要绑定XID,例如使用了gin框架

// 接受传递过来的XID,绑定到本地ctx
// xid:=c.Request.Header.Get("XID")
// 获取到ctx
// ctx := c.Request.Context()
// ctx = context.WithValue(ctx,"XID",xid)

// ctx绑定XID之后,调用业务事务
_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

    // 业务代码......

	// 如果返回的err不是nil,本地事务和分布式事务就会回滚
	return nil, err
})



// 建议以下代码放到单独的文件里
//................//

// ZormGlobalTransaction 包装seata/hptx的*tm.DefaultGlobalTransaction,实现zorm.IGlobalTransaction接口
type ZormGlobalTransaction struct {
	*tm.DefaultGlobalTransaction
}

// MyFuncGlobalTransaction zorm适配seata/hptx 全局分布式事务的函数
// 重要!!!!需要配置zorm.DataSourceConfig.FuncGlobalTransaction=MyFuncGlobalTransaction 重要!!!
func MyFuncGlobalTransaction(ctx context.Context) (zorm.IGlobalTransaction, context.Context, error) {
	//获取seata/hptx的rootContext
	rootContext := gtxContext.NewRootContext(ctx)
	//创建seata/hptx事务
	globalTx := tm.GetCurrentOrCreate(rootContext)
	//使用zorm.IGlobalTransaction接口对象包装分布式事务,隔离seata/hptx依赖
	globalTransaction := &ZormGlobalTransaction{globalTx}

	return globalTransaction, rootContext, nil
}


//实现zorm.IGlobalTransaction 托管全局分布式事务接口,seata和hptx目前实现代码一致,只是引用的实现包不同
// BeginGTX 开启全局分布式事务
func (gtx *ZormGlobalTransaction) BeginGTX(ctx context.Context, globalRootContext context.Context) error {
	rootContext := globalRootContext.(*gtxContext.RootContext)
	return gtx.BeginWithTimeout(int32(6000), rootContext)
}

// CommitGTX 提交全局分布式事务
func (gtx *ZormGlobalTransaction) CommitGTX(ctx context.Context, globalRootContext context.Context) error {
	rootContext := globalRootContext.(*gtxContext.RootContext)
	return gtx.Commit(rootContext)
}

// RollbackGTX 回滚全局分布式事务
func (gtx *ZormGlobalTransaction) RollbackGTX(ctx context.Context, globalRootContext context.Context) error {
	rootContext := globalRootContext.(*gtxContext.RootContext)
	//如果是Participant角色,修改为Launcher角色,允许分支事务提交全局事务.
	if gtx.Role != tm.Launcher {
		gtx.Role = tm.Launcher
	}
	return gtx.Rollback(rootContext)
}
// GetGTXID 获取全局分布式事务的XID
func (gtx *ZormGlobalTransaction) GetGTXID(ctx context.Context, globalRootContext context.Context) (string,error) {
	rootContext := globalRootContext.(*gtxContext.RootContext)
	return rootContext.GetXID(), nil
}

//................//
```

### 基于dbpack实现分布式事务  
```dbpack``` 文档:https://cectc.github.io/dbpack-doc/#/README  
使用 Mesh 方式部署,对应用集成比较简单,只需要获取xid,放到sql语句的hint就可以了
```golang
// 开启dbpack事务前,ctx需要绑定sql hint,例如使用gin框架获取header传递过来的xid
xid := c.Request.Header.Get("xid")
// 使用xid生成sql的hint内容,然后将hint绑定到ctx
hint := fmt.Sprintf("/*+ XID('%s') */", xid)
// 获取到ctx
ctx := c.Request.Context()
// 将hint绑定到ctx
ctx,_ = zorm.BindContextSQLHint(ctx,hint)

// ctx绑定sql hint之后,调用业务事务,传递ctx实现分布式事务的传播
_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

    // 业务代码......

	// 如果返回的err不是nil,本地事务和分布式事务就会回滚
	return nil, err
})

```


##  性能压测

   测试代码:https://github.com/springrain/goormbenchmark  
   zorm 1.2.x 版本实现了基础功能,读性能比gorm和xorm快一倍.随着功能持续增加,造成性能下降,目前读性能只快了50%.  
   zorm会持续优化改进性能.  


