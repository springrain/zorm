package zorm

import (
	"fmt"
	"log"
)

func init() {
	//设置默认的日志显示信息,显示文件和行号
	//Set the default log display information, display file and line number.
	log.SetFlags(log.Llongfile | log.LstdFlags)
}

//LogCallDepth 记录日志调用层级,用于定位到业务层代码
//Log Call Depth Record the log call level, used to locate the business layer code
var LogCallDepth = 4

//FuncLogError 记录error日志
//FuncLogError Record error log
var FuncLogError func(err error) = defaultLogError

//FuncLogPanic  记录panic日志,默认使用"defaultLogError"实现
//FuncLogPanic Record panic log, using "defaultLogError" by default
var FuncLogPanic func(err error) = defaultLogPanic

//FuncPrintSQL 打印sql语句,参数和执行时间,小于0是禁用日志输出;等于0是只输出日志,不计算SQ执行时间;大于0是计算执行时间,并且大于指定值
//FuncPrintSQL Print sql statement and parameters
var FuncPrintSQL func(sqlstr string, args []interface{}, slowSQLMillis int64) = defaultPrintSQL

func defaultLogError(err error) {
	log.Output(LogCallDepth, fmt.Sprintln(err))
}
func defaultLogPanic(err error) {
	defaultLogError(err)
}
func defaultPrintSQL(sqlstr string, args []interface{}, slowSQLMillis int64) {
	if args != nil {
		log.Output(LogCallDepth, fmt.Sprintln("sql:", sqlstr, ",args:", args, ",slowSQLMillis:", slowSQLMillis))
	} else {
		log.Output(LogCallDepth, fmt.Sprintln("sql:", sqlstr, ",args: [] ", ",slowSQLMillis:", slowSQLMillis))
	}

}
