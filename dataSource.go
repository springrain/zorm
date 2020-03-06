package zorm

import (
	"database/sql"
	"errors"
	"fmt"
	"readygo/logger"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// dataSorce对象,隔离mysql原生对象
type dataSource struct {
	*sql.DB
}

//DataSourceConfig 数据库连接池的配置
type DataSourceConfig struct {
	Host     string
	Port     int
	DBName   string
	UserName string
	PassWord string
	//mysql,postgres,oci8,adodb
	DBType string
}

//newDataSource 创建一个新的datasource,内部调用,避免外部直接使用datasource
func newDataSource(config *DataSourceConfig) (*dataSource, error) {
	dsn, e := wrapDBDSN(config)
	if e != nil {
		e = fmt.Errorf("获取数据库连接字符串失败:%w", e)
		logger.Error(e)
		return nil, e
	}

	db, err := sql.Open(config.DBType, dsn)
	if err != nil {
		err = fmt.Errorf("数据库打开失败:%w", err)
		logger.Error(err)
		return nil, err
	}

	//设置数据库最大连接数
	db.SetMaxOpenConns(50)
	//设置数据库最大空闲连接数
	db.SetMaxIdleConns(50)
	//连接存活10分钟,10分钟后连接被销毁重建.避免数据库断开连接,造成死连接.MySQL默认wait_timeout 28800秒(8小时)
	db.SetConnMaxLifetime(time.Minute * 10)

	//验证连接
	if pingerr := db.Ping(); pingerr != nil {
		pingerr = fmt.Errorf("ping数据库失败:%w", pingerr)
		logger.Error(pingerr)
		return nil, pingerr
	}

	return &dataSource{db}, nil
}

//事务参照:https://www.jianshu.com/p/2a144332c3db

//const beginStatus = 1

//DBConnection 数据库dbConnection会话,可以原生查询或者事务
//方法都应包含 dbConnection DBConnection这样的入参,如果有dbConnection就传入,不考虑从哪获得的.如果在上下文中找不到dbConnection,就传入nil,会新建dbConnection,传nil要谨慎啊
type DBConnection struct {
	db *sql.DB // 原生db
	tx *sql.Tx // 原生事务
	//mysql,postgres,oci8,adodb
	dbType string

	//commitSign   int8    // 提交标记，控制是否提交事务
	//rollbackSign bool    // 回滚标记，控制是否回滚事务
}

// begin 开启事务
func (dbConnection *DBConnection) begin() error {
	//s.rollbackSign = true
	if dbConnection.tx == nil {
		tx, err := dbConnection.db.Begin()
		if err != nil {
			err = fmt.Errorf("事务开启失败:%w", err)
			//logger.Error(err)
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
func (dbConnection *DBConnection) rollback() error {
	//if s.tx != nil && s.rollbackSign == true {
	if dbConnection.tx != nil {
		err := dbConnection.tx.Rollback()
		if err != nil {
			err = fmt.Errorf("事务回滚失败:%w", err)
			//logger.Error(err)
			return err
		}
		dbConnection.tx = nil
		return nil
	}
	return nil
}

// commit 提交事务
func (dbConnection *DBConnection) commit() error {
	//s.rollbackSign = false
	if dbConnection.tx == nil {
		return errors.New("事务为空")

	}
	err := dbConnection.tx.Commit()
	if err != nil {
		err = fmt.Errorf("事务提交失败:%w", err)
		//logger.Error(err)
		return err
	}
	dbConnection.tx = nil
	return nil

}

// exec 执行sql语句，如果已经开启事务，就以事务方式执行，如果没有开启事务，就以非事务方式执行
func (dbConnection *DBConnection) exec(query string, args ...interface{}) (sql.Result, error) {
	if dbConnection.tx != nil {
		return dbConnection.tx.Exec(query, args...)
	}
	return dbConnection.db.Exec(query, args...)
}

// queryRow 如果已经开启事务，就以事务方式执行，如果没有开启事务，就以非事务方式执行
func (dbConnection *DBConnection) queryRow(query string, args ...interface{}) *sql.Row {
	if dbConnection.tx != nil {
		return dbConnection.tx.QueryRow(query, args...)
	}
	return dbConnection.db.QueryRow(query, args...)
}

// query 查询数据，如果已经开启事务，就以事务方式执行，如果没有开启事务，就以非事务方式执行
func (dbConnection *DBConnection) query(query string, args ...interface{}) (*sql.Rows, error) {
	if dbConnection.tx != nil {
		return dbConnection.tx.Query(query, args...)
	}
	return dbConnection.db.Query(query, args...)
}

// prepare 预执行，如果已经开启事务，就以事务方式执行，如果没有开启事务，就以非事务方式执行
func (dbConnection *DBConnection) prepare(query string) (*sql.Stmt, error) {
	if dbConnection.tx != nil {
		return dbConnection.tx.Prepare(query)
	}

	return dbConnection.db.Prepare(query)
}
