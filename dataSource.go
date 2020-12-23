package zorm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// dataSorce对象,隔离sql原生对象
type dataSource struct {
	*sql.DB
}

//DataSourceConfig 数据库连接池的配置
type DataSourceConfig struct {
	//DSN dataSourceName 连接字符串
	DSN string
	//数据库驱动名称:mysql,postgres,oci8,sqlserver,sqlite3,dm,kingbase 和DBType对应,处理数据库有多个驱动
	DriverName string
	//数据库类型(方言判断依据):mysql,postgresql,oracle,mssql,sqlite,dm,kingbase 和 DriverName 对应,处理数据库有多个驱动
	DBType string
	//PrintSQL 是否打印SQL语句.使用zorm.ZormPrintSQL记录SQL
	PrintSQL bool
	//MaxOpenConns 数据库最大连接数 默认50
	MaxOpenConns int
	//MaxIdleConns 数据库最大空闲连接数 默认50
	MaxIdleConns int
	//ConnMaxLifetimeSecond 连接存活秒时间. 默认600(10分钟)后连接被销毁重建.避免数据库主动断开连接,造成死连接.MySQL默认wait_timeout 28800秒(8小时)
	ConnMaxLifetimeSecond int
}

//newDataSource 创建一个新的datasource,内部调用,避免外部直接使用datasource
func newDataSource(config *DataSourceConfig) (*dataSource, error) {
	/*
		dsn, e := wrapDBDSN(config)
		if e != nil {
			e = fmt.Errorf("获取数据库连接字符串失败:%w", e)
			ZormErrorLog(e)
			return nil, e
		}
	*/
	db, err := sql.Open(config.DriverName, config.DSN)
	if err != nil {
		err = fmt.Errorf("数据库打开失败:%w", err)
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
	db.SetMaxOpenConns(maxOpenConns)
	//设置数据库最大空闲连接数
	db.SetMaxIdleConns(maxIdleConns)
	//连接存活秒时间. 默认600(10分钟)后连接被销毁重建.避免数据库主动断开连接,造成死连接.MySQL默认wait_timeout 28800秒(8小时)
	db.SetConnMaxLifetime(time.Second * time.Duration(connMaxLifetimeSecond))

	//验证连接
	if pingerr := db.Ping(); pingerr != nil {
		pingerr = fmt.Errorf("ping数据库失败:%w", pingerr)
		FuncLogError(pingerr)
		return nil, pingerr
	}

	return &dataSource{db}, nil
}

//事务参照:https://www.jianshu.com/p/2a144332c3db

//const beginStatus = 1

//dataBaseConnection 数据库dbConnection会话,可以原生查询或者事务
//方法都应包含 dbConnection dataBaseConnection这样的入参,context必须传入,不能为空
type dataBaseConnection struct {
	db *sql.DB // 原生db
	tx *sql.Tx // 原生事务
	//数据库驱动名称:mysql,postgres,oci8,sqlserver,sqlite3,dm,kingbase 和DBType对应,处理数据库有多个驱动
	driverName string
	//数据库类型(方言判断依据):mysql,postgresql,oracle,mssql,sqlite,dm,kingbase 和 DriverName 对应,处理数据库有多个驱动
	dbType string

	//是否打印sql
	printSQL bool

	//commitSign   int8    // 提交标记，控制是否提交事务
	//rollbackSign bool    // 回滚标记，控制是否回滚事务
}

// beginTx 开启事务
func (dbConnection *dataBaseConnection) beginTx(ctx context.Context) error {
	//s.rollbackSign = true
	if dbConnection.tx == nil {
		tx, err := dbConnection.db.BeginTx(ctx, nil)
		if err != nil {
			err = fmt.Errorf("事务开启失败:%w", err)
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
func (dbConnection *dataBaseConnection) rollback() error {
	//if s.tx != nil && s.rollbackSign == true {
	if dbConnection.tx != nil {
		err := dbConnection.tx.Rollback()
		if err != nil {
			err = fmt.Errorf("事务回滚失败:%w", err)
			//ZormErrorLog(err)
			return err
		}
		dbConnection.tx = nil
		return nil
	}
	return nil
}

// commit 提交事务
func (dbConnection *dataBaseConnection) commit() error {
	//s.rollbackSign = false
	if dbConnection.tx == nil {
		return errors.New("事务为空")

	}
	err := dbConnection.tx.Commit()
	if err != nil {
		err = fmt.Errorf("事务提交失败:%w", err)
		//ZormErrorLog(err)
		return err
	}
	dbConnection.tx = nil
	return nil

}

// execContext 执行sql语句，如果已经开启事务，就以事务方式执行，如果没有开启事务，就以非事务方式执行
func (dbConnection *dataBaseConnection) execContext(ctx context.Context, execsql string, args ...interface{}) (sql.Result, error) {

	//打印SQL
	if dbConnection.printSQL {
		//logger.Info("printSQL", logger.String("sql", execsql), logger.Any("args", args))
		FuncPrintSQL(execsql, args)
	}

	if dbConnection.tx != nil {
		return dbConnection.tx.ExecContext(ctx, execsql, args...)
	}
	return dbConnection.db.ExecContext(ctx, execsql, args...)
}

// queryRowContext 如果已经开启事务，就以事务方式执行，如果没有开启事务，就以非事务方式执行
func (dbConnection *dataBaseConnection) queryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	//打印SQL
	if dbConnection.printSQL {
		//logger.Info("printSQL", logger.String("sql", query), logger.Any("args", args))
		FuncPrintSQL(query, args)
	}

	if dbConnection.tx != nil {
		return dbConnection.tx.QueryRowContext(ctx, query, args...)
	}
	return dbConnection.db.QueryRowContext(ctx, query, args...)
}

// queryContext 查询数据，如果已经开启事务，就以事务方式执行，如果没有开启事务，就以非事务方式执行
func (dbConnection *dataBaseConnection) queryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	//打印SQL
	if dbConnection.printSQL {
		//logger.Info("printSQL", logger.String("sql", query), logger.Any("args", args))
		FuncPrintSQL(query, args)
	}

	if dbConnection.tx != nil {
		return dbConnection.tx.QueryContext(ctx, query, args...)
	}
	return dbConnection.db.QueryContext(ctx, query, args...)
}

// prepareContext 预执行，如果已经开启事务，就以事务方式执行，如果没有开启事务，就以非事务方式执行
func (dbConnection *dataBaseConnection) prepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	//打印SQL
	if dbConnection.printSQL {
		//logger.Info("printSQL", logger.String("sql", query))
		FuncPrintSQL(query, nil)
	}

	if dbConnection.tx != nil {
		return dbConnection.tx.PrepareContext(ctx, query)
	}

	return dbConnection.db.PrepareContext(ctx, query)
}
