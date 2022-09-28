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

// newDataSource 创建一个新的datasource,内部调用,避免外部直接使用datasource
// newDAtaSource Create a new datasource and call it internally to avoid direct external use of the datasource
func newDataSource(config *DataSourceConfig) (*dataSource, error) {

	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	if config.DriverName == "" {
		return nil, errors.New("DriverName cannot be empty")
	}
	//兼容处理,DBType即将废弃,请使用Dialect属性
	if len(config.DBType) > 0 && len(config.Dialect) == 0 {
		config.Dialect = config.DBType
	}
	if config.Dialect == "" {
		return nil, errors.New("Dialect cannot be empty")
	}
	var db *sql.DB
	var errSQLOpen error

	if config.SQLDB == nil { //没有已经存在的数据库连接,使用DSN初始化
		if config.DSN == "" {
			return nil, errors.New("DSN cannot be empty")
		}
		db, errSQLOpen = sql.Open(config.DriverName, config.DSN)
		if errSQLOpen != nil {
			errSQLOpen = fmt.Errorf("->newDataSource-->open数据库打开失败:%w", errSQLOpen)
			FuncLogError(nil, errSQLOpen)
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
		pingerr = fmt.Errorf("->newDataSource-->ping数据库失败:%w", pingerr)
		FuncLogError(nil, pingerr)
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
			err = fmt.Errorf("->beginTx事务开启失败:%w", err)
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
			err = fmt.Errorf("->rollback事务回滚失败:%w", err)
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
		err = fmt.Errorf("->commit事务提交失败:%w", err)
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
	err = reBindSQL(dbConnection.config.Dialect, execsql, args)
	if err != nil {
		return nil, err
	}
	// 更新语句处理ClickHouse特殊语法
	err = reUpdateSQL(dbConnection.config.Dialect, execsql)
	if err != nil {
		return nil, err
	}
	//执行前加入 hint
	err = wrapSQLHint(ctx, execsql)
	if err != nil {
		return nil, err
	}
	var start *time.Time
	var res sql.Result
	//小于0是禁用日志输出;等于0是只输出日志,不计算SQ执行时间;大于0是计算执行时间,并且大于指定值
	if dbConnection.config.SlowSQLMillis == 0 {
		//logger.Info("printSQL", logger.String("sql", execsql), logger.Any("args", args))
		FuncPrintSQL(ctx, *execsql, args, 0)
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
			FuncPrintSQL(ctx, *execsql, args, slow)
		}
	}

	return &res, err
}

// queryRowContext 如果已经开启事务,就以事务方式执行,如果没有开启事务,就以非事务方式执行
func (dbConnection *dataBaseConnection) queryRowContext(ctx context.Context, query *string, args []interface{}) (*sql.Row, error) {
	var err error
	//如果是TDengine,重新处理 字符类型的参数 '?'
	err = reBindSQL(dbConnection.config.Dialect, query, args)
	if err != nil {
		return nil, err
	}
	//执行前加入 hint
	err = wrapSQLHint(ctx, query)
	if err != nil {
		return nil, err
	}
	var start *time.Time
	var row *sql.Row
	//小于0是禁用日志输出;等于0是只输出日志,不计算SQ执行时间;大于0是计算执行时间,并且大于指定值
	if dbConnection.config.SlowSQLMillis == 0 {
		//logger.Info("printSQL", logger.String("sql", query), logger.Any("args", args))
		FuncPrintSQL(ctx, *query, args, 0)
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
			FuncPrintSQL(ctx, *query, args, slow)
		}
	}
	return row, nil
}

// queryContext 查询数据,如果已经开启事务,就以事务方式执行,如果没有开启事务,就以非事务方式执行
// queryRowContext Execute sql  row statement,If the transaction has been opened,it will be executed in transaction mode, if the transaction is not opened,it will be executed in non-transactional mode
func (dbConnection *dataBaseConnection) queryContext(ctx context.Context, query *string, args []interface{}) (*sql.Rows, error) {
	var err error
	//如果是TDengine,重新处理 字符类型的参数 '?'
	err = reBindSQL(dbConnection.config.Dialect, query, args)
	if err != nil {
		return nil, err
	}
	//执行前加入 hint
	err = wrapSQLHint(ctx, query)
	if err != nil {
		return nil, err
	}
	var start *time.Time
	var rows *sql.Rows
	//小于0是禁用日志输出;等于0是只输出日志,不计算SQ执行时间;大于0是计算执行时间,并且大于指定值
	if dbConnection.config.SlowSQLMillis == 0 {
		//logger.Info("printSQL", logger.String("sql", query), logger.Any("args", args))
		FuncPrintSQL(ctx, *query, args, 0)
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
			FuncPrintSQL(ctx, *query, args, slow)
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
		FuncPrintSQL(ctx,*query, nil)
	}

	if dbConnection.tx != nil {
		return dbConnection.tx.PrepareContext(ctx, *query)
	}

	return dbConnection.db.PrepareContext(ctx, *query)
}
*/
