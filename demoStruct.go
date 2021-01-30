package zorm

import (
	"database/sql"
)

//建表语句

/*

DROP TABLE IF EXISTS `t_demo`;
CREATE TABLE `t_demo`  (
  `id` INT PRIMARY KEY AUTO_INCREMENT,
  `userName` varchar(30)  NOT NULL COMMENT '姓名',
  `password` varchar(50)  NOT NULL COMMENT '密码',
  `createTime` datetime(0) NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
  `active` int(0) NOT NULL DEFAULT 1 COMMENT '是否有效(0否,1是)'
) ENGINE = InnoDB CHARACTER SET = utf8mb4  COMMENT = '例子' ;

*/

//demoStructTableName 表名常量,方便直接调用
const demoStructTableName = "t_demo"

// demoStruct 例子
type demoStruct struct {
	//引入默认的struct,隔离IEntityStruct的方法改动
	EntityStruct

	//Id 主键
	Id int64 `column:"ID"`

	//UserName 姓名
	UserName string `column:"userName"`

	//Password 密码
	Password sql.NullString `column:"password"`

	//CreateTime <no value>
	//CreateTime time.Time `column:"createTime"`

	//Active 是否有效(0否,1是)
	Active uint16 `column:"active"`

	//------------------数据库字段结束,自定义字段写在下面---------------//

}

//GetTableName 获取表名称
func (entity *demoStruct) GetTableName() string {
	return demoStructTableName
}

//GetPKColumnName 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称.
func (entity *demoStruct) GetPKColumnName() string {
	return "ID"
}

//newDemoStruct 创建一个默认对象
func newDemoStruct() demoStruct {
	demo := demoStruct{
		//如果Id=="",保存时zorm会调用FuncGenerateStringID(),默认UUID字符串,也可以自己定义实现方式,例如 FuncGenerateStringID=funcmyId
		//Id:         FuncGenerateStringID(),
		//UserName: "defaultUserName",
		//Password:   "defaultPassword",
		Active: 1,
		//CreateTime: time.Now(),
	}
	return demo
}
