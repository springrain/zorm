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

// Package zorm 使用原生的sql语句,没有对sql语法做限制.语句使用Finder作为载体
// 占位符统一使用?,zorm会根据数据库类型,语句执行前会自动替换占位符,postgresql 把?替换成$1,$2...;mssql替换成@P1,@p2...;orace替换成:1,:2...
// zorm使用 ctx context.Context 参数实现事务传播,ctx从web层传递进来即可,例如gin的c.Request.Context()
// zorm的事务操作需要显示使用zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {})开启
// "package zorm" Use native SQL statements, no restrictions on SQL syntax. Statements use Finder as a carrier
// Use placeholders uniformly "?" "zorm" automatically replaces placeholders before statements are executed,depending on the database type. Replaced with $1, $2... ; Replace MSSQL with @p1,@p2... ; Orace is replaced by :1,:2...,
// "zorm" uses the "ctx context.Context" parameter to achieve transaction propagation,and ctx can be passed in from the web layer, such as "gin's c.Request.Context()",
// "zorm" Transaction operations need to be displayed using "zorm.transaction" (ctx, func(ctx context.context) (interface{}, error) {})
package zorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// FuncReadWriteStrategy 数据库的读写分离的策略,用于外部重写实现自定义的逻辑,也可以使用ctx标识,处理多库的场景,rwType=0 read,rwType=1 write
// 不能归属到DBDao里,BindContextDBConnection已经是指定数据库的连接了,和这个函数会冲突.就作为读写分离的处理方式
// 即便是放到DBDao里,因为是多库,BindContextDBConnection函数调用少不了,业务包装一个方法,指定一下读写获取一个DBDao效果是一样的,唯一就是需要根据业务指定一下读写,其实更灵活了
// FuncReadWriteStrategy Single database read and write separation strategy,used for external replication to implement custom logic, rwType=0 read, rwType=1 write.
// "BindContextDBConnection" is already a connection to the specified database and will conflict with this function. As a single database read and write separation of processing
var FuncReadWriteStrategy = func(ctx context.Context, rwType int) (*DBDao, error) {
	if defaultDao == nil {
		return nil, errors.New("->FuncReadWriteStrategy-->defaultDao为nil,请检查数据库初始化配置是否正确,主要是DSN,DriverName和Dialect")
	}
	return defaultDao, nil
}

// wrapContextStringKey 包装context的key,不直接使用string类型,避免外部直接注入使用
type wrapContextStringKey string

// contextDBConnectionValueKey context WithValue的key,不能是基础类型,例如字符串,包装一下
// The key of context WithValue cannot be a basic type, such as a string, wrap it
const contextDBConnectionValueKey = wrapContextStringKey("contextDBConnectionValueKey")

// contextTxOptionsKey 事务选项设置TxOptions的key,设置事务的隔离级别
const contextTxOptionsKey = wrapContextStringKey("contextTxOptionsKey")

// stringBuilderGrowLen 默认长度
const stringBuilderGrowLen = 100

// DataSourceConfig 数据库连接池的配置
// DateSourceConfig Database connection pool configuration
type DataSourceConfig struct {
	// DSN dataSourceName 连接字符串
	// DSN DataSourceName Database connection string
	DSN string

	// DriverName 数据库驱动名称:mysql,postgres,oracle(go-ora),sqlserver,sqlite3,go_ibm_db,clickhouse,dm,kingbase,aci,taosSql|taosRestful 和Dialect对应
	// DriverName:mysql,dm,postgres,opi8,sqlserver,sqlite3,go_ibm_db,clickhouse,kingbase,aci,taosSql|taosRestful corresponds to Dialect
	DriverName string

	// Dialect 数据库方言:mysql,postgresql,oracle,mssql,sqlite,db2,clickhouse,dm,kingbase,shentong,tdengine 和 DriverName 对应
	// Dialect:mysql,postgresql,oracle,mssql,sqlite,db2,clickhouse,dm,kingbase,shentong,tdengine corresponds to DriverName
	Dialect string

	// Deprecated
	// DBType 即将废弃,请使用Dialect属性
	// DBType is about to be deprecated, please use the Dialect property
	// DBType string

	// SlowSQLMillis 慢sql的时间阈值,单位毫秒.小于0是禁用SQL语句输出;等于0是只输出SQL语句,不计算执行时间;大于0是计算SQL执行时间,并且>=SlowSQLMillis值
	SlowSQLMillis int

	// MaxOpenConns 数据库最大连接数,默认50
	// MaxOpenConns Maximum number of database connections, Default 50
	MaxOpenConns int

	// MaxIdleConns 数据库最大空闲连接数,默认50
	// MaxIdleConns The maximum number of free connections to the database default 50
	MaxIdleConns int

	// ConnMaxLifetimeSecond 连接存活秒时间. 默认600(10分钟)后连接被销毁重建.避免数据库主动断开连接,造成死连接.MySQL默认wait_timeout 28800秒(8小时)
	// ConnMaxLifetimeSecond (Connection survival time in seconds)Destroy and rebuild the connection after the default 600 seconds (10 minutes)
	// Prevent the database from actively disconnecting and causing dead connections. MySQL Default wait_timeout 28800 seconds
	ConnMaxLifetimeSecond int

	// DefaultTxOptions 事务隔离级别的默认配置,默认为nil
	DefaultTxOptions *sql.TxOptions

	// DisableTransaction 禁用事务,默认false,如果设置了DisableTransaction=true,Transaction方法失效,不再要求有事务.为了处理某些数据库不支持事务,比如TDengine
	// 禁用事务应该有驱动伪造事务API,不应该由orm实现
	DisableTransaction bool

	// MockSQLDB 用于mock测试的入口,如果MockSQLDB不为nil,则不使用DSN,直接使用MockSQLDB
	// db, mock, err := sqlmock.New()
	// MockSQLDB *sql.DB

	// FuncGlobalTransaction seata/hptx全局分布式事务的适配函数,返回IGlobalTransaction接口的实现
	// 业务必须调用zorm.BindContextEnableGlobalTransaction(ctx)开启全局分布事务
	// seata-go 的ctx是统一的绑定的是struct,也不是XID字符串.  hptx是分离的,所以返回了两个ctx,兼容两个库
	FuncGlobalTransaction func(ctx context.Context) (IGlobalTransaction, context.Context, context.Context, error)

	// DisableAutoGlobalTransaction 属性已废弃,请勿使用,相关注释仅作记录备忘
	// DisableAutoGlobalTransaction 禁用自动全局分布式事务,默认false,虽然设置了FuncGlobalTransaction,但是并不想全部业务自动开启全局事务
	// DisableAutoGlobalTransaction = false; ctx,_=zorm.BindContextEnableGlobalTransaction(ctx,false) 默认使用全局事务,ctx绑定为false才不开启
	// DisableAutoGlobalTransaction = true;  ctx,_=zorm.BindContextEnableGlobalTransaction(ctx,true) 默认禁用全局事务,ctx绑定为true才开启
	// DisableAutoGlobalTransaction bool

	// SQLDB 使用现有的数据库连接,优先级高于DSN
	SQLDB *sql.DB

	// TDengineInsertsColumnName TDengine批量insert语句中是否有列名.默认false没有列名,插入值和数据库列顺序保持一致,减少语句长度
	TDengineInsertsColumnName bool
}

// DBDao 数据库操作基类,隔离原生操作数据库API入口,所有数据库操作必须通过DBDao进行
// DBDao Database operation base class, isolate the native operation database API entry,all database operations must be performed through DB Dao
type DBDao struct {
	config     *DataSourceConfig
	dataSource *dataSource
}

var defaultDao *DBDao = nil

// NewDBDao 创建dbDao,一个数据库要只执行一次,业务自行控制
// 第一个执行的数据库为 defaultDao,后续zorm.xxx方法,默认使用的就是defaultDao
// NewDBDao Creates dbDao, a database must be executed only once, and the business is controlled by itself
// The first database to be executed is defaultDao, and the subsequent zorm.xxx method is defaultDao by default
func NewDBDao(config *DataSourceConfig) (*DBDao, error) {
	dataSource, err := newDataSource(config)
	if err != nil {
		err = fmt.Errorf("->NewDBDao创建dataSource失败:%w", err)
		FuncLogError(nil, err)
		return nil, err
	}
	dbdao, err := FuncReadWriteStrategy(nil, 1)
	if dbdao == nil {
		defaultDao = &DBDao{config, dataSource}
		return defaultDao, nil
	}
	if err != nil {
		return dbdao, err
	}
	return &DBDao{config, dataSource}, nil
}

// newDBConnection 获取一个dbConnection
// 如果参数dbConnection为nil,使用默认的datasource进行获取dbConnection
// 如果是多库,Dao手动调用newDBConnection(),获得dbConnection,WithValue绑定到子context
// newDBConnection Get a db Connection
// If the parameter db Connection is nil, use the default datasource to get db Connection.
// If it is multi-database, Dao manually calls new DB Connection() to obtain db Connection, and With Value is bound to the sub-context
func (dbDao *DBDao) newDBConnection() (*dataBaseConnection, error) {
	if dbDao == nil || dbDao.dataSource == nil {
		return nil, errors.New("->newDBConnection-->请不要自己创建dbDao,请使用NewDBDao方法进行创建")
	}
	dbConnection := new(dataBaseConnection)
	dbConnection.db = dbDao.dataSource.DB
	dbConnection.config = dbDao.config
	return dbConnection, nil
}

// BindContextDBConnection 多库的时候,通过dbDao创建DBConnection绑定到子context,返回的context就有了DBConnection. parent 不能为空
// BindContextDBConnection In the case of multiple databases, create a DB Connection through db Dao and bind it to a sub-context,and the returned context will have a DB Connection. parent is not nil
func (dbDao *DBDao) BindContextDBConnection(parent context.Context) (context.Context, error) {
	if parent == nil {
		return nil, errors.New("->BindContextDBConnection-->context的parent不能为nil")
	}
	dbConnection, errDBConnection := dbDao.newDBConnection()
	if errDBConnection != nil {
		return parent, errDBConnection
	}
	ctx := context.WithValue(parent, contextDBConnectionValueKey, dbConnection)
	return ctx, nil
}

// BindContextTxOptions 绑定事务的隔离级别,参考sql.IsolationLevel,如果txOptions为nil,使用默认的事务隔离级别.parent不能为空
// 需要在事务开启前调用,也就是zorm.Transaction方法前,不然事务开启之后再调用就无效了
func (dbDao *DBDao) BindContextTxOptions(parent context.Context, txOptions *sql.TxOptions) (context.Context, error) {
	if parent == nil {
		return nil, errors.New("->BindContextTxOptions-->context的parent不能为nil")
	}

	ctx := context.WithValue(parent, contextTxOptionsKey, txOptions)
	return ctx, nil
}

// CloseDB 关闭所有数据库连接
// 请谨慎调用这个方法,会关闭所有数据库连接,用于处理特殊场景,正常使用无需手动关闭数据库连接
func (dbDao *DBDao) CloseDB() error {
	if dbDao == nil || dbDao.dataSource == nil {
		return errors.New("->CloseDB-->请不要自己创建dbDao,请使用NewDBDao方法进行创建")
	}
	return dbDao.dataSource.Close()
}

