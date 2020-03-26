// zorm 使用原生的sql语句,没有对sql语法做限制.语句使用Finder作为载体,具体使用方法参见Finder.go文件
// 占位符统一使用?,zorm会根据数据库类型,语句执行前会自动替换占位符,postgresql 把?替换成$1,$2...;mssql替换成@P1,@p2...;orace替换成:1,:2...
// 为了保持数据库兼容性,分页语句需要有order by.mysql没有order by可以正常分页,mssql就必须要有order by才能分页,避免以后迁移风险,zorm对没有order by的分页语句会抛出异常
// zorm的执行方法第一个参数都是 ctx context.Context,业务方法一直传递ctx参数即可,事务传播也是依靠ctx实现.这是golang的标准方式,从web层传递进来即可,例如gin的c.Request.Context()
// zorm的事务操作需要显示使用zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {})开启,zorm通过ctx实现了事务传播,如果当前有事务就使用当前事务,如果当前无事务,就开启新的事务

package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gitee.com/chunanyong/zorm"

	//00.引入数据库驱动
	_ "github.com/go-sql-driver/mysql"
)

//baseDao 代表一个数据库,如果有多个数据库,就对应声明多个BaseDao
var baseDao *zorm.BaseDao

// ctx默认应该有 web层传入,例如gin的c.Request.Context().这里只是模拟
var ctx = context.Background()

//01.初始化BaseDao
func init() {
	//baseDaoConfig 数据库的配置
	baseDaoConfig := zorm.DataSourceConfig{
		//DSN 数据库的连接字符串
		DSN: "root:root@tcp(127.0.0.1:3306)/readygo?charset=utf8&parseTime=true",
		//DriverName 数据库驱动名称,和DBType对应,一个数据库可以有多个驱动(DriverName)
		DriverName: "mysql",
		//DBType 数据库类型(mysql,postgresql,oracle,mssql,sqlite),zorm判断方言的依据,一个数据库可以有多个驱动(DriverName)
		DBType: "mysql",
		//MaxOpenConns 数据库最大连接数 默认50
		MaxOpenConns: 50,
		//MaxIdleConns 数据库最大空闲连接数 默认50
		MaxIdleConns: 50,
		//ConnMaxLifetimeSecond 连接存活秒时间. 默认600(10分钟)后连接被销毁重建.避免数据库主动断开连接,造成死连接.MySQL默认wait_timeout 28800秒(8小时)
		ConnMaxLifetimeSecond: 600,
		//PrintSQL 打印SQL.会使用logger.info记录SQL
		PrintSQL: true,
	}

	// 根据baseDaoConfig创建baseDao, 一个数据库只执行一次,第一个执行的数据库为 defaultDao,后续zorm.xxx方法,默认使用的就是defaultDao
	baseDao, _ = zorm.NewBaseDao(&baseDaoConfig)
}

//TestSaveStruct 02.测试保存Struct对象
func TestSaveStruct(t *testing.T) {

	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		//创建一个demo对象
		demo := newDemoStruct()

		//保存对象,参数是对象指针.如果主键是自增,会赋值到对象的主键属性
		err := zorm.SaveStruct(ctx, &demo)
		//如果返回的err不是nil,事务就会回滚
		return nil, err
	})
	//标记测试失败
	if err != nil {
		t.Errorf("错误:%v", err)
	}
}

//TestSaveEntityMap 03.测试保存EntityMap对象,用于不方便使用struct的场景,使用Map作为载体
func TestSaveEntityMap(t *testing.T) {

	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚
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

		err := zorm.SaveEntityMap(ctx, entityMap)
		//如果返回的err不是nil,事务就会回滚
		return nil, err
	})
	//标记测试失败
	if err != nil {
		t.Errorf("错误:%v", err)
	}
}

