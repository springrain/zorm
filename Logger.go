package zorm

import (
	"fmt"
	"log"
)

func init() {
	//设置默认的日志显示信息,显示文件和行号
	log.SetFlags(log.Llongfile | log.LstdFlags)
}

//LogCallDepth 记录日志调用层级,用于定位到业务层代码
var LogCallDepth = 4

//FuncLogError 记录error日志
var FuncLogError func(err error) = defaultLogError

//FuncLogPanic 记录panic日志,默认使用ZormErrorLog实现
var FuncLogPanic func(err error) = defaultLogPanic

//FuncPrintSQL 打印sql语句和参数
var FuncPrintSQL func(sqlstr string, args []interface{}) = defaultPrintSQL

func defaultLogError(err error) {
	log.Output(LogCallDepth, fmt.Sprintln(err))
}
func defaultLogPanic(err error) {
	defaultLogError(err)
}
func defaultPrintSQL(sqlstr string, args []interface{}) {
	if args != nil {
		log.Output(LogCallDepth, fmt.Sprintln("sql:", sqlstr, ",args:", args))
	} else {
		log.Output(LogCallDepth, fmt.Sprintln("sql:", sqlstr))
	}

}