/*
Transaction 的示例代码
  //匿名函数return的error如果不为nil,事务就会回滚
  zorm.Transaction(ctx context.Context,func(ctx context.Context) (interface{}, error) {

	  //业务代码


	  //return的error如果不为nil,事务就会回滚
      return nil, nil
  })
*/
// 事务方法,隔离dbConnection相关的API.必须通过这个方法进行事务处理,统一事务方式.如果设置了DisableTransaction=true,Transaction方法失效,不再要求有事务
// 如果入参ctx中没有dbConnection,使用defaultDao开启事务并最后提交
// 如果入参ctx有dbConnection且没有事务,调用dbConnection.begin()开启事务并最后提交
// 如果入参ctx有dbConnection且有事务,只使用不提交,有开启方提交事务
// 但是如果遇到错误或者异常,虽然不是事务的开启方,也会回滚事务,让事务尽早回滚
// 在多库的场景,手动获取dbConnection,然后绑定到一个新的context,传入进来
// 不要去掉匿名函数的context参数,因为如果Transaction的context中没有dbConnection,会新建一个context并放入dbConnection,此时的context指针已经变化,不能直接使用Transaction的context参数
// bug(springrain)如果有大神修改了匿名函数内的参数名,例如改为ctx2,这样业务代码实际使用的是Transaction的context参数,如果为没有dbConnection,会抛异常,如果有dbConnection,实际就是一个对象.影响有限.也可以把匿名函数抽到外部
// 如果zorm.DataSourceConfig.DefaultTxOptions配置不满足需求,可以在zorm.Transaction事务方法前设置事务的隔离级别,例如 ctx, _ := dbDao.BindContextTxOptions(ctx, &sql.TxOptions{Isolation: sql.LevelDefault}),如果txOptions为nil,使用zorm.DataSourceConfig.DefaultTxOptions
// return的error如果不为nil,事务就会回滚
// 如果使用了分布式事务,需要设置分布式事务函数zorm.DataSourceConfig.FuncGlobalTransaction,实现IGlobalTransaction接口
// 如果是分布式事务开启方,需要在本地事务前开启分布事务,开启之后获取XID,设值到ctx的XID和TX_XID.XID是seata/hptx MySQL驱动需要,TX_XID是gtxContext.NewRootContext需要
// 分布式事务需要传递XID,接收方context.WithValue(ctx, "XID", XID)绑定到ctx
// 如果分支事务出现异常或者回滚,会立即回滚分布式事务
// Transaction method, isolate db Connection related API. This method must be used for transaction processing and unified transaction mode
// If there is no db Connection in the input ctx, use default Dao to start the transaction and submit it finally
// If the input ctx has db Connection and no transaction, call db Connection.begin() to start the transaction and finally commit
// If the input ctx has a db Connection and a transaction, only use non-commit, and the open party submits the transaction
// If you encounter an error or exception, although it is not the initiator of the transaction, the transaction will be rolled back,
// so that the transaction can be rolled back as soon as possible
// In a multi-database scenario, manually obtain db Connection, then bind it to a new context and pass in
// Do not drop the anonymous function's context parameter, because if the Transaction context does not have a DBConnection,
// then a new context will be created and placed in the DBConnection
// The context pointer has changed and the Transaction context parameters cannot be used directly
// "bug (springrain)" If a great god changes the parameter name in the anonymous function, for example, change it to ctx 2,
// so that the business code actually uses the context parameter of Transaction. If there is no db Connection,
// an exception will be thrown. If there is a db Connection, the actual It is an object
// The impact is limited. Anonymous functions can also be extracted outside
// If the return error is not nil, the transaction will be rolled back
func Transaction(ctx context.Context, doTransaction func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	return transaction(ctx, doTransaction)
}

var transaction = func(ctx context.Context, doTransaction func(ctx context.Context) (interface{}, error)) (info interface{}, err error) {
	// 是否是dbConnection的开启方,如果是开启方,才可以提交事务
	// Whether it is the opener of db Connection, if it is the opener, the transaction can be submitted
	localTxOpen := false
	// 是否是分布式事务的开启方.如果ctx中没有xid,认为是开启方
	globalTxOpen := false
	// 如果dbConnection不存在,则会用默认的datasource开启事务
	// If db Connection does not exist, the default datasource will be used to start the transaction
	var dbConnection *dataBaseConnection
	ctx, dbConnection, err = checkDBConnection(ctx, dbConnection, false, 1)
	if err != nil {
		FuncLogError(ctx, err)
		return nil, err
	}

	// 适配全局事务的函数
	funcGlobalTx := dbConnection.config.FuncGlobalTransaction

	// 实现IGlobalTransaction接口的事务对象
	var globalTransaction IGlobalTransaction
	// 分布式事务的 rootContext,和业务的ctx区别开来,如果业务ctx使用WithValue,就会出现差异
	var globalRootContext context.Context
	// 分布式事务的异常
	var errGlobal error

	// 如果没有事务,并且事务没有被禁用,开启事务
	// 开启本地事务前,需要拿到分布式事务对象
	if dbConnection.tx == nil && (!getContextBoolValue(ctx, contextDisableTransactionValueKey, dbConnection.config.DisableTransaction)) {
		// if dbConnection.tx == nil {
		// 是否使用分布式事务
		enableGlobalTransaction := funcGlobalTx != nil
		if enableGlobalTransaction { // 判断ctx里是否有绑定 enableGlobalTransaction
			/*
				ctxGTXval := ctx.Value(contextEnableGlobalTransactionValueKey)
				if ctxGTXval != nil { //如果有值
					enableGlobalTransaction = ctxGTXval.(bool)
				} else { //如果ctx没有值,就取值DisableAutoGlobalTransaction
					//enableGlobalTransaction = !dbConnection.config.DisableAutoGlobalTransaction
					enableGlobalTransaction = false
				}
			*/
			enableGlobalTransaction = getContextBoolValue(ctx, contextEnableGlobalTransactionValueKey, false)
		}

		// 需要开启分布式事务,初始化分布式事务对象,判断是否是分布式事务入口
		if enableGlobalTransaction {
			// 获取分布式事务的XID
			ctxXIDval := ctx.Value("XID")
			if ctxXIDval != nil { // 如果本地ctx中有XID
				globalXID, _ := ctxXIDval.(string)
				// 不知道为什么需要两个Key,还需要请教seata/hptx团队
				// seata/hptx mysql驱动需要 XID,gtxContext.NewRootContext 需要 TX_XID
				ctx = context.WithValue(ctx, "TX_XID", globalXID)
			} else { // 如果本地ctx中没有XID,也就是没有传递过来XID,认为是分布式事务的开启方.ctx中没有XID和TX_XID的值
				globalTxOpen = true
			}
			// 获取分布式事务实现对象,用于控制事务提交和回滚.分支事务需要ctx中TX_XID有值,将分支事务关联到主事务
			globalTransaction, ctx, globalRootContext, errGlobal = funcGlobalTx(ctx)
			if errGlobal != nil {
				errGlobal = fmt.Errorf("->Transaction-->global:Transaction FuncGlobalTransaction获取IGlobalTransaction接口实现失败:%w ", errGlobal)
				FuncLogError(ctx, errGlobal)
				return nil, errGlobal
			}
			if globalTransaction == nil || globalRootContext == nil {
				errGlobal = errors.New("->Transaction-->global:Transaction FuncGlobalTransaction获取IGlobalTransaction接口的实现为nil ")
				FuncLogError(ctx, errGlobal)
				return nil, errGlobal
			}

		}
		if globalTxOpen { // 如果是分布事务开启方,启动分布式事务
			errGlobal = globalTransaction.BeginGTX(ctx, globalRootContext)
			if errGlobal != nil {
				errGlobal = fmt.Errorf("->Transaction-->global:Transaction 分布式事务开启失败:%w ", errGlobal)
				FuncLogError(ctx, errGlobal)
				return nil, errGlobal
			}

			// 分布式事务开启成功,获取XID,设置到ctx的XID和TX_XID
			// seata/hptx mysql驱动需要 XID,gtxContext.NewRootContext 需要 TX_XID
			globalXID, errGlobal := globalTransaction.GetGTXID(ctx, globalRootContext)
			if errGlobal != nil {
				FuncLogError(ctx, errGlobal)
				return nil, errGlobal
			}
			if globalXID == "" {
				errGlobal = errors.New("->Transaction-->global:globalTransaction.Begin无异常开启后,获取的XID为空")
				FuncLogError(ctx, errGlobal)
				return nil, errGlobal
			}
			ctx = context.WithValue(ctx, "XID", globalXID)
			ctx = context.WithValue(ctx, "TX_XID", globalXID)
		}

		// 开启本地事务/分支事务
		errBeginTx := dbConnection.beginTx(ctx)
		if errBeginTx != nil {
			errBeginTx = fmt.Errorf("->Transaction 事务开启失败:%w ", errBeginTx)
			FuncLogError(ctx, errBeginTx)
			return nil, errBeginTx
		}
		// 本方法开启的事务,由本方法提交
		// The transaction opened by this method is submitted by this method
		localTxOpen = true
	}

	defer func() {
		if r := recover(); r != nil {
			//err = fmt.Errorf("->事务开启失败:%w ", err)
			//记录异常日志
			//if _, ok := r.(runtime.Error); ok {
			//	panic(r)
			//}
			var errOk bool
			err, errOk = r.(error)
			if errOk {
				err = fmt.Errorf("->Transaction-->recover异常:%w", err)
				FuncLogPanic(ctx, err)
			} else {
				err = fmt.Errorf("->Transaction-->recover异常:%v", r)
				FuncLogPanic(ctx, err)
			}
			//if !txOpen { //如果不是开启方,也应该回滚事务,虽然可能造成日志不准确,但是回滚要尽早
			//	return
			//}
			//如果禁用了事务
			if getContextBoolValue(ctx, contextDisableTransactionValueKey, dbConnection.config.DisableTransaction) {
				return
			}
			rberr := dbConnection.rollback()
			if rberr != nil {
				rberr = fmt.Errorf("->Transaction-->recover内事务回滚失败:%w", rberr)
				FuncLogError(ctx, rberr)
			}
			// 任意一个分支事务回滚,分布式事务就整体回滚
			if globalTransaction != nil {
				errGlobal = globalTransaction.RollbackGTX(ctx, globalRootContext)
				if errGlobal != nil {
					errGlobal = fmt.Errorf("->Transaction-->global:recover内globalTransaction事务回滚失败:%w", errGlobal)
					FuncLogError(ctx, errGlobal)
				}
			}

		}
	}()

	// 执行业务的事务函数
	info, err = doTransaction(ctx)

	if err != nil {
		err = fmt.Errorf("->Transaction-->doTransaction业务执行错误:%w", err)
		FuncLogError(ctx, err)

		// 如果禁用了事务
		if getContextBoolValue(ctx, contextDisableTransactionValueKey, dbConnection.config.DisableTransaction) {
			return info, err
		}

		// 不是开启方回滚事务,有可能造成日志记录不准确,但是回滚最重要了,尽早回滚
		// It is not the start party to roll back the transaction, which may cause inaccurate log records,but rollback is the most important, roll back as soon as possible
		errRollback := dbConnection.rollback()
		if errRollback != nil {
			errRollback = fmt.Errorf("->Transaction-->rollback事务回滚失败:%w", errRollback)
			FuncLogError(ctx, errRollback)
		}
		// 任意一个分支事务回滚,分布式事务就整体回滚
		if globalTransaction != nil {
			errGlobal = globalTransaction.RollbackGTX(ctx, globalRootContext)
			if errGlobal != nil {
				errGlobal = fmt.Errorf("->Transaction-->global:Transaction-->rollback globalTransaction事务回滚失败:%w", errGlobal)
				FuncLogError(ctx, errGlobal)
			}
		}
		return info, err
	}
	// 如果是事务开启方,提交事务
	// If it is the transaction opener, commit the transaction
	if localTxOpen {
		errCommit := dbConnection.commit()
		// 本地事务提交成功,如果是全局事务的开启方,提交分布式事务
		if errCommit == nil && globalTxOpen {
			errGlobal = globalTransaction.CommitGTX(ctx, globalRootContext)
			if errGlobal != nil {
				errGlobal = fmt.Errorf("->Transaction-->global:Transaction-->commit globalTransaction 事务提交失败:%w", errGlobal)
				FuncLogError(ctx, errGlobal)
			}
		}
		if errCommit != nil {
			errCommit = fmt.Errorf("->Transaction-->commit事务提交失败:%w", errCommit)
			FuncLogError(ctx, errCommit)
			// 任意一个分支事务回滚,分布式事务就整体回滚
			if globalTransaction != nil {
				errGlobal = globalTransaction.RollbackGTX(ctx, globalRootContext)
				if errGlobal != nil {
					errGlobal = fmt.Errorf("->Transaction-->global:Transaction-->commit失败,然后回滚globalTransaction事务也失败:%w", errGlobal)
					FuncLogError(ctx, errGlobal)
				}
			}

			return info, errCommit
		}

	}

	return info, err
}

