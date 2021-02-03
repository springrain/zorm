## Introduction
This is a lightweight ORM that supports DM,Kingbase,shentong,mysql,postgresql,oracle,mssql,sqlite databases.

Source address:https://gitee.com/chunanyong/zorm  
Author blog:[https://www.jiagou.com](https://www.jiagou.com)  

``` 
go get gitee.com/chunanyong/zorm 
```  

* Written based on native SQL statements,It is the streamlining and optimization of [springrain](https://gitee.com/chunanyong/springrain).
* [Built-in code generator](https://gitee.com/chunanyong/readygo/tree/master/codegenerator)  
* The code is streamlined, with a total of about 2000 lines, detailed comments, convenient for customization and modification. 
* <font color=red>Support transaction propagation, which is the main reason for the birth of zorm</font>
* Support mysql, postgresql, oracle, mssql, sqlite, dm (Da Meng), kingbase (Ren Da Jincang)
* Support database read and write separation.
* The update performance of zorm, gorm, and xorm is equivalent. The read performance of zorm is twice as fast as that of gorm and xorm.
zorm生产环境使用参考: [UserStructService.go](https://gitee.com/chunanyong/readygo/tree/master/permission/permservice)  

## Support domestic database  
DM(Da Meng) database driver: [https://gitee.com/chunanyong/dm](https://gitee.com/chunanyong/dm)  

kingbase(Ren Da Jincang)Driver Instructions: [https://help.kingbase.com.cn/doc-view-8108.html](https://help.kingbase.com.cn/doc-view-8108.html)  
The core of Kingbase(Ren Da Jincang) 8 is based on postgresql 9.6. You can use [https://github.com/lib/pq](https://github.com/lib/pq) for testing. The official driver is recommended for the production environment.    

## Test case  
https://gitee.com/chunanyong/readygo/blob/master/test/testzorm/BaseDao_test.go  

```go  
// Zorm uses native SQL statements and does not impose restrictions on SQL syntax. Statements use Finder as the carrier.
// Use "?" as a placeholder. , Zorm automatically replaces placeholders based on the database type, 
// such as "?" in a PostgreSQL database, Replaced with $1, $2...
// Zorm uses the ctx context. context parameter to propagate the transaction, and ctx is passed in from the web layer, such as gin's c.retest.context ().
// The transaction operation of zorm needs to be displayed using zorm.Transaction(ctx, func(ctx context.Context) (interface(), error) ()) to open
``` 



## Database scripts and entity classes
https://gitee.com/chunanyong/readygo/blob/master/test/testzorm/demoStruct.go

Generate entity classes or write manually, it is recommended to use a code generator ： 
https://gitee.com/chunanyong/readygo/tree/master/codegenerator
```go 

package testzorm

import (
	"time"

	"gitee.com/chunanyong/zorm"
)

//Table building statement

/*

DROP TABLE IF EXISTS `t_demo`;
CREATE TABLE `t_demo`  (
  `id` varchar(50)  NOT NULL COMMENT 'Primary key',
  `userName` varchar(30)  NOT NULL COMMENT 'Name',
  `password` varchar(50)  NOT NULL COMMENT 'password',
  `createTime` datetime(0) NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
  `active` int(0) NOT NULL DEFAULT 1 COMMENT 'Is it valid (0 no, 1 yes)',
  PRIMARY KEY (`id`)
) ENGINE = InnoDB CHARACTER SET = utf8mb4  COMMENT = 'example' ;

*/

//demoStructTableName  Table name constant, easy to call directly
const demoStructTableName = "t_demo"

// demoStruct example
type demoStruct struct {
	//Default structs are introduced to insulate IEntityStructs from method changes
	zorm.EntityStruct

	//Id: Primary key
	Id string `column:"id"`

	//UserName: Name 
	UserName string `column:"userName"`

	//Password: password
	Password string `column:"password"`

	//CreateTime <no value>
	CreateTime time.Time `column:"createTime"`

	//Active: Is it valid (0 no, 1 yes)
	Active int `column:"active"`

	//------------------The end of the database field, the custom field is written below---------------//

}

//GetTableName: Get the table name
func (entity *demoStruct) GetTableName() string {
	return demoStructTableName
}

//GetPKColumnName: Get the primary key field name of the database table. Because it is compatible with Map, it can only be the field name of the database.
func (entity *demoStruct) GetPKColumnName() string {
	return "id"
}

//newDemoStruct: Create a default object
func newDemoStruct() demoStruct {
	demo := demoStruct{
		// If Id=="",When saving, zorm will call zorm.Func Generate String ID(),
        // the default UUID string, or you can define your own implementation,E.g: zorm.FuncGenerateStringID=funcmyId
		Id:         zorm.FuncGenerateStringID(),
		UserName:   "defaultUserName",
		Password:   "defaultPassword",
		Active:     1,
		CreateTime: time.Now(),
	}
	return demo
}


```

## Test cases are documents

```go  

// testzorm: Use native SQL statements, no restrictions on SQL syntax. Statements use Finder as a carrier
// Use "?" as a placeholder. , Zorm automatically replaces placeholders based on the database type, 
// such as "?" in a PostgreSQL database, Replaced with $1, $2...
// Zorm uses the ctx context. context parameter to propagate the transaction, and ctx is passed in from the web layer, such as gin's c.retest.context ().
// The transaction operation of zorm needs to be displayed using zorm.Transaction(ctx, func(ctx context.Context) (interface(), error) ()) to open
package testzorm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gitee.com/chunanyong/zorm"

	//00.Introduce database driver
	_ "github.com/go-sql-driver/mysql"
)

//dbDao: Represents a database. If there are multiple databases, declare multiple DB Dao accordingly
var dbDao *zorm.DBDao

// ctx should be passed in by the web layer by default, such as gin's c.Request.Context(). This is just a simulation.
var ctx = context.Background()

//01.Initialize DB Dao
func init() {

	//Custom zorm log output
	//zorm.LogCalldepth = 4 //Level of log call
	//zorm.FuncLogError = myFuncLogError //Function to record exception log.
	//zorm.FuncLogPanic = myFuncLogPanic //Record panic log, use Zorm Error Log by default
	//zorm.FuncPrintSQL = myFuncPrintSQL //A function that prints SQL

	//Customize the log output format and re-assign the Func Print SQL functio.
	//log.SetFlags(log.LstdFlags)
	//zorm.FuncPrintSQL = zorm.FuncPrintSQL

	//dbDaoConfig: Database configuration
	dbDaoConfig := zorm.DataSourceConfig{
		// DSN: Database connection string
		DSN: "root:root@tcp(127.0.0.1:3306)/readygo?charset=utf8&parseTime=true",
		// Database driver name: mysql, postgres, oci8, sqlserver, sqlite3, 
        // dm, kingbase and DBType correspond, there are multiple drivers for processing databases
		DriverName: "mysql",
		// Database type (based on dialect judgment): mysql, postgresql, 
        // oracle, mssql, sqlite, dm, kingbase and DriverName correspond to multiple drivers for processing databases
		DBType: "mysql",
		//MaxOpenConns: Maximum number of database connections Default 50
		MaxOpenConns: 50,
		//MaxIdleConns: The maximum number of free connections to the database default 50
		MaxIdleConns: 50,
		//ConnMaxLifetimeSecond: The connection survival time in seconds. The connection is destroyed and rebuilt after the default 600 (10 minutes). 
        //To prevent the database from actively disconnecting and causing dead connections. MySQL default wait_timeout 28800 seconds (8 hours)
		ConnMaxLifetimeSecond: 600,
		//PrintSQL: Print SQL. Func Print SQL will be used to record SQL
		PrintSQL: true,
		
		//MockSQLDB 用于mock测试的入口,如果MockSQLDB不为nil,则不使用DSN,直接使用MockSQLDB
		//db, mock, err := sqlmock.New()
		//MockSQLDB: db,
	}

	// Create dbDao according to dbDaoConfig, a database is executed only once,
    // the first executed database is defaultDao, and subsequent zorm.xxx methods, defaultDao is used by default.
	dbDao, _ = zorm.NewDBDao(&dbDaoConfig)
}

//TestInsert: 02.Test save Struct object
func TestInsert(t *testing.T) {

	//You need to manually start the transaction. 
    //If the error returned by the anonymous function is not nil, the transaction will be rolled back.
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		//Create a demo object
		demo := newDemoStruct()

		// Save the object, the parameter is the object pointer. 
        // If the primary key is incremented, it will be assigned to the primary key attribute of the object
		_, err := zorm.Insert(ctx, &demo)

		//If the returned err is not nil, the transaction will be rolled back.
		return nil, err
	})
	//Mark test failed.
	if err != nil {
		t.Errorf("err:%v", err)
	}
}

//TestInsertSlice 03.Test the Slice that saves Struct objects in batches.
//If it is an auto-increasing primary key, you cannot assign a value to the primary key attribute in the Struct object.
func TestInsertSlice(t *testing.T) {

	// You need to manually start the transaction. 
    // If the error returned by the anonymous function is not nil, the transaction will be rolled back.
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

		//The type stored by slice is zorm.I Entity Struct!!!, golang currently does not have generics, 
        //uses the I Entity Struct interface, and is compatible with the Struct entity class.
		demoSlice := make([]zorm.IEntityStruct, 0)

		//Create object 1
		demo1 := newDemoStruct()
		demo1.UserName = "demo1"
		//Create object 2
		demo2 := newDemoStruct()
		demo2.UserName = "demo2"

		demoSlice = append(demoSlice, &demo1, &demo2)

		//To save objects in batches, if the primary key is auto-increment, the auto-increment ID cannot be saved in the object.
		_, err := zorm.InsertSlice(ctx, demoSlice)

		//If the returned err is not nil, the transaction will be rolled back.
		return nil, err
	})
	//Mark test failed.
	if err != nil {
		t.Errorf("错误:%v", err)
	}
}

//TestInsertEntityMap 04.Test to save the Entity Map object for scenarios where it is not convenient to use struct, using Map as a carrier
func TestInsertEntityMap(t *testing.T) {

	// You need to manually start the transaction. If the error returned by the anonymous function is not nil, the transaction will be rolled back.
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		//To create an Entity Map, you need to pass in the table name.
		entityMap := zorm.NewEntityMap(demoStructTableName)
		//Set the primary key name.
		entityMap.PkColumnName = "id"
		//If it is an auto-increasing sequence, set the value of the sequence.
		//entityMap.PkSequence = "mySequence"

		//Set Set the field value of the database
		//If the primary key is auto-increment or sequence, don't entity Map.Set the value of the primary key.
		entityMap.Set("id", zorm.FuncGenerateStringID())
		entityMap.Set("userName", "entityMap-userName")
		entityMap.Set("password", "entityMap-password")
		entityMap.Set("createTime", time.Now())
		entityMap.Set("active", 1)

		//carried out
		_, err := zorm.InsertEntityMap(ctx, entityMap)

		//If the returned err is not nil, the transaction will be rolled back
		return nil, err
	})
	//Mark test failed
	if err != nil {
		t.Errorf("error:%v", err)
	}
}

//TestQueryRow 05.Test query a struct object
func TestQueryRow(t *testing.T) {

	//Declare a pointer to an object to carry the returned data.
	demo := &demoStruct{}

	//Finder for constructing query.
	finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo
	//finder = zorm.NewSelectFinder(demoStructTableName, "id,userName") // select id,userName from t_demo
	//finder = zorm.NewFinder().Append("SELECT * FROM " + demoStructTableName) // select * from t_demo

	// finder.Append： The first parameter is the statement, and the following parameters are the corresponding values.
    // The order of the values ​​must be correct. Use the statement uniformly? Zorm will handle the difference in the database
	finder.Append("WHERE id=? and active in(?)", "41b2aa4f-379a-4319-8af9-08472b6e514e", []int{0, 1})

	//Execute query
	err := zorm.QueryRow(ctx, finder, demo)

	if err != nil { //Mark test failed
		t.Errorf("error:%v", err)
	}
	//Print result
	fmt.Println(demo)
}

//TestQueryRowMap 06.Test query map receiving results, used in scenarios that are not suitable for struct, more flexible
func TestQueryRowMap(t *testing.T) {

	//Finder for constructing query.
	finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo
	//finder.Append: The first parameter is the statement, and the following parameters are the corresponding values. 
    //The order of the values ​​must be correct. Use the statement uniformly? Zorm will handle the difference in the database
	finder.Append("WHERE id=? and active in(?)", "41b2aa4f-379a-4319-8af9-08472b6e514e", []int{0, 1})
	//Execute query
	resultMap, err := zorm.QueryRowMap(ctx, finder)

	if err != nil { //Mark test failed
		t.Errorf("error:%v", err)
	}
	//Print result
	fmt.Println(resultMap)
}

//TestQuerySlice 07.Test query object list
func TestQuery(t *testing.T) {
	//Create a slice to receive the result
	list := make([]*demoStruct, 0)

	//Finder for constructing query
	finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo
	//Create a paging object. After the query is completed, the page object can be directly used by the front-end paging component.
	page := zorm.NewPage()
	page.PageNo = 1    //Query page 1, default is 1
	page.PageSize = 20 //20 items per page, the default is 20

	//Execute query
	err := zorm.Query(ctx, finder, &list, page)
	if err != nil { //Mark test failed
		t.Errorf("error:%v", err)
	}
	//Print result
	fmt.Println("Total number:", page.TotalCount, "  List:", list)
}

//TestQueryMap 08.Test query map list, used in scenarios where struct is not convenient, a record is a map object.
func TestQueryMap(t *testing.T) {
	//Finder for constructing query.
	finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo
	
	//Create a paging object. After the query is completed, the page object can be directly used by the front-end paging component。
	page := zorm.NewPage()

	//Execute query
	listMap, err := zorm.QueryMap(ctx, finder, page)
	if err != nil { //Mark test failed
		t.Errorf("error:%v", err)
	}
	//Print result
	fmt.Println("Total number:", page.TotalCount, "  List:", listMap)
}

//TestUpdateNotZeroValue 09.Update the struct object, only update fields that are not zero. The primary key must have a value.
func TestUpdateNotZeroValue(t *testing.T) {

	// You need to manually start the transaction. If the error returned by the anonymous function is not nil,
    // the transaction will be rolled back.
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		//Declare a pointer to an object to update data
		demo := &demoStruct{}
		demo.Id = "41b2aa4f-379a-4319-8af9-08472b6e514e"
		demo.UserName = "UpdateNotZeroValue"

		//Update "sql":"UPDATE t_demo SET userName=? WHERE id=?","args":["UpdateNotZeroValue","41b2aa4f-379a-4319-8af9-08472b6e514e"]
		_, err := zorm.UpdateNotZeroValue(ctx, demo)

		//If the returned err is not nil, the transaction will be rolled back.
		return nil, err
	})
	if err != nil { 
        //Mark test failed
		t.Errorf("error:%v", err)
	}

}

//TestUpdate 10.Update the struct object, update all fields. The primary key must have a value.
func TestUpdate(t *testing.T) {

	// You need to manually start the transaction. 
    // If the error returned by the anonymous function is not nil, the transaction will be rolled back.
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

		//Declare a pointer to an object to update data.
		demo := &demoStruct{}
		demo.Id = "41b2aa4f-379a-4319-8af9-08472b6e514e"
		demo.UserName = "TestUpdate"

		_, err := zorm.Update(ctx, demo)

		//If the returned err is not nil, the transaction will be rolled back.
		return nil, err
	})
	if err != nil { 
        //Mark test failed
		t.Errorf("error:%v", err)
	}
}

//TestUpdateFinder 11.Through finder update, zorm is the most flexible way, you can write any update statement, 
// or even manually write insert statement
func TestUpdateFinder(t *testing.T) {
	//You need to manually start the transaction. If the error returned by the anonymous function is not nil, the transaction will be rolled back.
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		finder := zorm.NewUpdateFinder(demoStructTableName) // UPDATE t_demo SET
		//finder = zorm.NewDeleteFinder(demoStructTableName)  // DELETE FROM t_demo
		//finder = zorm.NewFinder().Append("UPDATE").Append(demoStructTableName).Append("SET") // UPDATE t_demo SET
		finder.Append("userName=?,active=?", "TestUpdateFinder", 1).Append("WHERE id=?", "41b2aa4f-379a-4319-8af9-08472b6e514e")

		//Update "sql":"UPDATE t_demo SET  userName=?,active=? WHERE id=?","args":["TestUpdateFinder",1,"41b2aa4f-379a-4319-8af9-08472b6e514e"]
		_, err := zorm.UpdateFinder(ctx, finder)

		//If the returned err is not nil, the transaction will be rolled back.
		return nil, err
	})
	if err != nil { //Mark test failed
		t.Errorf("error:%v", err)
	}

}

//TestUpdateEntityMap 12.Update an Entity Map, the primary key must have a value
func TestUpdateEntityMap(t *testing.T) {
	//You need to manually start the transaction. 
    //If the error returned by the anonymous function is not nil, the transaction will be rolled back.
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		//To create an Entity Map, you need to pass in the table name.
		entityMap := zorm.NewEntityMap(demoStructTableName)
		//Set the primary key name.
		entityMap.PkColumnName = "id"
		//Set： Set the field value of the database, the primary key must have a value.
		entityMap.Set("id", "41b2aa4f-379a-4319-8af9-08472b6e514e")
		entityMap.Set("userName", "TestUpdateEntityMap")
		//Update "sql":"UPDATE t_demo SET userName=? WHERE id=?","args":["TestUpdateEntityMap","41b2aa4f-379a-4319-8af9-08472b6e514e"]
		_, err := zorm.UpdateEntityMap(ctx, entityMap)

		//If the returned err is not nil, the transaction will be rolled back.
		return nil, err
	})
	if err != nil { 
        //Mark test failed
		t.Errorf("error:%v", err)
	}

}

//TestDelete 13.To delete a struct object, the primary key must have a value.
func TestDelete(t *testing.T) {
	//You need to manually start the transaction. If the error returned by the anonymous function is not nil, the transaction will be rolled back.
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		demo := &demoStruct{}
		demo.Id = "ae9987ac-0467-4fe2-a260-516c89292684"

		//delete： "sql":"DELETE FROM t_demo WHERE id=?","args":["ae9987ac-0467-4fe2-a260-516c89292684"]
		_, err := zorm.Delete(ctx, demo)

		//If the returned err is not nil, the transaction will be rolled back.
		return nil, err
	})
	if err != nil { 
        //Mark test failed
		t.Errorf("error:%v", err)
	}

}


//TestProc 14.Test call stored procedure
func TestProc(t *testing.T) {
	demo := &demoStruct{}
	finder := zorm.NewFinder().Append("call testproc(?) ", "u_10001")
	zorm.QueryRow(ctx, finder, demo)
	fmt.Println(demo)
}

//TestFunc 15.Test call custom function.
func TestFunc(t *testing.T) {
	userName := ""
	finder := zorm.NewFinder().Append("select testfunc(?) ", "u_10001")
	zorm.QueryRow(ctx, finder, &userName)
	fmt.Println(userName)
}

//TestOther 16.Some other instructions. Thank you very much for seeing this line.
func TestOther(t *testing.T) {

	//Scenario 1. Multiple databases. Through the db Dao of the corresponding database, call the Bind Context DB Connection function, 
    //bind the connection of this database to the returned ctx, and then pass the ctx to the zorm function.
	newCtx, err := dbDao.BindContextDBConnection(ctx)
	if err != nil {
         //Mark test failed
		t.Errorf("error:%v", err)
	}

	finder := zorm.NewSelectFinder(demoStructTableName)
	//Pass the newly generated new Ctx to the function of zorm.
	list, _ := zorm.QueryRowMap(newCtx, finder, nil)
	fmt.Println(list)

	//Scenario 2. Read-write separation of a single database. 
    //Set the strategy function for read-write separation.
	zorm.FuncReadWriteStrategy = myReadWriteStrategy

	//Scenario 3. If there are multiple databases, 
    //each database is also separated from reading and writing, and processed according to scenario 1.

}

//Strategies for the separation of read and write of a single database rwType=0 read,rwType=1 write
func myReadWriteStrategy(rwType int) *zorm.DBDao {
	//According to your own business scenario, return the required read and write dao, and call this function every time you need a database connection
	return dbDao
}

//---------------------------------//

//To implement the interface of CustomDriverValueConver,extend the custom type, such as text type of dm database, the mapped type is dm.DmClob type , cannot use string type to receive directly.
type CustomDMText struct{}
//GetDriverValue according to the database column type and entity class field type, return driver.Value Instance. If the return value is nil, no type replacement is performed and the default method is used.
func (dmtext CustomDMText) GetDriverValue(columnType *sql.ColumnType, structFieldType reflect.Type) (driver.Value, error) {
	return &dm.DmClob{}, nil
}

//ConverDriverValue database column type, entity class field type, GetDriverValue returned driver.Value New value, return the pointer according to the receiving type value, pointer, pointer!!!!
func (dmtext CustomDMText) ConverDriverValue(columnType *sql.ColumnType, structFieldType reflect.Type, tempDriverValue driver.Value) (interface{}, error) {
	dm, _ := tempDriverValue.(*dm.DmClob)
	dmlen, _ := dm.GetLength()
	strInt64 := strconv.FormatInt(dmlen, 10)
	dmlenInt, _ := strconv.Atoi(strInt64)
	str, _ := dm.ReadString(1, dmlenInt)
	return &str, nil
}
//zorm.CustomDriverValueMap for configuration driver.Value and the corresponding processing relationship, key is the string of drier.Value. For example *dm.DmClob
//It is usually added in the init method
zorm.CustomDriverValueMap["*dm.DmClob"] = CustomDMText{}


```  


##  Performance stress test

   Test code:https://github.com/alphayan/goormbenchmark

   Index description
   Total time, average number of nanoseconds per time, average memory allocated per time, average number of memory allocated per time.

   The update performance of zorm, gorm, and xorm is equivalent. The read performance of zorm is twice as fast as that of gorm and xorm.  

```
2000 times - Insert
      zorm:     9.05s      4524909 ns/op    2146 B/op     33 allocs/op
      gorm:     9.60s      4800617 ns/op    5407 B/op    119 allocs/op
      xorm:    12.63s      6315205 ns/op    2365 B/op     56 allocs/op

    2000 times - BulkInsert 100 row
      xorm:    23.89s     11945333 ns/op  253812 B/op   4250 allocs/op
      gorm:     Don't support bulk insert - https://github.com/jinzhu/gorm/issues/255
      zorm:     Don't support bulk insert

    2000 times - Update
      xorm:     0.39s       195846 ns/op    2529 B/op     87 allocs/op
      zorm:     0.51s       253577 ns/op    2232 B/op     32 allocs/op
      gorm:     0.73s       366905 ns/op    9157 B/op    226 allocs/op

  2000 times - Read
      zorm:     0.28s       141890 ns/op    1616 B/op     43 allocs/op
      gorm:     0.45s       223720 ns/op    5931 B/op    138 allocs/op
      xorm:     0.55s       276055 ns/op    8648 B/op    227 allocs/op

  2000 times - MultiRead limit 1000
      zorm:    13.93s      6967146 ns/op  694286 B/op  23054 allocs/op
      gorm:    26.40s     13201878 ns/op 2392826 B/op  57031 allocs/op
      xorm:    30.77s     15382967 ns/op 1637098 B/op  72088 allocs/op
```
