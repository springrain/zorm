package zorm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// dataSorce对象,隔离sql原生对象
// dataSorce  Isolate sql native objects
type dataSource struct {
	*sql.DB
	//config *DataSourceConfig
}

// DataSourceConfig 数据库连接池的配置
// DateSourceConfig Database connection pool configuration
type DataSourceConfig struct {
	//DSN dataSourceName 连接字符串
	//DSN DataSourceName Database connection string
	DSN string
	//数据库驱动名称:mysql,postgres,oci8,sqlserver,sqlite3,clickhouse,dm,kingbase,aci,taosSql|taosRestful 和DBType对应,处理数据库有多个驱动
	//Database diver name:mysql,dm,postgres,opi8,sqlserver,sqlite3,clickhouse,kingbase,aci,taosSql|taosRestful corresponds to DBType,A database may have multiple drivers
	DriverName string
	//数据库类型(方言判断依据):mysql,postgresql,oracle,mssql,sqlite,clickhouse,dm,kingbase,shentong,tdengine 和 DriverName 对应,处理数据库有多个驱动
	//Database Type:mysql,postgresql,oracle,mssql,sqlite,clickhouse,dm,kingbase,shentong,tdengine corresponds to DriverName,A database may have multiple drivers
	DBType string
	//SlowSQLMillis 慢sql的时间阈值,单位毫秒.小于0是禁用SQL语句输出;等于0是只输出SQL语句,不计算执行时间;大于0是计算SQL执行时间,并且>=SlowSQLMillis值
	SlowSQLMillis int
	//MaxOpenConns 数据库最大连接数,默认50
	//MaxOpenConns Maximum number of database connections, Default 50
	MaxOpenConns int
	//MaxIdleConns 数据库最大空闲连接数,默认50
	//MaxIdleConns The maximum number of free connections to the database default 50
	MaxIdleConns int
	//ConnMaxLifetimeSecond 连接存活秒时间. 默认600(10分钟)后连接被销毁重建.避免数据库主动断开连接,造成死连接.MySQL默认wait_timeout 28800秒(8小时)
	//ConnMaxLifetimeSecond: (Connection survival time in seconds)Destroy and rebuild the connection after the default 600 seconds (10 minutes)
	//Prevent the database from actively disconnecting and causing dead connections. MySQL Default wait_timeout 28800 seconds
	ConnMaxLifetimeSecond int

	//DefaultTxOptions 事务隔离级别的默认配置,默认为nil
	DefaultTxOptions *sql.TxOptions

	//DisableTransaction 全局禁用事务,默认false,如果设置了DisableTransaction=true,Transaction方法失效,不再要求有事务.为了处理某些数据库不支持事务,比如TDengine
	//禁用事务应该有驱动伪造事务API,不应该由orm实现
	DisableTransaction bool

	//MockSQLDB 用于mock测试的入口,如果MockSQLDB不为nil,则不使用DSN,直接使用MockSQLDB
	//db, mock, err := sqlmock.New()
	//MockSQLDB *sql.DB

	//FuncGlobalTransaction seata/hptx全局分布式事务的适配函数,返回IGlobalTransaction接口的实现
	//业务必须调用zorm.BindContextEnableGlobalTransaction(ctx)开启全局分布事务
	FuncGlobalTransaction func(ctx context.Context) (IGlobalTransaction, context.Context, error)
	//DisableAutoGlobalTransaction 禁用自动全局分布式事务,默认false,虽然设置了FuncGlobalTransaction,但是并不想全部业务自动开启全局事务
	//DisableAutoGlobalTransaction = false; ctx,_=zorm.BindContextEnableGlobalTransaction(ctx,false) 默认使用全局事务,ctx绑定为false才不开启
	//DisableAutoGlobalTransaction = true;  ctx,_=zorm.BindContextEnableGlobalTransaction(ctx,true) 默认禁用全局事务,ctx绑定为true才开启
	//DisableAutoGlobalTransaction bool

	//SQLDB 使用现有的数据库连接,优先级高于DSN
	SQLDB *sql.DB
}

