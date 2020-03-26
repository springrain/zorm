package test

import (
	"context"
	"testing"

	"gitee.com/chunanyong/zorm"

	//0.引入数据库驱动
	_ "github.com/go-sql-driver/mysql"
)

//全局
var baseDao *zorm.BaseDao

//ctx默认应该有 web层传入,例如gin的c.Request.Context().这里只是模拟
var ctx = context.Background()

//1.初始化BaseDao
func init() {
	baseDaoConfig := zorm.DataSourceConfig{
		DSN:        "root:root@tcp(127.0.0.1:3306)/readygo?charset=utf8&parseTime=true",
		DriverName: "mysql",
		PrintSQL:   true,
	}
	baseDao, _ = zorm.NewBaseDao(&baseDaoConfig)
}

func TestTransaction(t *testing.T) {

}
func TestQueryStruct(t *testing.T) {

}
func TestQueryStructList(t *testing.T) {

}
func TestQueryMap(t *testing.T) {

}
func TestQueryMapList(t *testing.T) {

}
func TestUpdateFinder(t *testing.T) {

}

//保存 Struct
func TestSaveStruct(t *testing.T) {

}
func TestUpdateStruct(t *testing.T) {

}
func TestUpdateStructNotZeroValue(t *testing.T) {

}
func TestDeleteStruct(t *testing.T) {
	zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		err := zorm.SaveStruct(ctx, &demoStruct)
		if err != nil {
			t.Error(err.Error())
		}
		return nil, nil
	})
}

//保存 EntityMap
func TestSaveEntityMap(t *testing.T) {

}
