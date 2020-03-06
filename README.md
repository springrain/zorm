# readygo

#### 介绍
golang开发脚手架

#### 软件架构
基于gin和自研ORM  
[自带代码生成器](https://gitee.com/chunanyong/readygo/tree/master/codeGenerator)  
使用orm.Finder作为sql载体,所有的sql语句最终都是通过finder执行.  
支持事务传播  


#### 例子
具体可以参照 [UserStructService.go](https://gitee.com/chunanyong/readygo/tree/master/permission/permservice)

0. 初始化zorm
    ```
dataSourceConfig := zorm.DataSourceConfig{
		Host:     "127.0.0.1",
		Port:     3306,
		DBName:   "readygo",
		UserName: "root",
		PassWord: "root",
		DBType:   "mysql",
	}
	zorm.NewBaseDao(&dataSourceConfig)
    ```
1.  增
    ```
    var user permstruct.UserStruct
    err := zorm.SaveStruct(nil, &user)
    ```
2.  删
    ```
    err := zorm.DeleteStruct(nil,&user)
    ```
  
3.  改
    ```
    err := zorm.UpdateStruct(nil,&user)
    //finder更新
    err := zorm.UpdateFinder(nil,finder)
    ```
4.  查
    ```
	finder := zorm.NewSelectFinder(permstruct.UserStructTableName)
	page := zorm.NewPage()
	var users = make([]permstruct.UserStruct, 0)
	err := zorm.QueryStructList(nil, finder, &users, &page)
    ```
5.  事务传播
    ```
    //匿名函数return的error如果不为nil,事务就会回滚
	_, errSaveUserStruct := zorm.Transaction(dbConnection, func(dbConnection *zorm.DBConnection) (interface{}, error) {

		//事务下的业务代码开始
		errSaveUserStruct := zorm.SaveStruct(dbConnection, userStruct)

		if errSaveUserStruct != nil {
			return nil, errSaveUserStruct
		}

		return nil, nil
		//事务下的业务代码结束

	})
    ```
6.  [测试](https://www.jianshu.com/p/1adc69468b6f)
    ```
    //函数测试
    go test -run TestAdd2
    //性能测试
    go test -bench=.
    go test -v -bench=. -cpu=8 -benchtime="3s" -timeout="5s" -benchmem
    ```

