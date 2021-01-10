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
}

// DataSourceConfig 数据库连接池的配置
// DateSourceConfig Database connection pool configuration
type DataSourceConfig struct {
	//DSN dataSourceName 连接字符串
	//DSN DataSourceName Database connection string
	DSN string
	//数据库驱动名称:mysql,postgres,oci8,sqlserver,sqlite3,dm,kingbase 和DBType对应,处理数据库有多个驱动
	//Database diver name:mysql,dm,postgres,opi8,sqlserver,sqlite3,kingbase corresponds to DBType,A database may have multiple drivers
	DriverName string
	//数据库类型(方言判断依据):mysql,postgresql,oracle,mssql,sqlite,dm,kingbase 和 DriverName 对应,处理数据库有多个驱动
	//Database Type:mysql,postgresql,oracle,mssql,sqlite,dm,kingbase corresponds to DriverName,A database may have multiple drivers
	DBType string
	//PrintSQL 是否打印SQL语句.使用zorm.PrintSQL记录SQL
	//PrintSQL Whether to print SQL, use zorm.PrintSQL record sql
	PrintSQL bool
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
}

// newDataSource 创建一个新的datasource,内部调用,避免外部直接使用datasource
// newDAtaSource Create a new datasource and call it internally to avoid direct external use of the datasource
func newDataSource(config *DataSourceConfig) (*dataSource, error) {
	db, err := sql.Open(config.DriverName, config.DSN)
	if err != nil {
		err = fmt.Errorf("newDataSource-->open数据库打开失败:%w", err)
		FuncLogError(err)
		return nil, err
	}

	maxOpenConns := config.MaxOpenConns
	maxIdleConns := config.MaxIdleConns
	connMaxLifetimeSecond := config.ConnMaxLifetimeSecond
	if maxOpenConns == 0 {
		maxOpenConns = 50
	}
	if maxIdleConns == 0 {
		maxIdleConns = 50
	}

	if connMaxLifetimeSecond == 0 {
		connMaxLifetimeSecond = 600
	}

	//设置数据库最大连接数
	//Set the maximum number of database connections
	db.SetMaxOpenConns(maxOpenConns)
	//设置数据库最大空闲连接数
	//Set the maximum number of free connections to the database
	db.SetMaxIdleConns(maxIdleConns)
	//连接存活秒时间. 默认600(10分钟)后连接被销毁重建.避免数据库主动断开连接,造成死连接.MySQL默认wait_timeout 28800秒(8小时)
	//(Connection survival time in seconds) Destroy and rebuild the connection after the default 600 seconds (10 minutes)
	//Prevent the database from actively disconnecting and causing dead connections. MySQL Default wait_timeout 28800 seconds
	db.SetConnMaxLifetime(time.Second * time.Duration(connMaxLifetimeSecond))

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
// 方法都应包含 dbConnection dataBaseConnection这样的入参,context必须传入,不能为空
// dataBaseConnection Database session, native query or transaction.
// The method should contain input parameters such as dbConnection dataBaseConnection,and the "context" must be passed in and cannot be empty.
type dataBaseConnection struct {

	// 原生db
	// native db
	db *sql.DB
	// 原生事务
	// native Transaction
	tx *sql.Tx
	//数据库驱动名称:mysql,postgres,oci8,sqlserver,sqlite3,dm,kingbase 和DBType对应,处理数据库有多个驱动
	//Database diver name:mysql,dm,postgres,opi8,sqlserver,sqlite3,kingbase corresponds to DBType,A database may have multiple drivers
	driverName string
	//数据库类型(方言判断依据):mysql,postgresql,oracle,mssql,sqlite,dm,kingbase 和 DriverName 对应,处理数据库有多个驱动
	//Database Type:mysql,postgresql,oracle,mssql,sqlite,dm,kingbase corresponds to DriverName,A database may have multiple drivers
	dbType string

	//是否打印sql
	//Whether to print SQL, use zorm.PrintSQL record sql
	printSQL bool

	//commitSign   int8    // 提交标记,控制是否提交事务
	//rollbackSign bool    // 回滚标记,控制是否回滚事务
}

// beginTx 开启事务
// beginTx Open transaction
func (dbConnection *dataBaseConnection) beginTx(ctx context.Context) error {
	//s.rollbackSign = true
	if dbConnection.tx == nil {
		tx, err := dbConnection.db.BeginTx(ctx, nil)
		if err != nil {
			err = fmt.Errorf("beginTx事务开启失败:%w", err)
			//ZormErrorLog(err)
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
			//ZormErrorLog(err)
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

	//打印SQL
	//print SQL
	if dbConnection.printSQL {
		//logger.Info("printSQL", logger.String("sql", execsql), logger.Any("args", args))
		FuncPrintSQL(*execsql, args)
	}

	if dbConnection.tx != nil {
		res, reserr := dbConnection.tx.ExecContext(ctx, *execsql, args...)
		return &res, reserr
	}
	res, reserr := dbConnection.db.ExecContext(ctx, *execsql, args...)
	return &res, reserr
}

// queryRowContext 如果已经开启事务,就以事务方式执行,如果没有开启事务,就以非事务方式执行
func (dbConnection *dataBaseConnection) queryRowContext(ctx context.Context, query *string, args ...interface{}) *sql.Row {
	//打印SQL
	if dbConnection.printSQL {
		//logger.Info("printSQL", logger.String("sql", query), logger.Any("args", args))
		FuncPrintSQL(*query, args)
	}

	if dbConnection.tx != nil {
		return dbConnection.tx.QueryRowContext(ctx, *query, args...)
	}
	return dbConnection.db.QueryRowContext(ctx, *query, args...)
}

// queryContext 查询数据,如果已经开启事务,就以事务方式执行,如果没有开启事务,就以非事务方式执行
// queryRowContext Execute sql  row statement,If the transaction has been opened,it will be executed in transaction mode, if the transaction is not opened,it will be executed in non-transactional mode
func (dbConnection *dataBaseConnection) queryContext(ctx context.Context, query *string, args ...interface{}) (*sql.Rows, error) {
	//打印SQL
	if dbConnection.printSQL {
		//logger.Info("printSQL", logger.String("sql", query), logger.Any("args", args))
		FuncPrintSQL(*query, args)
	}

	if dbConnection.tx != nil {
		return dbConnection.tx.QueryContext(ctx, *query, args...)
	}
	return dbConnection.db.QueryContext(ctx, *query, args...)
}

// prepareContext 预执行,如果已经开启事务,就以事务方式执行,如果没有开启事务,就以非事务方式执行
// prepareContext Pre-execution,If the transaction has been opened,it will be executed in transaction mode,if the transaction is not opened,it will be executed in non-transactional mode
func (dbConnection *dataBaseConnection) prepareContext(ctx context.Context, query *string) (*sql.Stmt, error) {
	//打印SQL
	//print SQL
	if dbConnection.printSQL {
		//logger.Info("printSQL", logger.String("sql", query))
		FuncPrintSQL(*query, nil)
	}

	if dbConnection.tx != nil {
		return dbConnection.tx.PrepareContext(ctx, *query)
	}

	return dbConnection.db.PrepareContext(ctx, *query)
}
