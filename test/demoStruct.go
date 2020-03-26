package test



//建表语句
/*
DROP TABLE IF EXISTS `t_demo`;
CREATE TABLE `t_demo`  (
  `id` varchar(50)  NOT NULL COMMENT ' ',
  `userName` varchar(30)  NOT NULL COMMENT '姓名',
  `password` varchar(50)  NOT NULL COMMENT '密码',
  `mobile` varchar(16)  NOT NULL COMMENT '手机号码',
  `createTime` datetime(0) NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
  `active` int(0) NOT NULL DEFAULT 1 COMMENT '是否有效(0否,1是)',
  PRIMARY KEY (`id`)
) ENGINE = InnoDB CHARACTER SET = utf8mb4  COMMENT = '例子' ;
*/


//demoStructTableName 表名常量,方便直接调用
const demoStructTableName = "t_demo"

// demoStruct 例子
type demoStruct struct {
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
func (entity *demoStruct) GetTableName() string {
	return demoStructTableName
}

//GetPKColumnName 获取数据库表的主键字段名称.因为要兼容Map,只能是数据库的字段名称.
func (entity *demoStruct) GetPKColumnName() string {
	return "id"
}

//newDemoStruct 创建一个默认对象
func newDemoStruct() demoStruct{
	demo := demoStruct{
		Id:         zorm.FuncGenerateStringID(),
		UserName:   "defaultUserName",
		Password:   "defaultPassword",
		Active:     1,
		CreateTime: time.Now(),
	}
	return demo
}