var errQueryRow = errors.New("->QueryRow查询出多条数据")

// QueryRow 不要偷懒调用Query返回第一条,问题1.需要构建一个slice,问题2.调用方传递的对象其他值会被抛弃或者覆盖.
// 只查询一个字段,需要使用这个字段的类型进行接收,目前不支持整个struct对象接收
// 根据Finder和封装为指定的entity类型,entity必须是*struct类型或者基础类型的指针.把查询的数据赋值给entity,所以要求指针类型
// context必须传入,不能为空
// 如果数据库是null,基本类型不支持,会返回异常,不做默认值处理,Query因为是列表,会设置为默认值
// QueryRow Don't be lazy to call Query to return the first one
// Question 1. A selice needs to be constructed, and question 2. Other values ​​of the object passed by the caller will be discarded or overwritten
// context must be passed in and cannot be empty
func QueryRow(ctx context.Context, finder *Finder, entity interface{}) (bool, error) {
	return queryRow(ctx, finder, entity)
}

var queryRow = func(ctx context.Context, finder *Finder, entity interface{}) (has bool, err error) {
	typeOf, errCheck := checkEntityKind(entity)
	if errCheck != nil {
		errCheck = fmt.Errorf("->QueryRow-->checkEntityKind类型检查错误:%w", errCheck)
		FuncLogError(ctx, errCheck)
		return has, errCheck
	}
	// 从contxt中获取数据库连接,可能为nil
	// Get database connection from contxt, may be nil
	dbConnection, errFromContxt := getDBConnectionFromContext(ctx)
	if errFromContxt != nil {
		FuncLogError(ctx, errFromContxt)
		return has, errFromContxt
	}
	// 自己构建的dbConnection
	// dbConnection built by yourself
	if dbConnection != nil && dbConnection.db == nil {
		FuncLogError(ctx, errDBConnection)
		return has, errDBConnection
	}

	config, errConfig := getConfigFromConnection(ctx, dbConnection, 0)
	if errConfig != nil {
		FuncLogError(ctx, errConfig)
		return has, errConfig
	}
	// 获取到sql语句
	// Get the sql statement
	sqlstr, errSQL := wrapQuerySQL(ctx, config, finder, nil)
	if errSQL != nil {
		errSQL = fmt.Errorf("->QueryRow-->wrapQuerySQL获取查询SQL语句错误:%w", errSQL)
		FuncLogError(ctx, errSQL)
		return has, errSQL
	}

	// 检查dbConnection.有可能会创建dbConnection或者开启事务,所以要尽可能的接近执行时检查
	// Check db Connection. It is possible to create a db Connection or start a transaction, so check it as close as possible to the execution
	var errDbConnection error
	ctx, dbConnection, errDbConnection = checkDBConnection(ctx, dbConnection, false, 0)
	if errDbConnection != nil {
		FuncLogError(ctx, errDbConnection)
		return has, errDbConnection
	}

	// 根据语句和参数查询
	// Query based on statements and parameters
	rows, errQueryContext := dbConnection.queryContext(ctx, &sqlstr, &finder.values)
	if errQueryContext != nil {
		errQueryContext = fmt.Errorf("->QueryRow-->queryContext查询数据库错误:%w", errQueryContext)
		FuncLogError(ctx, errQueryContext)
		return has, errQueryContext
	}
	// 先判断error 再关闭
	defer func() {
		// 先判断error 再关闭
		rows.Close()
		// 捕获panic,赋值给err,避免程序崩溃
		if r := recover(); r != nil {
			has = false
			var errOk bool
			err, errOk = r.(error)
			if errOk {
				err = fmt.Errorf("->QueryRow-->recover异常:%w", err)
				FuncLogPanic(ctx, err)
			} else {
				err = fmt.Errorf("->QueryRow-->recover异常:%v", r)
				FuncLogPanic(ctx, err)
			}
		}
	}()

	// typeOf := reflect.TypeOf(entity).Elem()

	// 数据库字段类型
	columnTypes, errColumnTypes := rows.ColumnTypes()
	if errColumnTypes != nil {
		errColumnTypes = fmt.Errorf("->QueryRow-->rows.ColumnTypes数据库类型错误:%w", errColumnTypes)
		FuncLogError(ctx, errColumnTypes)
		return has, errColumnTypes
	}
	// 查询的字段长度
	ctLen := len(columnTypes)
	// 是否只有一列,而且可以直接赋值
	oneColumnScanner := false
	if ctLen < 1 { // 没有返回列
		errColumn0 := errors.New("->QueryRow-->ctLen<1,没有返回列")
		FuncLogError(ctx, errColumn0)
		return has, errColumn0
	} else if ctLen == 1 { // 如果只查询一个字段
		// 是否是可以直接扫描的类型
		_, oneColumnScanner = entity.(sql.Scanner)
		if !oneColumnScanner {
			pkgPath := (*typeOf).PkgPath()
			if pkgPath == "" || pkgPath == "time" { // 系统内置变量和time包
				oneColumnScanner = true
			}
		}

	}
	var dbColumnFieldMap *map[string]reflect.StructField
	var exportFieldMap *map[string]reflect.StructField
	if !oneColumnScanner { // 如果不是一个直接可以映射的字段,默认为是sturct
		// 获取到类型的字段缓存
		// Get the type field cache
		dbColumnFieldMap, exportFieldMap, err = getDBColumnExportFieldMap(typeOf)
		if err != nil {
			err = fmt.Errorf("->QueryRow-->getDBColumnFieldMap获取字段缓存错误:%w", err)
			return has, err
		}
	}

	// 反射获取 []driver.Value的值,用于处理nil值和自定义类型
	driverValue := reflect.Indirect(reflect.ValueOf(rows))
	driverValue = driverValue.FieldByName("lastcols")

	// 循环遍历结果集
	// Loop through the result set
	for i := 0; rows.Next(); i++ {
		has = true
		if i > 0 {
			FuncLogError(ctx, errQueryRow)
			return has, errQueryRow
		}
		if oneColumnScanner {
			err = sqlRowsValues(ctx, config, nil, typeOf, rows, &driverValue, columnTypes, entity, dbColumnFieldMap, exportFieldMap)
		} else {
			pv := reflect.ValueOf(entity)
			err = sqlRowsValues(ctx, config, &pv, typeOf, rows, &driverValue, columnTypes, nil, dbColumnFieldMap, exportFieldMap)
		}

		// pv = pv.Elem()
		// scan赋值.是一个指针数组,已经根据struct的属性类型初始化了,sql驱动能感知到参数类型,所以可以直接赋值给struct的指针.这样struct的属性就有值了
		// scan assignment. It is an array of pointers that has been initialized according to the attribute type of the struct,The sql driver can perceive the parameter type,so it can be directly assigned to the pointer of the struct. In this way, the attributes of the struct have values
		// scanerr := rows.Scan(values...)
		if err != nil {
			err = fmt.Errorf("->Query-->sqlRowsValues错误:%w", err)
			FuncLogError(ctx, err)
			return has, err
		}

	}

	return has, err
}

var errQuerySlice = errors.New("->Query数组必须是*[]struct类型或者*[]*struct或者基础类型数组的指针")

// Query 不要偷懒调用QueryMap,需要处理sql驱动支持的sql.Nullxxx的数据类型,也挺麻烦的
// 只查询一个字段,需要使用这个字段的类型进行接收,目前不支持整个struct对象接收
// 根据Finder和封装为指定的entity类型,entity必须是*[]struct类型,已经初始化好的数组,此方法只Append元素,这样调用方就不需要强制类型转换了
// context必须传入,不能为空.如果想不分页,查询所有数据,page传入nil
// Query:Don't be lazy to call QueryMap, you need to deal with the sql,Nullxxx data type supported by the sql driver, which is also very troublesome
// According to the Finder and encapsulation for the specified entity type, the entity must be of the *[]struct type, which has been initialized,This method only Append elements, so the caller does not need to force type conversion
// context must be passed in and cannot be empty
var Query = func(ctx context.Context, finder *Finder, rowsSlicePtr interface{}, page *Page) error {
	return query(ctx, finder, rowsSlicePtr, page)
}