//TestQueryStruct 04.测试查询一个struct对象
func TestQueryStruct(t *testing.T) {

	//声明一个对象的指针,用于承载返回的数据
	demo := &demoStruct{}

	//构造查询用的finder
	finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo
	//finder = zorm.NewSelectFinder(demoStructTableName, "id,userName") // select id,userName from t_demo
	//finder = zorm.NewFinder().Append("SELECT * FROM " + demoStructTableName) // select * from t_demo

	//finder.Append 第一个参数是语句,后面的参数是对应的值,值的顺序要正确.语句统一使用?,zorm会处理数据库的差异
	finder.Append("WHERE id=? and active in(?)", "41b2aa4f-379a-4319-8af9-08472b6e514e", []int{0, 1})

	//执行查询
	err := zorm.QueryStruct(ctx, finder, demo)

	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}
	//打印结果
	fmt.Println(demo)
}

//TestQueryMap 05.测试查询map接收结果,用于不太适合struct的场景,比较灵活
func TestQueryMap(t *testing.T) {

	//构造查询用的finder
	finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo
	//finder.Append 第一个参数是语句,后面的参数是对应的值,值的顺序要正确.语句统一使用?,zorm会处理数据库的差异
	finder.Append("WHERE id=? and active in(?)", "41b2aa4f-379a-4319-8af9-08472b6e514e", []int{0, 1})
	//执行查询
	resultMap, err := zorm.QueryMap(ctx, finder)

	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}
	//打印结果
	fmt.Println(resultMap)
}

//TestQueryStructList 06.测试查询对象列表
func TestQueryStructList(t *testing.T) {
	//创建用于接收结果的slice
	list := make([]demoStruct, 0)

	//构造查询用的finder
	finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo
	//为了保证数据库迁移,分页语句必须要有order by
	finder.Append("order by id asc")

	//创建分页对象,查询完成后,page对象可以直接给前端分页组件使用
	page := zorm.NewPage()
	page.PageNo = 1    //查询第1页,默认是1
	page.PageSize = 20 //每页20条,默认是20

	//执行查询
	err := zorm.QueryStructList(ctx, finder, &list, page)
	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}
	//打印结果
	fmt.Println("总条数:", page.TotalCount, "  列表:", list)
}

//TestQueryMapList 07.测试查询map列表,用于不方便使用struct的场景,一条记录是一个map对象
func TestQueryMapList(t *testing.T) {
	//构造查询用的finder
	finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo
	//为了保证数据库迁移,分页语句必须要有order by
	finder.Append("order by id asc")

	//创建分页对象,查询完成后,page对象可以直接给前端分页组件使用
	page := zorm.NewPage()

	//执行查询
	listMap, err := zorm.QueryMapList(ctx, finder, page)
	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}
	//打印结果
	fmt.Println("总条数:", page.TotalCount, "  列表:", listMap)
}

//TestUpdateStructNotZeroValue 08.更新struct对象,只更新不为零值的字段.主键必须有值
func TestUpdateStructNotZeroValue(t *testing.T) {

	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		//声明一个对象的指针,用于承载返回的数据
		demo := &demoStruct{}
		demo.Id = "41b2aa4f-379a-4319-8af9-08472b6e514e"
		demo.UserName = "UpdateStructNotZeroValue"

		//更新 "sql":"UPDATE t_demo SET userName=? WHERE id=?","args":["UpdateStructNotZeroValue","41b2aa4f-379a-4319-8af9-08472b6e514e"]
		err := zorm.UpdateStructNotZeroValue(ctx, demo)

		//如果返回的err不是nil,事务就会回滚
		return nil, err
	})
	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}

}

//TestUpdateStruct 09.更新struct对象,更新所有字段.主键必须有值
func TestUpdateStruct(t *testing.T) {

	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

		//声明一个对象的指针,用于承载返回的数据
		demo := &demoStruct{}
		demo.Id = "41b2aa4f-379a-4319-8af9-08472b6e514e"
		demo.UserName = "TestUpdateStruct"

		err := zorm.UpdateStruct(ctx, demo)

		//如果返回的err不是nil,事务就会回滚
		return nil, err
	})
	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}
}

