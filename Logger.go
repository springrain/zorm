package zorm

import (
	"fmt"
	"log"
)

func init() {
	log.SetFlags(log.Llongfile | log.LstdFlags)
}

//LogCalldepth 记录日志行数的文件深度,用于定位到业务层代码
var LogCalldepth = 4

//FuncLogError 记录error日志
var FuncLogError func(err error) = defaultLogError

//FuncLogPanic 记录panic日志,默认使用ZormErrorLog实现
var FuncLogPanic func(err error) = defaultLogPanic

//FuncPrintSQL 打印sql语句和参数
var FuncPrintSQL func(sqlstr string, args []interface{}) = defaultPrintSQL

func defaultLogError(err error) {
	log.Output(LogCalldepth, fmt.Sprintln(err))
}
func defaultLogPanic(err error) {
	defaultLogError(err)
}
func defaultPrintSQL(sqlstr string, args []interface{}) {
	if args != nil {
		log.Output(LogCalldepth, fmt.Sprintln("sql:", sqlstr, ",args:", args))
	} else {
		log.Output(LogCalldepth, fmt.Sprintln("sql:", sqlstr))
	}

}