var query = func(ctx context.Context, finder *Finder, rowsSlicePtr interface{}, page *Page) (err error) {
	if rowsSlicePtr == nil { // 如果为nil
		FuncLogError(ctx, errQuerySlice)
		return errQuerySlice
	}

	pvPtr := reflect.ValueOf(rowsSlicePtr)
	if pvPtr.Kind() != reflect.Ptr { // 如果不是指针
		FuncLogError(ctx, errQuerySlice)
		return errQuerySlice
	}

	sliceValue := reflect.Indirect(pvPtr)

	// 如果不是数组
	// If it is not an array.
	if sliceValue.Kind() != reflect.Slice {
		FuncLogError(ctx, errQuerySlice)
		return errQuerySlice
	}
	// 获取数组内的元素类型
	// Get the element type in the array
	sliceElementType := sliceValue.Type().Elem()

	// slice数组里是否是指针,实际参数类似 *[]*struct,兼容这种类型
	sliceElementTypePtr := false
	// 如果数组里还是指针类型
	if sliceElementType.Kind() == reflect.Ptr {
		sliceElementTypePtr = true
		sliceElementType = sliceElementType.Elem()
	}

	//如果不是struct
	//if !(sliceElementType.Kind() == reflect.Struct || allowBaseTypeMap[sliceElementType.Kind()]) {
	//	return errors.New("->Query数组必须是*[]struct类型或者*[]*struct或者基础类型数组的指针")
	//}
	//从contxt中获取数据库连接,可能为nil
	//Get database connection from contxt, may be nil
	dbConnection, errFromContxt := getDBConnectionFromContext(ctx)
	if errFromContxt != nil {
		FuncLogError(ctx, errFromContxt)
		return errFromContxt
	}
	// 自己构建的dbConnection
	// dbConnection built by yourself
	if dbConnection != nil && dbConnection.db == nil {
		FuncLogError(ctx, errDBConnection)
		return errDBConnection
	}
	config, errConfig := getConfigFromConnection(ctx, dbConnection, 0)
	if errConfig != nil {
		FuncLogError(ctx, errConfig)
		return errConfig
	}
	sqlstr, errSQL := wrapQuerySQL(ctx, config, finder, page)
	if errSQL != nil {
		errSQL = fmt.Errorf("->Query-->wrapQuerySQL获取查询SQL语句错误:%w", errSQL)
		FuncLogError(ctx, errSQL)
		return errSQL
	}

	// 检查dbConnection.有可能会创建dbConnection或者开启事务,所以要尽可能的接近执行时检查
	// Check db Connection. It is possible to create a db Connection or start a transaction, so check it as close as possible to the execution
	var errDbConnection error
	ctx, dbConnection, errDbConnection = checkDBConnection(ctx, dbConnection, false, 0)
	if errDbConnection != nil {
		FuncLogError(ctx, errDbConnection)
		return errDbConnection
	}

	// 根据语句和参数查询
	// Query based on statements and parameters
	rows, errQueryContext := dbConnection.queryContext(ctx, &sqlstr, &finder.values)
	if errQueryContext != nil {
		errQueryContext = fmt.Errorf("->Query-->queryContext查询rows错误:%w", errQueryContext)
		FuncLogError(ctx, errQueryContext)
		return errQueryContext
	}
	// 先判断error 再关闭
	defer func() {
		// 先判断error 再关闭
		rows.Close()
		// 捕获panic,赋值给err,避免程序崩溃
		if r := recover(); r != nil {
			var errOk bool
			err, errOk = r.(error)
			if errOk {
				err = fmt.Errorf("->Query-->recover异常:%w", err)
				FuncLogPanic(ctx, err)
			} else {
				err = fmt.Errorf("->Query-->recover异常:%v", r)
				FuncLogPanic(ctx, err)
			}
		}
	}()

	//_, ok := reflect.New(sliceElementType).Interface().(sql.Scanner)

	// 数据库返回的字段类型
	columnTypes, errColumnTypes := rows.ColumnTypes()
	if errColumnTypes != nil {
		errColumnTypes = fmt.Errorf("->Query-->rows.ColumnTypes数据库类型错误:%w", errColumnTypes)
		FuncLogError(ctx, errColumnTypes)
		return errColumnTypes
	}
	// 查询的字段长度
	ctLen := len(columnTypes)
	// 是否只有一列,而且可以直接赋值
	oneColumnScanner := false
	if ctLen < 1 { // 没有返回列
		errColumn0 := errors.New("->Query-->ctLen<1,没有返回列")
		FuncLogError(ctx, errColumn0)
		return errColumn0
	} else if ctLen == 1 { // 如果只查询一个字段
		// 是否是可以直接扫描的类型
		_, oneColumnScanner = reflect.New(sliceElementType).Interface().(sql.Scanner)
		if !oneColumnScanner {
			pkgPath := sliceElementType.PkgPath()
			if pkgPath == "" || pkgPath == "time" { // 系统内置变量和time包
				oneColumnScanner = true
			}
		}

	}
	var dbColumnFieldMap *map[string]reflect.StructField
	var exportFieldMap *map[string]reflect.StructField
	if !oneColumnScanner { // 如果不是一个直接可以映射的字段,默认为是sturct
		// 获取到类型的字段缓存
		// Get the type field cache
		dbColumnFieldMap, exportFieldMap, err = getDBColumnExportFieldMap(&sliceElementType)
		if err != nil {
			err = fmt.Errorf("->Query-->getDBColumnFieldMap获取字段缓存错误:%w", err)
			return err
		}
	}
	// 反射获取 []driver.Value的值,用于处理nil值和自定义类型
	driverValue := reflect.Indirect(reflect.ValueOf(rows))
	driverValue = driverValue.FieldByName("lastcols")
	// TODO 在这里确定字段直接接收或者struct反射,sqlRowsValues 就不再额外处理了,直接映射数据,提升性能
	// 循环遍历结果集
	// Loop through the result set
	for rows.Next() {
		pv := reflect.New(sliceElementType)
		if oneColumnScanner {
			err = sqlRowsValues(ctx, config, nil, &sliceElementType, rows, &driverValue, columnTypes, pv.Interface(), dbColumnFieldMap, exportFieldMap)
		} else {
			err = sqlRowsValues(ctx, config, &pv, &sliceElementType, rows, &driverValue, columnTypes, nil, dbColumnFieldMap, exportFieldMap)
		}

		// err = sqlRowsValues(ctx, dialect, &pv, rows, &driverValue, columnTypes, oneColumnScanner, structType, &dbColumnFieldMap, &exportFieldMap)
		pv = pv.Elem()
		// scan赋值.是一个指针数组,已经根据struct的属性类型初始化了,sql驱动能感知到参数类型,所以可以直接赋值给struct的指针.这样struct的属性就有值了
		// scan assignment. It is an array of pointers that has been initialized according to the attribute type of the struct,The sql driver can perceive the parameter type,so it can be directly assigned to the pointer of the struct. In this way, the attributes of the struct have values
		// scanerr := rows.Scan(values...)
		if err != nil {
			err = fmt.Errorf("->Query-->sqlRowsValues错误:%w", err)
			FuncLogError(ctx, err)
			return err
		}

		// values[i] = f.Addr().Interface()
		// 通过反射给slice添加元素
		// Add elements to slice through reflection
		if sliceElementTypePtr { // 如果数组里是指针地址,*[]*struct
			sliceValue.Set(reflect.Append(sliceValue, pv.Addr()))
		} else {
			sliceValue.Set(reflect.Append(sliceValue, pv))
		}

	}

	// 查询总条数
	// Query total number
	if finder.SelectTotalCount && page != nil {
		count, errCount := selectCount(ctx, finder)
		if errCount != nil {
			errCount = fmt.Errorf("->Query-->selectCount查询总条数错误:%w", errCount)
			FuncLogError(ctx, errCount)
			return errCount
		}
		page.setTotalCount(count)
	}

	return nil
}

var (
	errQueryRowMapFinder = errors.New("->QueryRowMap-->finder参数不能为nil")
	errQueryRowMapMany   = errors.New("->QueryRowMap查询出多条数据")
)

// QueryRowMap 根据Finder查询,封装Map
// context必须传入,不能为空
// QueryRowMap encapsulates Map according to Finder query
// context must be passed in and cannot be empty
func QueryRowMap(ctx context.Context, finder *Finder) (map[string]interface{}, error) {
	return queryRowMap(ctx, finder)
}

var queryRowMap = func(ctx context.Context, finder *Finder) (map[string]interface{}, error) {
	if finder == nil {
		FuncLogError(ctx, errQueryRowMapFinder)
		return nil, errQueryRowMapFinder
	}
	resultMapList, errList := QueryMap(ctx, finder, nil)
	if errList != nil {
		errList = fmt.Errorf("->QueryRowMap-->QueryMap查询错误:%w", errList)
		FuncLogError(ctx, errList)
		return nil, errList
	}
	if resultMapList == nil {
		return nil, nil
	}
	if len(resultMapList) > 1 {
		FuncLogError(ctx, errQueryRowMapMany)
		return resultMapList[0], errQueryRowMapMany
	} else if len(resultMapList) == 0 { // 数据库不存在值
		return nil, nil
	}
	return resultMapList[0], nil
}

var errQueryMapFinder = errors.New("->QueryMap-->finder参数不能为nil")

// QueryMap 根据Finder查询,封装Map数组
// 根据数据库字段的类型,完成从[]byte到Go类型的映射,理论上其他查询方法都可以调用此方法,但是需要处理sql.Nullxxx等驱动支持的类型
// context必须传入,不能为空
// QueryMap According to Finder query, encapsulate Map array
// According to the type of database field, the mapping from []byte to Go type is completed. In theory,other query methods can call this method, but need to deal with types supported by drivers such as sql.Nullxxx
// context must be passed in and cannot be empty
func QueryMap(ctx context.Context, finder *Finder, page *Page) ([]map[string]interface{}, error) {
	return queryMap(ctx, finder, page)
}

