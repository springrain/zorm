package zorm

import (
	"fmt"
	"log"
)

func init() {
	log.SetFlags(log.Llongfile | log.LstdFlags)
}

//ZormLogCalldepth 记录日志行数的文件深度,用于定位到业务层代码
var ZormLogCalldepth = 4

//ZormErrorLog 记录error日志
var ZormErrorLog func(err error) = zormErrorLog

//ZormPanicLog 记录panic日志
var ZormPanicLog func(err error) = zormPanicLog

//ZormPrintSQL 打印sql语句和参数
var ZormPrintSQL func(sqlstr string, args []interface{}) = zormPrintSQL

func zormErrorLog(err error) {
	log.Output(ZormLogCalldepth, fmt.Sprintln(err))
}
func zormPanicLog(err error) {
	log.Output(ZormLogCalldepth, fmt.Sprintln(err))
}
func zormPrintSQL(sqlstr string, args []interface{}) {
	if args != nil {
		log.Output(ZormLogCalldepth, fmt.Sprintln("sql:", sqlstr, ",args:", args))
	} else {
		log.Output(ZormLogCalldepth, fmt.Sprintln("sql:", sqlstr))
	}

}