//TestUpdateFinder 10.通过finder更新,zorm最灵活的方式,可以编写任何更新语句,甚至手动编写insert语句
func TestUpdateFinder(t *testing.T) {
	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		finder := zorm.NewUpdateFinder(demoStructTableName) // UPDATE t_demo SET
		//finder = zorm.NewDeleteFinder(demoStructTableName)  // DELETE t_demo
		//finder = zorm.NewFinder().Append("UPDATE").Append(demoStructTableName).Append("SET") // UPDATE t_demo SET
		finder.Append("userName=?,active=?", "TestUpdateFinder", 1).Append("WHERE id=?", "41b2aa4f-379a-4319-8af9-08472b6e514e")

		//更新 "sql":"UPDATE t_demo SET  userName=?,active=? WHERE id=?","args":["TestUpdateFinder",1,"41b2aa4f-379a-4319-8af9-08472b6e514e"]
		err := zorm.UpdateFinder(ctx, finder)

		//如果返回的err不是nil,事务就会回滚
		return nil, err
	})
	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}

}

//TestUpdateEntityMap 11.更新一个EntityMap,主键必须有值
func TestUpdateEntityMap(t *testing.T) {
	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		//创建一个EntityMap,需要传入表名
		entityMap := zorm.NewEntityMap(demoStructTableName)
		//设置主键名称
		entityMap.PkColumnName = "id"
		//Set 设置数据库的字段值,主键必须有值
		entityMap.Set("id", "41b2aa4f-379a-4319-8af9-08472b6e514e")
		entityMap.Set("userName", "TestUpdateEntityMap")
		//更新 "sql":"UPDATE t_demo SET userName=? WHERE id=?","args":["TestUpdateEntityMap","41b2aa4f-379a-4319-8af9-08472b6e514e"]
		err := zorm.UpdateEntityMap(ctx, entityMap)

		//如果返回的err不是nil,事务就会回滚
		return nil, err
	})
	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}

}

//TestDeleteStruct 12.删除一个struct对象,主键必须有值
func TestDeleteStruct(t *testing.T) {
	//需要手动开启事务,匿名函数返回的error如果不是nil,事务就会回滚
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		demo := &demoStruct{}
		demo.Id = "ae9987ac-0467-4fe2-a260-516c89292684"

		//删除 "sql":"DELETE FROM t_demo WHERE id=?","args":["ae9987ac-0467-4fe2-a260-516c89292684"]
		err := zorm.DeleteStruct(ctx, demo)

		//如果返回的err不是nil,事务就会回滚
		return nil, err
	})
	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}

}

//TestOther 13.其他的一些说明.非常感谢能看到这一行
func TestOther(t *testing.T) {

	//场景1.多个数据库.通过对应数据库的baseDao,调用BindContextDBConnection函数,把这个数据库的连接绑定到返回的ctx上,然后把ctx传递到zorm的函数即可.
	newCtx, err := baseDao.BindContextDBConnection(ctx)
	if err != nil { //标记测试失败
		t.Errorf("错误:%v", err)
	}
	//把新产生的ctx传递到zorm的函数
	finder := zorm.NewSelectFinder(demoStructTableName).Append("order by id ")
	list, _ := zorm.QueryMapList(newCtx, finder, nil)
	fmt.Println(list)

	//场景2.单个数据库的读写分离.设置读写分离的策略函数.
	zorm.FuncReadWriteStrategy = myReadWriteFuc

	//场景3.如果是多个数据库,每个数据库还读写分离,按照 场景1 处理

}

//单个数据库的读写分离的策略 rwType=0 read,rwType=1 write
func myReadWriteFuc(rwType int) *zorm.BaseDao {
	//根据自己的业务场景,返回需要的读写dao,每次需要数据库的连接的时候,会调用这个函数
	return baseDao
}