var queryMap = func(ctx context.Context, finder *Finder, page *Page) (resultMapList []map[string]interface{}, err error) {
	if finder == nil {
		FuncLogError(ctx, errQueryMapFinder)
		return nil, errQueryMapFinder
	}
	// 从contxt中获取数据库连接,可能为nil
	// Get database connection from contxt, may be nil
	dbConnection, errFromContxt := getDBConnectionFromContext(ctx)
	if errFromContxt != nil {
		FuncLogError(ctx, errFromContxt)
		return nil, errFromContxt
	}
	// 自己构建的dbConnection
	// dbConnection built by yourself
	if dbConnection != nil && dbConnection.db == nil {
		FuncLogError(ctx, errDBConnection)
		return nil, errDBConnection
	}

	config, errConfig := getConfigFromConnection(ctx, dbConnection, 0)
	if errConfig != nil {
		FuncLogError(ctx, errConfig)
		return nil, errConfig
	}
	sqlstr, errSQL := wrapQuerySQL(ctx, config, finder, page)
	if errSQL != nil {
		errSQL = fmt.Errorf("->QueryMap -->wrapQuerySQL查询SQL语句错误:%w", errSQL)
		FuncLogError(ctx, errSQL)
		return nil, errSQL
	}

	// 检查dbConnection.有可能会创建dbConnection或者开启事务,所以要尽可能的接近执行时检查
	// Check db Connection. It is possible to create a db Connection or start a transaction, so check it as close as possible to the execution
	var errDbConnection error
	ctx, dbConnection, errDbConnection = checkDBConnection(ctx, dbConnection, false, 0)
	if errDbConnection != nil {
		return nil, errDbConnection
	}

	// 根据语句和参数查询
	// Query based on statements and parameters
	rows, errQueryContext := dbConnection.queryContext(ctx, &sqlstr, &finder.values)
	if errQueryContext != nil {
		errQueryContext = fmt.Errorf("->QueryMap-->queryContext查询rows错误:%w", errQueryContext)
		FuncLogError(ctx, errQueryContext)
		return nil, errQueryContext
	}
	// 先判断error 再关闭
	defer func() {
		// 先判断error 再关闭
		rows.Close()
		// 捕获panic,赋值给err,避免程序崩溃
		if r := recover(); r != nil {
			var errOk bool
			err, errOk = r.(error)
			if errOk {
				err = fmt.Errorf("->QueryMap-->recover异常:%w", err)
				FuncLogPanic(ctx, err)
			} else {
				err = fmt.Errorf("->QueryMap-->recover异常:%v", r)
				FuncLogPanic(ctx, err)
			}
		}
	}()

	// 数据库返回的列类型
	// The types returned by column Type.scan Type are all []byte, use column Type.database Type to judge one by one
	columnTypes, errColumnTypes := rows.ColumnTypes()
	if errColumnTypes != nil {
		errColumnTypes = fmt.Errorf("->QueryMap-->rows.ColumnTypes数据库返回列名错误:%w", errColumnTypes)
		FuncLogError(ctx, errColumnTypes)
		return nil, errColumnTypes
	}
	// 反射获取 []driver.Value的值
	driverValue := reflect.Indirect(reflect.ValueOf(rows))
	driverValue = driverValue.FieldByName("lastcols")
	resultMapList = make([]map[string]interface{}, 0)
	columnTypeLen := len(columnTypes)
	// 循环遍历结果集
	// Loop through the result set
	for rows.Next() {
		// 接收数据库返回的数据,需要使用指针接收
		// To receive the data returned by the database, you need to use the pointer to receive
		values := make([]interface{}, columnTypeLen)
		// 使用指针类型接收字段值,需要使用interface{}包装一下
		// To use the pointer type to receive the field value, you need to use interface() to wrap it
		result := make(map[string]interface{})

		// 记录需要类型转换的字段信息
		var fieldTempDriverValueMap map[int]*driverValueInfo
		if iscdvm {
			fieldTempDriverValueMap = make(map[int]*driverValueInfo)
		}

		// 给数据赋值初始化变量
		// Initialize variables by assigning values ​​to data
		for i, columnType := range columnTypes {
			dv := driverValue.Index(i)
			if dv.IsValid() && dv.InterfaceData()[0] == 0 { // 该字段的数据库值是null,不再处理,使用默认值
				values[i] = new(interface{})
				continue
			}
			// 类型转换的接口实现
			var customDriverValueConver ICustomDriverValueConver
			// 是否需要类型转换
			var converOK bool = false
			// 类型转换的临时值
			var tempDriverValue driver.Value
			// 根据接收的类型,获取到类型转换的接口实现,优先匹配指定的数据库类型
			databaseTypeName := strings.ToUpper(columnType.DatabaseTypeName())
			// 判断是否有自定义扩展,避免无意义的反射
			if iscdvm {
				customDriverValueConver, converOK = customDriverValueMap[config.Dialect+"."+databaseTypeName]
				if !converOK {
					customDriverValueConver, converOK = customDriverValueMap[databaseTypeName]
				}
			}
			var errGetDriverValue error
			// 如果需要类型转换
			if converOK {
				// 获取需要转的临时值
				tempDriverValue, errGetDriverValue = customDriverValueConver.GetDriverValue(ctx, columnType, nil)
				if errGetDriverValue != nil {
					errGetDriverValue = fmt.Errorf("->QueryMap-->customDriverValueConver.GetDriverValue错误:%w", errGetDriverValue)
					FuncLogError(ctx, errGetDriverValue)
					return nil, errGetDriverValue
				}
				// 返回值为nil,不做任何处理,使用原始逻辑
				if tempDriverValue == nil {
					values[i] = new(interface{})
				} else { // 如果需要类型转换
					values[i] = tempDriverValue
					dvinfo := driverValueInfo{}
					dvinfo.customDriverValueConver = customDriverValueConver
					dvinfo.columnType = columnType
					dvinfo.tempDriverValue = tempDriverValue
					fieldTempDriverValueMap[i] = &dvinfo
				}

				continue
			}

			switch databaseTypeName {

			case "CHAR", "NCHAR", "VARCHAR", "NVARCHAR", "VARCHAR2", "NVARCHAR2", "TINYTEXT", "MEDIUMTEXT", "TEXT", "NTEXT", "LONGTEXT", "LONG", "CHARACTER", "MEMO":
				values[i] = new(string)
			case "INT", "INT4", "INTEGER", "SERIAL", "SERIAL4", "SERIAL2", "TINYINT", "MEDIUMINT", "SMALLINT", "SMALLSERIAL", "INT2", "VARBIT", "AUTONUMBER":
				values[i] = new(int)
			case "BIGINT", "BIGSERIAL", "INT8", "SERIAL8":
				values[i] = new(int64)
			case "FLOAT", "REAL", "FLOAT4", "SINGLE":
				values[i] = new(float32)
			case "DOUBLE", "FLOAT8":
				values[i] = new(float64)
			case "DATE", "TIME", "DATETIME", "TIMESTAMP", "TIMESTAMPTZ", "TIMETZ", "INTERVAL", "DATETIME2", "SMALLDATETIME", "DATETIMEOFFSET":
				values[i] = new(time.Time)
			case "NUMBER":
				precision, scale, isDecimal := columnType.DecimalSize()
				if isDecimal || precision > 18 || precision-scale > 18 { // 如果是Decimal类型
					values[i] = FuncDecimalValue(ctx, config)
				} else if scale > 0 { // 有小数位,默认使用float64接收
					values[i] = new(float64)
				} else if precision-scale > 9 { // 超过9位,使用int64
					values[i] = new(int64)
				} else { // 默认使用int接收
					values[i] = new(int)
				}

			case "DECIMAL", "NUMERIC", "DEC":
				values[i] = FuncDecimalValue(ctx, config)
			case "BOOLEAN", "BOOL", "BIT":
				values[i] = new(bool)
			default:
				// 不需要类型转换,正常赋值
				values[i] = new(interface{})
			}
		}
		// scan赋值
		// scan assignment
		errScan := rows.Scan(values...)
		if errScan != nil {
			errScan = fmt.Errorf("->QueryMap-->rows.Scan错误:%w", errScan)
			FuncLogError(ctx, errScan)
			return nil, errScan
		}

		// 循环 需要类型转换的字段,把临时值赋值给实际的接收对象
		for i, driverValueInfo := range fieldTempDriverValueMap {
			// driverValueInfo := *driverValueInfoPtr
			// 根据列名,字段类型,新值 返回符合接收类型值的指针,返回值是个指针,指针,指针!!!!
			rightValue, errConverDriverValue := driverValueInfo.customDriverValueConver.ConverDriverValue(ctx, driverValueInfo.columnType, driverValueInfo.tempDriverValue, nil)
			if errConverDriverValue != nil {
				errConverDriverValue = fmt.Errorf("->QueryMap-->customDriverValueConver.ConverDriverValue错误:%w", errConverDriverValue)
				FuncLogError(ctx, errConverDriverValue)
				return nil, errConverDriverValue
			}
			// result[driverValueInfo.columnType.Name()] = reflect.ValueOf(rightValue).Elem().Interface()
			values[i] = rightValue
		}

		// 获取每一列的值
		// Get the value of each column
		for i, columnType := range columnTypes {

			// 取到指针下的值,[]byte格式
			// Get the value under the pointer, []byte format
			// v := *(values[i].(*interface{}))
			v := reflect.ValueOf(values[i]).Elem().Interface()
			// 从[]byte转化成实际的类型值,例如string,int
			// Convert from []byte to actual type value, such as string, int
			// v = converValueColumnType(v, columnType)
			// 赋值到Map
			// Assign to Map
			result[columnType.Name()] = v

		}

		// 添加Map到数组
		// Add Map to the array
		resultMapList = append(resultMapList, result)

	}

	// 查询总条数
	// Query total number
	if finder.SelectTotalCount && page != nil {
		count, errCount := selectCount(ctx, finder)
		if errCount != nil {
			errCount = fmt.Errorf("->QueryMap-->selectCount查询总条数错误:%w", errCount)
			FuncLogError(ctx, errCount)
			return resultMapList, errCount
		}
		page.setTotalCount(count)
	}

	return resultMapList, nil
}

// UpdateFinder 更新Finder语句
// ctx不能为nil,参照使用zorm.Transaction方法传入ctx.也不要自己构建DBConnection
// affected影响的行数,如果异常或者驱动不支持,返回-1
// UpdateFinder Update Finder statement
// ctx cannot be nil, refer to zorm.Transaction method to pass in ctx. Don't build DB Connection yourself
// The number of rows affected by affected, if it is abnormal or the driver does not support it, return -1
func UpdateFinder(ctx context.Context, finder *Finder) (int, error) {
	return updateFinder(ctx, finder)
}

var updateFinder = func(ctx context.Context, finder *Finder) (int, error) {
	affected := -1
	if finder == nil {
		return affected, errors.New("->UpdateFinder-->finder不能为空")
	}
	sqlstr, err := finder.GetSQL()
	if err != nil {
		err = fmt.Errorf("->UpdateFinder-->finder.GetSQL()错误:%w", err)
		FuncLogError(ctx, err)
		return affected, err
	}

	// 包装update执行,赋值给影响的函数指针变量,返回*sql.Result
	_, errexec := wrapExecUpdateValuesAffected(ctx, &affected, &sqlstr, &(finder.values), nil)
	if errexec != nil {
		errexec = fmt.Errorf("->UpdateFinder-->wrapExecUpdateValuesAffected执行更新错误:%w", errexec)
		FuncLogError(ctx, errexec)
	}

	return affected, errexec
}

// Insert 保存Struct对象,必须是IEntityStruct类型
// ctx不能为nil,参照使用zorm.Transaction方法传入ctx.也不要自己构建DBConnection
// affected影响的行数,如果异常或者驱动不支持,返回-1
// Insert saves the Struct object, which must be of type IEntityStruct
// ctx cannot be nil, refer to zorm.Transaction method to pass in ctx. Don't build dbConnection yourself
// The number of rows affected by affected, if it is abnormal or the driver does not support it, return -1
func Insert(ctx context.Context, entity IEntityStruct) (int, error) {
	return insert(ctx, entity)
}

