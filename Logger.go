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
	"encoding/json"
	"fmt"
	"log"
)

func init() {
	// 设置默认的日志显示信息,显示文件和行号
	// Set the default log display information, display file and line number.
	log.SetFlags(log.Llongfile | log.LstdFlags)
}

// LogCallDepth 记录日志调用层级,用于定位到业务层代码
// Log Call Depth Record the log call level, used to locate the business layer code
var LogCallDepth = 4

// FuncLogError 记录error日志.NewDBDao方法里的异常,ctx为nil,扩展时请注意
// FuncLogError Record error log
var FuncLogError func(ctx context.Context, err error) = defaultLogError

// FuncLogPanic  记录panic日志,默认使用"defaultLogError"实现
// FuncLogPanic Record panic log, using "defaultLogError" by default
var FuncLogPanic func(ctx context.Context, err error) = defaultLogPanic

// FuncPrintSQL 打印sql语句,参数和执行时间,小于0是禁用日志输出;等于0是只输出日志,不计算SQ执行时间;大于0是计算执行时间,并且大于指定值
// FuncPrintSQL Print sql statement and parameters
var FuncPrintSQL func(ctx context.Context, sqlstr string, args []interface{}, execSQLMillis int64) = defaultPrintSQL

func defaultLogError(ctx context.Context, err error) {
	log.Output(LogCallDepth, fmt.Sprintln(err))
}

func defaultLogPanic(ctx context.Context, err error) {
	defaultLogError(ctx, err)
}

func defaultPrintSQL(ctx context.Context, sqlstr string, args []interface{}, execSQLMillis int64) {
	if args != nil {
		log.Output(LogCallDepth, fmt.Sprintln("sql:", sqlstr, ",args:", args, ",execSQLMillis:", execSQLMillis))
	} else {
		log.Output(LogCallDepth, fmt.Sprintln("sql:", sqlstr, ",args: [] ", ",execSQLMillis:", execSQLMillis))
	}
}

// sqlErrorValues2String 处理values值日志记录格式
func sqlErrorValues2String(values []interface{}) string {
	jsonStr := "[]"
	if values == nil || len(values) < 1 {
		return jsonStr
	}
	bytes, err := json.Marshal(values)
	if err == nil {
		jsonStr = string(bytes)
	}
	return jsonStr
}