// newDataSource 创建一个新的datasource,内部调用,避免外部直接使用datasource
// newDAtaSource Create a new datasource and call it internally to avoid direct external use of the datasource
func newDataSource(config *DataSourceConfig) (*dataSource, error) {

	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	if config.DriverName == "" {
		return nil, errors.New("DriverName cannot be empty")
	}
	if config.DBType == "" {
		return nil, errors.New("DBType cannot be empty")
	}
	var db *sql.DB
	var errSQLOpen error

	if config.SQLDB == nil { //没有已经存在的数据库连接,使用DSN初始化
		if config.DSN == "" {
			return nil, errors.New("DSN cannot be empty")
		}
		db, errSQLOpen = sql.Open(config.DriverName, config.DSN)
		if errSQLOpen != nil {
			errSQLOpen = fmt.Errorf("newDataSource-->open数据库打开失败:%w", errSQLOpen)
			FuncLogError(errSQLOpen)
			return nil, errSQLOpen
		}
	} else { //使用已经存在的数据库连接
		db = config.SQLDB
	}

	if config.MaxOpenConns == 0 {
		config.MaxOpenConns = 50
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 50
	}

	if config.ConnMaxLifetimeSecond == 0 {
		config.ConnMaxLifetimeSecond = 600
	}

	//设置数据库最大连接数
	//Set the maximum number of database connections
	db.SetMaxOpenConns(config.MaxOpenConns)
	//设置数据库最大空闲连接数
	//Set the maximum number of free connections to the database
	db.SetMaxIdleConns(config.MaxIdleConns)
	//连接存活秒时间. 默认600(10分钟)后连接被销毁重建.避免数据库主动断开连接,造成死连接.MySQL默认wait_timeout 28800秒(8小时)
	//(Connection survival time in seconds) Destroy and rebuild the connection after the default 600 seconds (10 minutes)
	//Prevent the database from actively disconnecting and causing dead connections. MySQL Default wait_timeout 28800 seconds
	db.SetConnMaxLifetime(time.Second * time.Duration(config.ConnMaxLifetimeSecond))

	//验证连接
	if pingerr := db.Ping(); pingerr != nil {
		pingerr = fmt.Errorf("newDataSource-->ping数据库失败:%w", pingerr)
		FuncLogError(pingerr)
		db.Close()
		return nil, pingerr
	}

	return &dataSource{db}, nil
}

// 事务参照:https://www.jianshu.com/p/2a144332c3db
//Transaction reference: https://www.jianshu.com/p/2a144332c3db

// const beginStatus = 1

// dataBaseConnection 数据库dbConnection会话,可以原生查询或者事务
// dataBaseConnection Database session, native query or transaction.
type dataBaseConnection struct {

	// 原生db
	// native db
	db *sql.DB
	// 原生事务
	// native Transaction
	tx *sql.Tx
	// 数据库配置
	config *DataSourceConfig

	//commitSign   int8    // 提交标记,控制是否提交事务
	//rollbackSign bool    // 回滚标记,控制是否回滚事务
}

// beginTx 开启事务
// beginTx Open transaction
func (dbConnection *dataBaseConnection) beginTx(ctx context.Context) error {
	//s.rollbackSign = true
	if dbConnection.tx == nil {

		//设置事务配置,主要是隔离级别
		var txOptions *sql.TxOptions
		contextTxOptions := ctx.Value(contextTxOptionsKey)
		if contextTxOptions != nil {
			txOptions, _ = contextTxOptions.(*sql.TxOptions)
		} else {
			txOptions = dbConnection.config.DefaultTxOptions
		}

		tx, err := dbConnection.db.BeginTx(ctx, txOptions)
		if err != nil {
			err = fmt.Errorf("beginTx事务开启失败:%w", err)
			return err
		}
		dbConnection.tx = tx
		//s.commitSign = beginStatus
		return nil
	}
	//s.commitSign++
	return nil
}

// rollback 回滚事务
// rollback Rollback transaction
func (dbConnection *dataBaseConnection) rollback() error {
	//if s.tx != nil && s.rollbackSign == true {
	if dbConnection.tx != nil {
		err := dbConnection.tx.Rollback()
		if err != nil {
			err = fmt.Errorf("rollback事务回滚失败:%w", err)
			return err
		}
		dbConnection.tx = nil
		return nil
	}
	return nil
}

// commit 提交事务
// commit Commit transaction
func (dbConnection *dataBaseConnection) commit() error {
	//s.rollbackSign = false
	if dbConnection.tx == nil {
		return errors.New("commit事务为空")

	}
	err := dbConnection.tx.Commit()
	if err != nil {
		err = fmt.Errorf("commit事务提交失败:%w", err)
		return err
	}
	dbConnection.tx = nil
	return nil

}

// execContext 执行sql语句,如果已经开启事务,就以事务方式执行,如果没有开启事务,就以非事务方式执行
// execContext Execute sql statement,If the transaction has been opened,it will be executed in transaction mode, if the transaction is not opened,it will be executed in non-transactional mode
func (dbConnection *dataBaseConnection) execContext(ctx context.Context, execsql *string, args []interface{}) (*sql.Result, error) {
	var err error
	//如果是TDengine,重新处理 字符类型的参数 '?'
	execsql, err = reBindSQL(dbConnection.config.DBType, execsql, args)
	if err != nil {
		return nil, err
	}
	// 更新语句处理ClickHouse特殊语法
	execsql, err = reUpdateSQL(dbConnection.config.DBType, execsql)
	if err != nil {
		return nil, err
	}
	//执行前加入 hint
	execsql, err = wrapSQLHint(ctx, execsql)
	if err != nil {
		return nil, err
	}
	var start *time.Time
	var res sql.Result
	//小于0是禁用日志输出;等于0是只输出日志,不计算SQ执行时间;大于0是计算执行时间,并且大于指定值
	if dbConnection.config.SlowSQLMillis == 0 {
		//logger.Info("printSQL", logger.String("sql", execsql), logger.Any("args", args))
		FuncPrintSQL(*execsql, args, 0)
	} else if dbConnection.config.SlowSQLMillis > 0 {
		now := time.Now() // 获取当前时间
		start = &now
	}
	if dbConnection.tx != nil {
		res, err = dbConnection.tx.ExecContext(ctx, *execsql, args...)
	} else {
		res, err = dbConnection.db.ExecContext(ctx, *execsql, args...)
	}
	if dbConnection.config.SlowSQLMillis > 0 {
		slow := time.Since(*start).Milliseconds()
		if slow-int64(dbConnection.config.SlowSQLMillis) >= 0 {
			FuncPrintSQL(*execsql, args, slow)
		}
	}

	return &res, err
}