var insert = func(ctx context.Context, entity IEntityStruct) (int, error) {
	affected := -1
	if entity == nil {
		return affected, errors.New("->Insert-->entity对象不能为空")
	}
	typeOf, columns, values, columnAndValueErr := columnAndValue(ctx, entity, false, true)
	if columnAndValueErr != nil {
		columnAndValueErr = fmt.Errorf("->Insert-->columnAndValue获取实体类的列和值错误:%w", columnAndValueErr)
		FuncLogError(ctx, columnAndValueErr)
		return affected, columnAndValueErr
	}
	if len(*columns) < 1 {
		return affected, errors.New("->Insert-->没有tag信息,请检查struct中 column 的tag")
	}
	// 从contxt中获取数据库连接,可能为nil
	// Get database connection from contxt, may be nil
	dbConnection, errFromContxt := getDBConnectionFromContext(ctx)
	if errFromContxt != nil {
		return affected, errFromContxt
	}
	// 自己构建的dbConnection
	// dbConnection built by yourself
	if dbConnection != nil && dbConnection.db == nil {
		return affected, errDBConnection
	}

	// SQL语句
	// SQL statement
	sqlstr, autoIncrement, pktype, err := wrapInsertSQL(ctx, dbConnection.config, typeOf, entity, columns, values)
	if err != nil {
		err = fmt.Errorf("->Insert-->wrapInsertSQL获取保存语句错误:%w", err)
		FuncLogError(ctx, err)
		return affected, err
	}

	// oracle 12c+ 支持IDENTITY属性的自增列,因为分页也要求12c+的语法,所以数据库就IDENTITY创建自增吧
	// 处理序列产生的自增主键,例如oracle,postgresql等
	var lastInsertID, zormSQLOutReturningID *int64
	// var zormSQLOutReturningID *int64
	// 如果是postgresql的SERIAL自增,需要使用 RETURNING 返回主键的值
	if autoIncrement > 0 {
		config, errConfig := getConfigFromConnection(ctx, dbConnection, 1)
		if errConfig != nil {
			return affected, errConfig
		}
		lastInsertID, zormSQLOutReturningID = wrapAutoIncrementInsertSQL(ctx, config, entity.GetPKColumnName(), sqlstr, values)

	}

	// 包装update执行,赋值给影响的函数指针变量,返回*sql.Result
	res, errexec := wrapExecUpdateValuesAffected(ctx, &affected, sqlstr, values, lastInsertID)
	if errexec != nil {
		errexec = fmt.Errorf("->Insert-->wrapExecUpdateValuesAffected执行保存错误:%w", errexec)
		FuncLogError(ctx, errexec)
		return affected, errexec
	}

	// 如果是自增主键
	// If it is an auto-incrementing primary key
	if autoIncrement > 0 {
		// 如果是oracle,shentong 的返回自增主键
		if lastInsertID == nil && zormSQLOutReturningID != nil {
			lastInsertID = zormSQLOutReturningID
		}

		var autoIncrementIDInt64 int64
		var err error
		if lastInsertID != nil {
			autoIncrementIDInt64 = *lastInsertID
		} else {
			// 需要数据库支持,获取自增主键
			// Need database support, get auto-incrementing primary key
			autoIncrementIDInt64, err = (*res).LastInsertId()
		}

		// 数据库不支持自增主键,不再赋值给struct属性
		// The database does not support self-incrementing primary keys, and no longer assigns values ​​to struct attributes
		if err != nil {
			err = fmt.Errorf("->Insert-->LastInsertId数据库不支持自增主键,不再赋值给struct属性:%w", err)
			FuncLogError(ctx, err)
			return affected, nil
		}
		pkName := entity.GetPKColumnName()
		if pktype == "int" {
			// int64 转 int
			// int64 to int
			autoIncrementIDInt, _ := typeConvertInt64toInt(autoIncrementIDInt64)
			// 设置自增主键的值
			// Set the value of the auto-incrementing primary key
			err = setFieldValueByColumnName(entity, pkName, autoIncrementIDInt)
		} else if pktype == "int64" {
			// 设置自增主键的值
			// Set the value of the auto-incrementing primary key
			err = setFieldValueByColumnName(entity, pkName, autoIncrementIDInt64)
		}

		if err != nil {
			err = fmt.Errorf("->Insert-->setFieldValueByColumnName反射赋值数据库返回的自增主键错误:%w", err)
			FuncLogError(ctx, err)
			return affected, err
		}
	}

	return affected, nil
}

// InsertSlice 批量保存Struct Slice 数组对象,必须是[]IEntityStruct类型,使用IEntityStruct接口,兼容Struct实体类
// 如果是自增主键,无法对Struct对象里的主键属性赋值
// ctx不能为nil,参照使用zorm.Transaction方法传入ctx.也不要自己构建DBConnection
// affected影响的行数,如果异常或者驱动不支持,返回-1
func InsertSlice(ctx context.Context, entityStructSlice []IEntityStruct) (int, error) {
	return insertSlice(ctx, entityStructSlice)
}

var insertSlice = func(ctx context.Context, entityStructSlice []IEntityStruct) (int, error) {
	affected := -1
	if entityStructSlice == nil || len(entityStructSlice) < 1 {
		return affected, errors.New("->InsertSlice-->entityStructSlice对象数组不能为空")
	}
	// 第一个对象,获取第一个Struct对象,用于获取数据库字段,也获取了值
	entity := entityStructSlice[0]
	typeOf, columns, values, columnAndValueErr := columnAndValue(ctx, entity, false, true)
	if columnAndValueErr != nil {
		columnAndValueErr = fmt.Errorf("->InsertSlice-->columnAndValue获取实体类的列和值错误:%w", columnAndValueErr)
		FuncLogError(ctx, columnAndValueErr)
		return affected, columnAndValueErr
	}
	if len(*columns) < 1 {
		return affected, errors.New("->InsertSlice-->columns没有tag信息,请检查struct中 column 的tag")
	}
	// 从contxt中获取数据库连接,可能为nil
	dbConnection, errFromContxt := getDBConnectionFromContext(ctx)
	if errFromContxt != nil {
		return affected, errFromContxt
	}
	// 自己构建的dbConnection
	if dbConnection != nil && dbConnection.db == nil {
		return affected, errDBConnection
	}
	config, errConfig := getConfigFromConnection(ctx, dbConnection, 1)
	if errConfig != nil {
		return affected, errConfig
	}
	// SQL语句
	sqlstr, _, err := wrapInsertSliceSQL(ctx, config, typeOf, entityStructSlice, columns, values)
	if err != nil {
		err = fmt.Errorf("->InsertSlice-->wrapInsertSliceSQL获取保存语句错误:%w", err)
		FuncLogError(ctx, err)
		return affected, err
	}
	// 包装update执行,赋值给影响的函数指针变量,返回*sql.Result
	_, errexec := wrapExecUpdateValuesAffected(ctx, &affected, sqlstr, values, nil)
	if errexec != nil {
		errexec = fmt.Errorf("->InsertSlice-->wrapExecUpdateValuesAffected执行保存错误:%w", errexec)
		FuncLogError(ctx, errexec)
	}

	return affected, errexec
}

// Update 更新struct所有属性,必须是IEntityStruct类型
// ctx不能为nil,参照使用zorm.Transaction方法传入ctx.也不要自己构建DBConnection
func Update(ctx context.Context, entity IEntityStruct) (int, error) {
	return update(ctx, entity)
}

var update = func(ctx context.Context, entity IEntityStruct) (int, error) {
	finder, err := WrapUpdateStructFinder(ctx, entity, false)
	if err != nil {
		err = fmt.Errorf("->Update-->WrapUpdateStructFinder包装Finder错误:%w", err)
		FuncLogError(ctx, err)
		return 0, err
	}
	return UpdateFinder(ctx, finder)
}

// UpdateNotZeroValue 更新struct不为默认零值的属性,必须是IEntityStruct类型,主键必须有值
// ctx不能为nil,参照使用zorm.Transaction方法传入ctx.也不要自己构建DBConnection
func UpdateNotZeroValue(ctx context.Context, entity IEntityStruct) (int, error) {
	return updateNotZeroValue(ctx, entity)
}

var updateNotZeroValue = func(ctx context.Context, entity IEntityStruct) (int, error) {
	finder, err := WrapUpdateStructFinder(ctx, entity, true)
	if err != nil {
		err = fmt.Errorf("->UpdateNotZeroValue-->WrapUpdateStructFinder包装Finder错误:%w", err)
		FuncLogError(ctx, err)
		return 0, err
	}
	return UpdateFinder(ctx, finder)
}

// Delete 根据主键删除一个对象.必须是IEntityStruct类型
// ctx不能为nil,参照使用zorm.Transaction方法传入ctx.也不要自己构建DBConnection
// affected影响的行数,如果异常或者驱动不支持,返回-1
func Delete(ctx context.Context, entity IEntityStruct) (int, error) {
	return delete(ctx, entity)
}

var delete = func(ctx context.Context, entity IEntityStruct) (int, error) {
	affected := -1
	typeOf, checkerr := checkEntityKind(entity)
	if checkerr != nil {
		return affected, checkerr
	}

	pkName, pkNameErr := entityPKFieldName(entity, typeOf)

	if pkNameErr != nil {
		pkNameErr = fmt.Errorf("->Delete-->entityPKFieldName获取主键名称错误:%w", pkNameErr)
		FuncLogError(ctx, pkNameErr)
		return affected, pkNameErr
	}

	value, e := structFieldValue(entity, pkName)
	if e != nil {
		e = fmt.Errorf("->Delete-->structFieldValue获取主键值错误:%w", e)
		FuncLogError(ctx, e)
		return affected, e
	}

	// SQL语句
	sqlstr, err := wrapDeleteSQL(ctx, entity)
	if err != nil {
		err = fmt.Errorf("->Delete-->wrapDeleteSQL获取SQL语句错误:%w", err)
		FuncLogError(ctx, err)
		return affected, err
	}
	// 包装update执行,赋值给影响的函数指针变量,返回*sql.Result
	values := make([]interface{}, 1)
	values[0] = value
	_, errexec := wrapExecUpdateValuesAffected(ctx, &affected, &sqlstr, &values, nil)
	if errexec != nil {
		errexec = fmt.Errorf("->Delete-->wrapExecUpdateValuesAffected执行删除错误:%w", errexec)
		FuncLogError(ctx, errexec)
	}

	return affected, errexec
}

// InsertEntityMap 保存*IEntityMap对象.使用Map保存数据,用于不方便使用struct的场景,如果主键是自增或者序列,不要entityMap.Set主键的值
// ctx不能为nil,参照使用zorm.Transaction方法传入ctx.也不要自己构建DBConnection
// affected影响的行数,如果异常或者驱动不支持,返回-1
func InsertEntityMap(ctx context.Context, entity IEntityMap) (int, error) {
	return insertEntityMap(ctx, entity)
}

