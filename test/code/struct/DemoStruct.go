package permstruct
import (
	"time"

	"gitee.com/chunanyong/zorm"
)

//DemoStructTableName 表名常量,方便直接调用
const DemoStructTableName = "t_demo"

// DemoStruct 栗子
type DemoStruct struct {
	//引入默认的struct,隔离IEntityStruct的方法改动
	zorm.EntityStruct
	
    //Id  
    Id string `column:"id"`
    
    //UserName 姓名
    UserName string `column:"userName"`
    
    //Password 密码
    Password string `column:"password"`
    
    //Mobile 手机号码
    Mobile string `column:"mobile"`
    
    //CreateTime <no value>
    CreateTime time.Time `column:"createTime"`
    
    //Active 是否有效(0否,1是)
    Active int `column:"active"`
    
	//------------------数据库字段结束,自定义字段写在下面---------------//


}


//GetTableName 获取表名称
func (entity *DemoStruct) GetTableName() string {
	return DemoStructTableName
}

//GetPKColumnName 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称.
func (entity *DemoStruct) GetPKColumnName() string {
	return "id"
}

