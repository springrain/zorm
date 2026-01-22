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

package zorm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// stmtSQLCacheMap 缓存预编译SQL的map.会增加维护的复杂度,暂不处理
// stmtSQLCacheMap Cache precompiled SQL map
//var stmtSQLCacheMap = sync.Map{}

// dataSorce对象,隔离sql原生对象
// dataSorce  Isolate sql native objects
type dataSource struct {
	*sql.DB
	// config *DataSourceConfig
}

// dbDialectMap 数据库方言的map,用来检查Dialect设置是否正确
// dbDialectMap is a map of database dialects, used to check whether the Dialect setting is correct
var dbDialectMap = map[string]string{
	"mysql":      "mysql",
	"postgresql": "postgres",
	"oracle":     "oracle",
	"mssql":      "sqlserver",
	"sqlite":     "sqlite3",
	"db2":        "go_ibm_db",
	"clickhouse": "clickhouse",
	"dm":         "dm",
	"kingbase":   "kingbase",
	"shentong":   "aci",
	"tdengine":   "taosSql",
}

// newDataSource 创建一个新的datasource,内部调用,避免外部直接使用datasource
// newDAtaSource Create a new datasource and call it internally to avoid direct external use of the datasource
func newDataSource(config *DataSourceConfig) (*dataSource, error) {
	if config == nil {
		return nil, errors.New("->newDataSource-->config cannot be nil")
	}

	if config.DriverName == "" {
		return nil, errors.New("->newDataSource-->DriverName cannot be empty")
	}

	if config.Dialect == "" {
		return nil, errors.New("->newDataSource-->Dialect cannot be empty")
	}

	_, has := dbDialectMap[config.Dialect]
	if !has {
		return nil, errors.New("->newDataSource-->Dialect not supported, please check the Dialect configuration. Supported Dialect include mysql,postgresql,oracle,mssql,sqlite,db2,clickhouse,dm,kingbase,shentong,tdengine ")
	}

	var db *sql.DB
	var errSQLOpen error

	if config.SQLDB == nil { // 没有已经存在的数据库连接,使用DSN初始化
		if config.DSN == "" {
			return nil, errors.New("->newDataSource-->DSN cannot be empty")
		}
		db, errSQLOpen = sql.Open(config.DriverName, config.DSN)
		if errSQLOpen != nil {
			errSQLOpen = fmt.Errorf("->newDataSource-->open数据库打开失败:%w", errSQLOpen)
			//FuncLogError(nil, errSQLOpen)
			return nil, errSQLOpen
		}
	} else { // 使用已经存在的数据库连接
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

	// 设置数据库最大连接数
	// Set the maximum number of database connections
	db.SetMaxOpenConns(config.MaxOpenConns)
	// 设置数据库最大空闲连接数
	// Set the maximum number of free connections to the database
	db.SetMaxIdleConns(config.MaxIdleConns)
	//连接存活秒时间. 默认600(10分钟)后连接被销毁重建.避免数据库主动断开连接,造成死连接.MySQL默认wait_timeout 28800秒(8小时)
	//(Connection survival time in seconds) Destroy and rebuild the connection after the default 600 seconds (10 minutes)
	//Prevent the database from actively disconnecting and causing dead connections. MySQL Default wait_timeout 28800 seconds
	db.SetConnMaxLifetime(time.Second * time.Duration(config.ConnMaxLifetimeSecond))

	// 验证连接
	if err := db.Ping(); err != nil {
		err = fmt.Errorf("->newDataSource-->ping数据库失败:%w", err)
		//FuncLogError(nil, err)
		db.Close()
		return nil, err
	}

	return &dataSource{db}, nil
}

// 事务参照:https://www.jianshu.com/p/2a144332c3db
// Transaction reference: https://www.jianshu.com/p/2a144332c3db

// dataBaseConnection 数据库dbConnection会话,可以原生查询或者事务
// dataBaseConnection Database session, native query or transaction.
type dataBaseConnection struct {
	// 原生db
	// native db
	db *sql.DB

	// 原生事务
	// native transaction
	tx *sql.Tx

	// 数据库配置
	config *DataSourceConfig
}

// beginTx 开启事务
// beginTx Open transaction
func (dbConnection *dataBaseConnection) beginTx(ctx context.Context) error {
	if dbConnection.tx != nil {
		return nil
	}
	// 设置事务配置,主要是隔离级别
	txOptions := dbConnection.config.DefaultTxOptions
	contextTxOptions := ctx.Value(contextTxOptionsKey)
	if contextTxOptions != nil {
		if opts, ok := contextTxOptions.(*sql.TxOptions); ok {
			txOptions = opts
		}
	}

	tx, err := dbConnection.db.BeginTx(ctx, txOptions)
	if err != nil {
		err = fmt.Errorf("->beginTx事务开启失败:%w", err)
		return err
	}
	dbConnection.tx = tx
	return nil
}

// rollback 回滚事务
// rollback Rollback transaction
func (dbConnection *dataBaseConnection) rollback() error {
	if dbConnection.tx == nil {
		return nil
	}

	err := dbConnection.tx.Rollback()
	if err != nil {
		err = fmt.Errorf("->rollback事务回滚失败:%w", err)
		return err
	}
	dbConnection.tx = nil
	return nil
}

// commit 提交事务
// commit Commit transaction
func (dbConnection *dataBaseConnection) commit() error {
	if dbConnection.tx == nil {
		return errors.New("->dbConnection.commit()事务为空")
	}

	err := dbConnection.tx.Commit()
	if err != nil {
		err = fmt.Errorf("->dbConnection.commit()事务提交失败:%w", err)
		return err
	}
	dbConnection.tx = nil
	return nil
}

// execContext 执行sql语句,如果已经开启事务,就以事务方式执行,如果没有开启事务,就以非事务方式执行
// execContext Execute sql statement,If the transaction has been opened,it will be executed in transaction mode, if the transaction is not opened,it will be executed in non-transactional mode
func (dbConnection *dataBaseConnection) execContext(ctx context.Context, sqlstr *string, argsValues *[]interface{}) (*sql.Result, error) {
	// reBuildSQL 重新处理参数代入方式
	execsql, args, err := reBuildSQL(ctx, dbConnection.config, sqlstr, argsValues)
	if err != nil {
		return nil, err
	}
	// 更新语句处理ClickHouse特殊语法
	err = reBuildUpdateSQL(ctx, dbConnection.config, execsql)
	if err != nil {
		return nil, err
	}
	// 执行前加入 hint
	err = wrapSQLHint(ctx, execsql)
	if err != nil {
		return nil, err
	}

	var start *time.Time
	var res sql.Result
	/*
		var stmt *sql.Stmt
		stmtLoad, ok := stmtSQLCacheMap.Load(*execsql)
		if ok {
			stmt = stmtLoad.(*sql.Stmt)
		} else {
			stmt, err = dbConnection.db.PrepareContext(ctx, *execsql)
			if err != nil {
				err = fmt.Errorf("->execContext-->db.PrepareContext:%w,-->zormErrorExecSQL:%s,-->zormErrorSQLValues:%s", err, *execsql, sqlErrorValues2String(*args))
			}
			stmtSQLCacheMap.Store(*execsql, stmt)
		}
	*/
	// 小于0是禁用日志输出;等于0是只输出日志,不计算SQ执行时间;大于0是计算执行时间,并且大于指定值
	slowSQLMillis := dbConnection.config.SlowSQLMillis
	if slowSQLMillis == 0 {
		FuncPrintSQL(ctx, *execsql, *args, 0)
	} else if slowSQLMillis > 0 {
		now := time.Now() // 获取当前时间
		start = &now
	}

	if dbConnection.tx != nil {
		//txStmt := dbConnection.tx.StmtContext(ctx, stmt)
		//res, err = txStmt.ExecContext(ctx, *args...)
		res, err = dbConnection.tx.ExecContext(ctx, *execsql, *args...)
	} else {
		//res, err = stmt.ExecContext(ctx, *args...)
		res, err = dbConnection.db.ExecContext(ctx, *execsql, *args...)
	}
	if slowSQLMillis > 0 {
		slow := time.Since(*start).Milliseconds()
		if slow-int64(slowSQLMillis) >= 0 {
			FuncPrintSQL(ctx, *execsql, *args, slow)
		}
	}
	if err != nil {
		err = fmt.Errorf("->execContext执行错误:%w,-->zormErrorExecSQL:%s,-->zormErrorSQLValues:%s", err, *execsql, sqlErrorValues2String(*args))
	}
	return &res, err
}

// queryRowContext 如果已经开启事务,就以事务方式执行,如果没有开启事务,就以非事务方式执行
func (dbConnection *dataBaseConnection) queryRowContext(ctx context.Context, sqlstr *string, argsValues *[]interface{}) (*sql.Row, error) {
	// reBuildSQL 重新处理参数代入方式
	query, args, err := reBuildSQL(ctx, dbConnection.config, sqlstr, argsValues)
	if err != nil {
		return nil, err
	}
	// 执行前加入 hint
	err = wrapSQLHint(ctx, query)
	if err != nil {
		return nil, err
	}
	var start *time.Time
	var row *sql.Row
	// 小于0是禁用日志输出;等于0是只输出日志,不计算SQ执行时间;大于0是计算执行时间,并且大于指定值
	slowSQLMillis := dbConnection.config.SlowSQLMillis
	if slowSQLMillis == 0 {
		FuncPrintSQL(ctx, *query, *args, 0)
	} else if slowSQLMillis > 0 {
		now := time.Now() // 获取当前时间
		start = &now
	}

	if dbConnection.tx != nil {
		row = dbConnection.tx.QueryRowContext(ctx, *query, *args...)
	} else {
		row = dbConnection.db.QueryRowContext(ctx, *query, *args...)
	}
	if slowSQLMillis > 0 {
		slow := time.Since(*start).Milliseconds()
		if slow-int64(slowSQLMillis) >= 0 {
			FuncPrintSQL(ctx, *query, *args, slow)
		}
	}
	return row, nil
}

// queryContext 查询数据,如果已经开启事务,就以事务方式执行,如果没有开启事务,就以非事务方式执行
// queryRowContext Execute sql  row statement,If the transaction has been opened,it will be executed in transaction mode, if the transaction is not opened,it will be executed in non-transactional mode
func (dbConnection *dataBaseConnection) queryContext(ctx context.Context, sqlstr *string, argsValues *[]interface{}) (*sql.Rows, error) {
	// reBuildSQL 重新处理参数代入方式
	query, args, err := reBuildSQL(ctx, dbConnection.config, sqlstr, argsValues)
	if err != nil {
		return nil, err
	}
	// 执行前加入 hint
	err = wrapSQLHint(ctx, query)
	if err != nil {
		return nil, err
	}
	var start *time.Time
	var rows *sql.Rows
	// 小于0是禁用日志输出;等于0是只输出日志,不计算SQ执行时间;大于0是计算执行时间,并且大于指定值
	slowSQLMillis := dbConnection.config.SlowSQLMillis
	if slowSQLMillis == 0 {
		FuncPrintSQL(ctx, *query, *args, 0)
	} else if slowSQLMillis > 0 {
		now := time.Now() // 获取当前时间
		start = &now
	}

	if dbConnection.tx != nil {
		rows, err = dbConnection.tx.QueryContext(ctx, *query, *args...)
	} else {
		rows, err = dbConnection.db.QueryContext(ctx, *query, *args...)
	}
	if slowSQLMillis > 0 {
		slow := time.Since(*start).Milliseconds()
		if slow-int64(slowSQLMillis) >= 0 {
			FuncPrintSQL(ctx, *query, *args, slow)
		}
	}
	if err != nil {
		err = fmt.Errorf("->queryContext执行错误:%w,-->zormErrorExecSQL:%s,-->zormErrorSQLValues:%s", err, *query, sqlErrorValues2String(*args))
	}
	return rows, err
}
