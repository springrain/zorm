package test

import (
	"gitee.com/chunanyong/zorm"
	"testing"
)

var baseDao *zorm.BaseDao


func init() {
	baseDaoConfig := zorm.DataSourceConfig{
		DSN:        "root:root@tcp(127.0.0.1:3306)/readygo?charset=utf8&parseTime=true",
		DriverName: "mysql",
		PrintSQL:   true,
	}

	baseDao, _ = zorm.NewBaseDao(&baseDaoConfig)
}


func TestTransaction(c *testing.T){

}
func TestQueryStruct(c *testing.T){

}
func TestQueryStructList(c *testing.T){

}
func TestQueryMap(c *testing.T){

}
func TestQueryMapList(c *testing.T){

}
func TestUpdateFinder(c *testing.T){

}
func TestSaveStruct(c *testing.T){

}
func TestUpdateStruct(c *testing.T){

}
func TestUpdateStructNotZeroValue(c *testing.T){

}
func TestDeleteStruct(c *testing.T){

}
func TestSaveEntityMap(c *testing.T){

}