// queryRowContext 如果已经开启事务,就以事务方式执行,如果没有开启事务,就以非事务方式执行
func (dbConnection *dataBaseConnection) queryRowContext(ctx context.Context, query *string, args []interface{}) (*sql.Row, error) {
	var err error
	//如果是TDengine,重新处理 字符类型的参数 '?'
	query, err = reBindSQL(dbConnection.config.DBType, query, args)
	if err != nil {
		return nil, err
	}
	//执行前加入 hint
	query, err = wrapSQLHint(ctx, query)
	if err != nil {
		return nil, err
	}
	var start *time.Time
	var row *sql.Row
	//小于0是禁用日志输出;等于0是只输出日志,不计算SQ执行时间;大于0是计算执行时间,并且大于指定值
	if dbConnection.config.SlowSQLMillis == 0 {
		//logger.Info("printSQL", logger.String("sql", query), logger.Any("args", args))
		FuncPrintSQL(*query, args, 0)
	} else if dbConnection.config.SlowSQLMillis > 0 {
		now := time.Now() // 获取当前时间
		start = &now
	}

	if dbConnection.tx != nil {
		row = dbConnection.tx.QueryRowContext(ctx, *query, args...)
	} else {
		row = dbConnection.db.QueryRowContext(ctx, *query, args...)
	}
	if dbConnection.config.SlowSQLMillis > 0 {
		slow := time.Since(*start).Milliseconds()
		if slow-int64(dbConnection.config.SlowSQLMillis) >= 0 {
			FuncPrintSQL(*query, args, slow)
		}
	}
	return row, nil
}

// queryContext 查询数据,如果已经开启事务,就以事务方式执行,如果没有开启事务,就以非事务方式执行
// queryRowContext Execute sql  row statement,If the transaction has been opened,it will be executed in transaction mode, if the transaction is not opened,it will be executed in non-transactional mode
func (dbConnection *dataBaseConnection) queryContext(ctx context.Context, query *string, args []interface{}) (*sql.Rows, error) {
	var err error
	//如果是TDengine,重新处理 字符类型的参数 '?'
	query, err = reBindSQL(dbConnection.config.DBType, query, args)
	if err != nil {
		return nil, err
	}
	//执行前加入 hint
	query, err = wrapSQLHint(ctx, query)
	if err != nil {
		return nil, err
	}
	var start *time.Time
	var rows *sql.Rows
	//小于0是禁用日志输出;等于0是只输出日志,不计算SQ执行时间;大于0是计算执行时间,并且大于指定值
	if dbConnection.config.SlowSQLMillis == 0 {
		//logger.Info("printSQL", logger.String("sql", query), logger.Any("args", args))
		FuncPrintSQL(*query, args, 0)
	} else if dbConnection.config.SlowSQLMillis > 0 {
		now := time.Now() // 获取当前时间
		start = &now
	}

	if dbConnection.tx != nil {
		rows, err = dbConnection.tx.QueryContext(ctx, *query, args...)
	} else {
		rows, err = dbConnection.db.QueryContext(ctx, *query, args...)
	}
	if dbConnection.config.SlowSQLMillis > 0 {
		slow := time.Since(*start).Milliseconds()
		if slow-int64(dbConnection.config.SlowSQLMillis) >= 0 {
			FuncPrintSQL(*query, args, slow)
		}
	}
	return rows, err
}

/*
// prepareContext 预执行,如果已经开启事务,就以事务方式执行,如果没有开启事务,就以非事务方式执行
// prepareContext Pre-execution,If the transaction has been opened,it will be executed in transaction mode,if the transaction is not opened,it will be executed in non-transactional mode
func (dbConnection *dataBaseConnection) prepareContext(ctx context.Context, query *string) (*sql.Stmt, error) {
	//打印SQL
	//print SQL
	if dbConnection.config.PrintSQL {
		//logger.Info("printSQL", logger.String("sql", query))
		FuncPrintSQL(*query, nil)
	}

	if dbConnection.tx != nil {
		return dbConnection.tx.PrepareContext(ctx, *query)
	}

	return dbConnection.db.PrepareContext(ctx, *query)
}
*/
