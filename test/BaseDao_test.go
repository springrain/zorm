package test

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"gitee.com/chunanyong/zorm"
 	_ "github.com/go-sql-driver/mysql"
	"strings"
	permstruct "test/code/struct"
	"testing"
	"time"
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


func TestTransaction(t *testing.T){

}
func TestQueryStruct(t *testing.T){

}
func TestQueryStructList(t *testing.T){

}
func TestQueryMap(t *testing.T){

}
func TestQueryMapList(t *testing.T){

}
func TestUpdateFinder(t *testing.T){

}

//保存 Struct
func TestSaveStruct(t *testing.T){

}
func TestUpdateStruct(t *testing.T){

}
func TestUpdateStructNotZeroValue(t *testing.T){

}
func TestDeleteStruct(t *testing.T){
	zorm.Transaction(context.Background(), func(ctx context.Context) (interface{}, error) {
		//uuid
		uuid := zorm.FuncGenerateStringID()

		var signStr = "123456"

		hash := md5.New()
		hash.Write([]byte(signStr))
		hashSign := hash.Sum(nil)
		demoStruct := permstruct.DemoStruct{
			Id:         uuid,
			UserName:   "范进",
			Password:   strings.ToUpper(hex.EncodeToString(hashSign)),
			Active:     1,
			CreateTime: time.Now(),
		}

		err := zorm.SaveStruct(ctx, &demoStruct)
		if err != nil {
			t.Error(err.Error())
		}
		return nil,nil
	})
}

//保存 EntityMap
func TestSaveEntityMap(t *testing.T){

}