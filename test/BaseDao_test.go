// zorm 使用原生的sql语句,没有对sql语法做限制.语句使用Finder作为载体,具体使用方法参见Finder.go文件
// 占位符统一使用?,zorm会根据数据库类型,语句执行前会自动替换占位符,postgresql 把?替换成$1,$2...;mssql替换成@P1,@p2...;orace替换成:1,:2...
// 为了保持数据库兼容性,分页语句需要有order by.mysql没有order by可以正常分页,mssql就必须要有order by才能分页,避免以后迁移风险,zorm对没有order by的分页语句会抛出异常
// zorm的执行方法第一个参数都是 ctx context.Context,业务方法一直传递ctx参数即可,事务传播也是依靠ctx实现.这是golang的标准方式,从web层传递进来即可,例如gin的c.Request.Context()
// zorm的事务操作需要显示使用zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {})开启,zorm通过ctx实现了事务传播,如果当前有事务就使用当前事务,如果当前无事务,就开启新的事务

package test

import (
	"context"
	"testing"

	"gitee.com/chunanyong/zorm"

	//00.引入数据库驱动
	_ "github.com/go-sql-driver/mysql"
)

//baseDao 代表一个数据库的操作,如果有多个数据库,就声明多个BaseDao
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

func TestQueryStruct(t *testing.T) {

}
func TestQueryStructList(t *testing.T) {

}
func TestQueryMap(t *testing.T) {

}
func TestQueryMapList(t *testing.T) {

}
func TestUpdateFinder(t *testing.T) {

}

func TestUpdateStruct(t *testing.T) {

}
func TestUpdateStructNotZeroValue(t *testing.T) {

}
func TestDeleteStruct(t *testing.T) {

}

//保存 EntityMap
func TestSaveEntityMap(t *testing.T) {

}