var insertEntityMap = func(ctx context.Context, entity IEntityMap) (int, error) {
	affected := -1
	// 检查是否是指针对象
	_, checkerr := checkEntityKind(entity)
	if checkerr != nil {
		return affected, checkerr
	}

	// 从contxt中获取数据库连接,可能为nil
	dbConnection, errFromContxt := getDBConnectionFromContext(ctx)
	if errFromContxt != nil {
		return affected, errFromContxt
	}

	// 自己构建的dbConnection
	if dbConnection != nil && dbConnection.db == nil {
		return affected, errDBConnection
	}

	// SQL语句
	sqlstr, values, autoIncrement, err := wrapInsertEntityMapSQL(ctx, dbConnection.config, entity)
	if err != nil {
		err = fmt.Errorf("->InsertEntityMap-->wrapInsertEntityMapSQL获取SQL语句错误:%w", err)
		FuncLogError(ctx, err)
		return affected, err
	}

	// 处理序列产生的自增主键,例如oracle,postgresql等
	var lastInsertID, zormSQLOutReturningID *int64
	// 如果是postgresql的SERIAL自增,需要使用 RETURNING 返回主键的值
	if autoIncrement && entity.GetPKColumnName() != "" {
		config, errConfig := getConfigFromConnection(ctx, dbConnection, 1)
		if errConfig != nil {
			return affected, errConfig
		}
		lastInsertID, zormSQLOutReturningID = wrapAutoIncrementInsertSQL(ctx, config, entity.GetPKColumnName(), &sqlstr, values)
	}

	// 包装update执行,赋值给影响的函数指针变量,返回*sql.Result
	res, errexec := wrapExecUpdateValuesAffected(ctx, &affected, &sqlstr, values, lastInsertID)
	if errexec != nil {
		errexec = fmt.Errorf("->InsertEntityMap-->wrapExecUpdateValuesAffected执行保存错误:%w", errexec)
		FuncLogError(ctx, errexec)
		return affected, errexec
	}

	// 如果是自增主键
	if autoIncrement {
		// 如果是oracle,shentong 的返回自增主键
		if lastInsertID == nil && zormSQLOutReturningID != nil {
			lastInsertID = zormSQLOutReturningID
		}

		var autoIncrementIDInt64 int64
		var e error
		if lastInsertID != nil {
			autoIncrementIDInt64 = *lastInsertID
		} else {
			// 需要数据库支持,获取自增主键
			// Need database support, get auto-incrementing primary key
			autoIncrementIDInt64, e = (*res).LastInsertId()
		}
		if e != nil { // 数据库不支持自增主键,不再赋值给struct属性
			e = fmt.Errorf("->InsertEntityMap数据库不支持自增主键,不再赋值给IEntityMap:%w", e)
			FuncLogError(ctx, e)
			return affected, nil
		}
		// int64 转 int
		strInt64 := strconv.FormatInt(autoIncrementIDInt64, 10)
		autoIncrementIDInt, _ := strconv.Atoi(strInt64)
		// 设置自增主键的值
		entity.Set(entity.GetPKColumnName(), autoIncrementIDInt)
	}

	return affected, nil
}

// InsertEntityMapSlice 保存[]IEntityMap对象.使用Map保存数据,用于不方便使用struct的场景,如果主键是自增或者序列,不要entityMap.Set主键的值
// ctx不能为nil,参照使用zorm.Transaction方法传入ctx.也不要自己构建DBConnection
// affected影响的行数,如果异常或者驱动不支持,返回-1
func InsertEntityMapSlice(ctx context.Context, entityMapSlice []IEntityMap) (int, error) {
	return insertEntityMapSlice(ctx, entityMapSlice)
}

var insertEntityMapSlice = func(ctx context.Context, entityMapSlice []IEntityMap) (int, error) {
	affected := -1
	// 从contxt中获取数据库连接,可能为nil
	dbConnection, errFromContxt := getDBConnectionFromContext(ctx)
	if errFromContxt != nil {
		return affected, errFromContxt
	}
	// 自己构建的dbConnection
	if dbConnection != nil && dbConnection.db == nil {
		return affected, errDBConnection
	}
	config, errConfig := getConfigFromConnection(ctx, dbConnection, 1)
	if errConfig != nil {
		return affected, errConfig
	}
	// SQL语句
	sqlstr, values, err := wrapInsertEntityMapSliceSQL(ctx, config, entityMapSlice)
	if err != nil {
		err = fmt.Errorf("->InsertEntityMapSlice-->wrapInsertEntityMapSliceSQL获取SQL语句错误:%w", err)
		FuncLogError(ctx, err)
		return affected, err
	}

	// 包装update执行,赋值给影响的函数指针变量,返回*sql.Result
	_, errexec := wrapExecUpdateValuesAffected(ctx, &affected, sqlstr, values, nil)
	if errexec != nil {
		errexec = fmt.Errorf("->InsertEntityMapSlice-->wrapExecUpdateValuesAffected执行保存错误:%w", errexec)
		FuncLogError(ctx, errexec)
		return affected, errexec
	}
	return affected, errexec
}

// UpdateEntityMap 更新IEntityMap对象.用于不方便使用struct的场景,主键必须有值
// ctx不能为nil,参照使用zorm.Transaction方法传入ctx.也不要自己构建DBConnection
// affected影响的行数,如果异常或者驱动不支持,返回-1
// UpdateEntityMap Update IEntityMap object. Used in scenarios where struct is not convenient, the primary key must have a value
// ctx cannot be nil, refer to zorm.Transaction method to pass in ctx. Don't build DB Connection yourself
// The number of rows affected by "affected", if it is abnormal or the driver does not support it, return -1
func UpdateEntityMap(ctx context.Context, entity IEntityMap) (int, error) {
	return updateEntityMap(ctx, entity)
}

var updateEntityMap = func(ctx context.Context, entity IEntityMap) (int, error) {
	affected := -1
	// 检查是否是指针对象
	// Check if it is a pointer
	_, checkerr := checkEntityKind(entity)
	if checkerr != nil {
		return affected, checkerr
	}
	// 从contxt中获取数据库连接,可能为nil
	// Get database connection from contxt, it may be nil
	dbConnection, errFromContxt := getDBConnectionFromContext(ctx)
	if errFromContxt != nil {
		return affected, errFromContxt
	}
	// 自己构建的dbConnection
	// dbConnection built by yourself
	if dbConnection != nil && dbConnection.db == nil {
		return affected, errDBConnection
	}

	// SQL语句
	// SQL statement
	sqlstr, values, err := wrapUpdateEntityMapSQL(ctx, entity)
	if err != nil {
		err = fmt.Errorf("->UpdateEntityMap-->wrapUpdateEntityMapSQL获取SQL语句错误:%w", err)
		FuncLogError(ctx, err)
		return affected, err
	}
	// 包装update执行,赋值给影响的函数指针变量,返回*sql.Result
	_, errexec := wrapExecUpdateValuesAffected(ctx, &affected, sqlstr, values, nil)
	if errexec != nil {
		errexec = fmt.Errorf("->UpdateEntityMap-->wrapExecUpdateValuesAffected执行更新错误:%w", errexec)
		FuncLogError(ctx, errexec)
	}

	return affected, errexec
}

// IsInTransaction 检查ctx是否包含事务
func IsInTransaction(ctx context.Context) (bool, error) {
	dbConnection, err := getDBConnectionFromContext(ctx)
	if err != nil {
		return false, err
	}
	if dbConnection != nil && dbConnection.tx != nil {
		return true, err
	}
	return false, err
}

// IsBindDBConnection 检查ctx是否已经绑定数据库连接
func IsBindDBConnection(ctx context.Context) (bool, error) {
	dbConnection, err := getDBConnectionFromContext(ctx)
	if err != nil {
		return false, err
	}
	if dbConnection != nil {
		return true, err
	}
	return false, err
}

// WrapUpdateStructFinder 返回更新IEntityStruct的Finder对象
// ctx不能为nil,参照使用zorm.Transaction方法传入ctx.也不要自己构建DBConnection
// Finder为更新执行的Finder,更新语句统一使用Finder执行
// updateStructFunc Update object
// ctx cannot be nil, refer to zorm.Transaction method to pass in ctx. Don't build DB Connection yourself
// Finder is the Finder that executes the update, and the update statement is executed uniformly using the Finder
func WrapUpdateStructFinder(ctx context.Context, entity IEntityStruct, onlyUpdateNotZero bool) (*Finder, error) {
	// affected := -1
	if entity == nil {
		return nil, errors.New("->WrapUpdateStructFinder-->entity对象不能为空")
	}

	typeOf, columns, values, columnAndValueErr := columnAndValue(ctx, entity, onlyUpdateNotZero, false)
	if columnAndValueErr != nil {
		return nil, columnAndValueErr
	}

	// SQL语句
	// SQL statement
	sqlstr, err := wrapUpdateSQL(ctx, typeOf, entity, columns, values)
	if err != nil {
		return nil, err
	}
	// finder对象
	finder := NewFinder()
	finder.sqlstr = sqlstr
	finder.sqlBuilder.WriteString(sqlstr)
	finder.values = *values
	return finder, nil
}

// selectCount 根据finder查询总条数
// context必须传入,不能为空
// selectCount Query the total number of items according to finder
// context must be passed in and cannot be empty
func selectCount(ctx context.Context, finder *Finder) (int, error) {
	if finder == nil {
		return -1, errors.New("->selectCount-->finder参数为nil")
	}
	// 自定义的查询总条数Finder,主要是为了在group by等复杂情况下,为了性能,手动编写总条数语句
	// Customized query total number Finder,mainly for the sake of performance in complex situations such as group by, manually write the total number of statements
	if finder.CountFinder != nil {
		count := -1
		_, err := QueryRow(ctx, finder.CountFinder, &count)
		if err != nil {
			return -1, err
		}
		return count, nil
	}

	countsql, counterr := finder.GetSQL()
	if counterr != nil {
		return -1, counterr
	}

	// 查询order by 的位置
	// Query the position of order by
	locOrderBy := findOrderByIndex(&countsql)
	// 如果存在order by
	// If there is order by
	if len(locOrderBy) > 0 {
		countsql = countsql[:locOrderBy[0]]
	}
	s := strings.ToLower(countsql)
	gbi := -1
	locGroupBy := findGroupByIndex(&countsql)
	if len(locGroupBy) > 0 {
		gbi = locGroupBy[0]
	}
	var sqlBuilder strings.Builder
	sqlBuilder.Grow(stringBuilderGrowLen)
	// 特殊关键字,包装SQL
	// Special keywords, wrap SQL
	if strings.Contains(s, " distinct ") || strings.Contains(s, " union ") || gbi > -1 {
		// countsql = "SELECT COUNT(*)  frame_row_count FROM (" + countsql + ") temp_frame_noob_table_name WHERE 1=1 "
		sqlBuilder.WriteString("SELECT COUNT(*)  frame_row_count FROM (")
		sqlBuilder.WriteString(countsql)
		sqlBuilder.WriteString(") temp_frame_noob_table_name WHERE 1=1 ")
	} else {
		locFrom := findSelectFromIndex(&countsql)
		// 没有找到FROM关键字,认为是异常语句
		// The FROM keyword was not found, which is considered an abnormal statement
		if len(locFrom) == 0 {
			return -1, errors.New("->selectCount-->findFromIndex没有FROM关键字,语句错误")
		}
		// countsql = "SELECT COUNT(*) " + countsql[locFrom[0]:]
		sqlBuilder.WriteString("SELECT COUNT(*) ")
		sqlBuilder.WriteString(countsql[locFrom[0]:])
	}
	countsql = sqlBuilder.String()
	countFinder := NewFinder()
	countFinder.Append(countsql)
	countFinder.values = finder.values
	countFinder.InjectionCheck = finder.InjectionCheck

	count := -1
	_, cerr := QueryRow(ctx, countFinder, &count)
	if cerr != nil {
		return -1, cerr
	}
	return count, nil
}

// getDBConnectionFromContext 从Conext中获取数据库连接
// getDBConnectionFromContext Get database connection from Conext
func getDBConnectionFromContext(ctx context.Context) (*dataBaseConnection, error) {
	if ctx == nil {
		return nil, errors.New("->getDBConnectionFromContext-->context不能为空")
	}
	// 获取数据库连接
	// Get database connection
	value := ctx.Value(contextDBConnectionValueKey)
	if value == nil {
		return nil, nil
	}
	dbConnection, isdb := value.(*dataBaseConnection)
	if !isdb { // 不是数据库连接
		return nil, errors.New("->getDBConnectionFromContext-->context传递了错误的*DBConnection类型值")
	}
	return dbConnection, nil
}

// 变量名建议errFoo这样的驼峰
// The variable name suggests a hump like "errFoo"
var errDBConnection = errors.New("更新操作需要使用zorm.Transaction开启事务.读取操作如果ctx没有dbConnection,使用FuncReadWriteStrategy(ctx,rwType).newDBConnection(),如果dbConnection有事务,就使用事务查询")

// checkDBConnection 检查dbConnection.有可能会创建dbConnection或者开启事务,所以要尽可能的接近执行时检查
// context必须传入,不能为空.rwType=0 read,rwType=1 write
// checkDBConnection It is possible to create a db Connection or open a transaction, so check it as close as possible to execution
// The context must be passed in and cannot be empty. rwType=0 read, rwType=1 write
func checkDBConnection(ctx context.Context, dbConnection *dataBaseConnection, hastx bool, rwType int) (context.Context, *dataBaseConnection, error) {
	var errFromContext error
	if dbConnection == nil {
		dbConnection, errFromContext = getDBConnectionFromContext(ctx)
		if errFromContext != nil {
			return ctx, nil, errFromContext
		}
	}

	// dbConnection为空
	// dbConnection is nil
	if dbConnection == nil {
		dbdao, err := FuncReadWriteStrategy(ctx, rwType)
		if err != nil {
			return ctx, nil, err
		}
		// 是否禁用了事务
		disabletx := getContextBoolValue(ctx, contextDisableTransactionValueKey, dbdao.config.DisableTransaction)
		// 如果要求有事务,事务需要手动zorm.Transaction显示开启.如果自动开启,就会为了偷懒,每个操作都自动开启,事务就失去意义了
		if hastx && (!disabletx) {
			// if hastx {
			return ctx, nil, errDBConnection
		}

		// 如果要求没有事务,实例化一个默认的dbConnection
		// If no transaction is required, instantiate a default db Connection
		var errGetDBConnection error

		dbConnection, errGetDBConnection = dbdao.newDBConnection()
		if errGetDBConnection != nil {
			return ctx, nil, errGetDBConnection
		}
		// 把dbConnection放入context
		// Put db Connection into context
		ctx = context.WithValue(ctx, contextDBConnectionValueKey, dbConnection)

	} else { // 如果dbConnection存在
		// If db Connection exists
		if dbConnection.db == nil { // 禁止外部构建
			return ctx, dbConnection, errDBConnection
		}
		if dbConnection.tx == nil && hastx && (!getContextBoolValue(ctx, contextDisableTransactionValueKey, dbConnection.config.DisableTransaction)) {
			// if dbConnection.tx == nil && hastx { //如果要求有事务,事务需要手动zorm.Transaction显示开启.如果自动开启,就会为了偷懒,每个操作都自动开启,事务就失去意义了
			return ctx, dbConnection, errDBConnection
		}
	}
	return ctx, dbConnection, nil
}

// wrapExecUpdateValuesAffected 包装update执行,赋值给影响的函数指针变量,返回*sql.Result
func wrapExecUpdateValuesAffected(ctx context.Context, affected *int, sqlstrptr *string, values *[]interface{}, lastInsertID *int64) (*sql.Result, error) {
	// 必须要有dbConnection和事务.有可能会创建dbConnection放入ctx或者开启事务,所以要尽可能的接近执行时检查
	// There must be a db Connection and transaction.It is possible to create a db Connection into ctx or open a transaction, so check as close as possible to the execution
	var dbConnectionerr error
	var dbConnection *dataBaseConnection
	ctx, dbConnection, dbConnectionerr = checkDBConnection(ctx, dbConnection, true, 1)
	if dbConnectionerr != nil {
		return nil, dbConnectionerr
	}

	var res *sql.Result
	var errexec error
	if lastInsertID != nil {
		sqlrow, errrow := dbConnection.queryRowContext(ctx, sqlstrptr, values)
		if errrow != nil {
			return res, errrow
		}
		errexec = sqlrow.Scan(lastInsertID)
		if errexec == nil { // 如果插入成功,返回
			*affected = 1
			return res, errexec
		}
	} else {
		res, errexec = dbConnection.execContext(ctx, sqlstrptr, values)
	}

	if errexec != nil {
		return res, errexec
	}
	// 影响的行数
	// Number of rows affected

	rowsAffected, errAffected := (*res).RowsAffected()
	if errAffected == nil {
		*affected, errAffected = typeConvertInt64toInt(rowsAffected)
	} else { // 如果不支持返回条数,设置位nil,影响的条数设置成-1
		*affected = -1
		errAffected = nil
	}
	return res, errAffected
}

// contextSQLHintValueKey 把sql hint放到context里使用的key
const contextSQLHintValueKey = wrapContextStringKey("contextSQLHintValueKey")

// BindContextSQLHint context中绑定sql的hint,使用这个Context的方法都会传播hint传播的语句
// hint 是完整的sql片段, 例如: hint:="/*+ XID('gs/aggregationSvc/2612341069705662465') */"
// 在第一个单词的后面拼接 hint sql,例如 select /*+ XID('gs/aggregationSvc/2612341069705662465') */ id,name from user
func BindContextSQLHint(parent context.Context, hint string) (context.Context, error) {
	if parent == nil {
		return nil, errors.New("->BindContextSQLHint-->context的parent不能为nil")
	}
	if hint == "" {
		return nil, errors.New("->BindContextSQLHint-->hint不能为空")
	}

	ctx := context.WithValue(parent, contextSQLHintValueKey, hint)
	return ctx, nil
}

// contextEnableGlobalTransactionValueKey 是否使用分布式事务放到context里使用的key
const contextEnableGlobalTransactionValueKey = wrapContextStringKey("contextEnableGlobalTransactionValueKey")

// BindContextEnableGlobalTransaction context启用分布式事务,不再自动设置,必须手动启用分布式事务,必须放到本地事务开启之前调用
func BindContextEnableGlobalTransaction(parent context.Context) (context.Context, error) {
	if parent == nil {
		return nil, errors.New("->BindContextEnableGlobalTransaction-->context的parent不能为nil")
	}
	ctx := context.WithValue(parent, contextEnableGlobalTransactionValueKey, true)
	return ctx, nil
}

// contextDisableTransactionValueKey 是否禁用事务放到context里使用的key
const contextDisableTransactionValueKey = wrapContextStringKey("contextDisableTransactionValueKey")

// BindContextDisableTransaction  context禁用事务,必须放到事务开启之前调用.用在不使用事务更新数据库的场景,强烈建议不要使用这个方法,更新数据库必须有事务!!!
func BindContextDisableTransaction(parent context.Context) (context.Context, error) {
	if parent == nil {
		return nil, errors.New("->BindContextDisableTransaction-->context的parent不能为nil")
	}
	ctx := context.WithValue(parent, contextDisableTransactionValueKey, true)
	return ctx, nil
}

/*
// contextDefaultValueKey 把属性的默认值放到context里使用的key
const contextDefaultValueKey = wrapContextStringKey("contextDefaultValueKey")

// BindContextDefaultValue 设置属性的默认值. 优先级高于 GetDefaultValue
// 默认值仅用于Insert和InsertSlice Struct,对Update和UpdateNotZeroValue无效
// defaultValueMap的key是Struct属性名,当属性值是零值时,会取值map的value,value可以是nil,不能是类型的默认值,比如int类型设置默认值为0
// ctx里bind的值zorm不会清空,使用时不要覆盖原始的ctx或者不要传给多个方法.
func BindContextDefaultValue(parent context.Context, defaultValueMap map[string]interface{}) (context.Context, error) {
	if parent == nil {
		return nil, errors.New("->BindContextDefaultValue-->context的parent不能为nil")
	}
	ctx := context.WithValue(parent, contextDefaultValueKey, defaultValueMap)
	return ctx, nil
}
*/
// contextMustUpdateColsValueKey 把仅更新的数据库字段放到context里使用的key
const contextMustUpdateColsValueKey = wrapContextStringKey("contextMustUpdateColsValueKey")

// BindContextMustUpdateCols 指定必须更新的数据库字段,只对UpdateNotZeroValue方法有效.cols是数据库列名切片
// ctx里bind的值zorm不会清空,使用时不要覆盖原始的ctx或者不要传给多个UpdateNotZeroValue方法.
func BindContextMustUpdateCols(parent context.Context, cols []string) (context.Context, error) {
	if parent == nil {
		return nil, errors.New("->BindContextMustUpdateCols-->context的parent不能为nil")
	}
	colsMap := make(map[string]bool)
	for i := 0; i < len(cols); i++ {
		colsMap[strings.ToLower(cols[i])] = true
	}
	ctx := context.WithValue(parent, contextMustUpdateColsValueKey, colsMap)
	return ctx, nil
}

// contextOnlyUpdateColsValueKey 把仅更新的数据库字段放到context里使用的key
const contextOnlyUpdateColsValueKey = wrapContextStringKey("contextOnlyUpdateColsValueKey")

// BindContextOnlyUpdateCols 指定仅更新的数据库字段,只对Update方法有效.cols是数据库列名切片
// ctx里bind的值zorm不会清空,使用时不要覆盖原始的ctx或者不要传给多个Update方法.
func BindContextOnlyUpdateCols(parent context.Context, cols []string) (context.Context, error) {
	if parent == nil {
		return nil, errors.New("->BindContextOnlyUpdateCols-->context的parent不能为nil")
	}
	colsMap := make(map[string]bool)
	for i := 0; i < len(cols); i++ {
		colsMap[strings.ToLower(cols[i])] = true
	}
	ctx := context.WithValue(parent, contextOnlyUpdateColsValueKey, colsMap)
	return ctx, nil
}

// getContextBoolValue 从ctx中获取key的bool值,ctx如果没有值使用defaultValue
func getContextBoolValue(ctx context.Context, key wrapContextStringKey, defaultValue bool) bool {
	boolValue := false
	ctxBoolValue := ctx.Value(key)
	if ctxBoolValue != nil { // 如果有值
		boolValue = ctxBoolValue.(bool)
	} else { // ctx如果没有值使用defaultValue
		boolValue = defaultValue
	}
	return boolValue
}
