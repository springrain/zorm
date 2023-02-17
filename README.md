## Introduction
![zorm logo](zorm-logo.png)  
This is a lightweight ORM,zero dependency, that supports DM,Kingbase,shentong,TDengine,mysql,postgresql,oracle,mssql,sqlite,db2,clickhouse...

Official website:https://zorm.cn  
source code address:https://gitee.com/chunanyong/zorm  
test case: https://gitee.com/wuxiangege/zorm-examples/  
Video tutorial: https://www.bilibili.com/video/BV1L24y1976U/


``` 
go get gitee.com/chunanyong/zorm 
```  

* Based on native SQL statements, the learning cost is lower  
* [Code generator](https://gitee.com/zhou-a-xing/zorm-generate-struct)	
* The code is concise, the main body is 2500 lines, zero dependency 4000 lines, detailed comments, easy to customize and modify
* <font color=red>Support for transaction propagation, which was the main reason for the birth of ZORM</font>		
* Support dm (dameng), kingbase (jincang), shentong (Shentong), gbase (Nantong), TDengine, mysql, postgresql, oracle, mssql, sqlite, db2, clickhouse...
* Supports multi-database and read/write splitting  
* Joint primary keys are not supported, workarounds are assumed to be no primary keys, and business control is implemented (difficult trade-offs)
* Support seata, HPTX, dbpack distributed transactions, support global transaction hosting, do not modify business code, zero intrusion distributed transactions  
* Support clickhouse, update, delete statements using SQL92 standard syntax.clickhouse-go official driver does not support batch insert syntax, it is recommended to use https://github.com/mailru/go-clickhouse

## Transaction propagation  
Transaction propagation is the core function of ZORM and the main reason why all methods of ZORM have ctx parameters.  
ZORM transaction operations need ```zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {})``` to be explicitly enabled, check transactions before executing closure functions, join transactions if there are transactions in ctx, and create new transactions if there are no transactions in ctx, so you only need to pass the same ctx object to achieve transaction propagation. In special scenarios, if you do not want transaction synchronization, you can declare a new ctx object to do transaction isolation. 

## Description of the source repository
The main libraries of the open source projects I led are all in GitHub, and there are project descriptions on GitHub to guide the jump to GitHub, which also causes the slow growth of the project, after all, there are more GitHub users.  
**Open source has no borders, but developers have their own homeland.**  
Strictly speaking, GitHub is governed by US law https://www.infoq.cn/article/SA72SsSeZBpUSH_ZH8XB
do my best to support the domestic open source community, don't like it, don't spray, thank you!

## Support domestic database  
ZORM spares no effort in adapting to domestic databases, and if you encounter domestic databases that are not adapted or have problems, please feedback to the community and work together to build a domestic software ecosystem

### Da Meng (DM)
- Configure zorm.DataSourceConfig ```DriverName:dm ,Dialect:dm```
- Damon Database Driver: gitee.com/chunanyong/dm
- The TEXT type of Damon will be mapped to ```dm.DmClob```, string cannot be received, and zorm needs to be implemented ```ICustomDriverValueConver``` interface, custom extension processing
```go
import (
	// 00. Introduce the database driver
	"gitee.com/chunanyong/dm"
	"io"
)

// CustomDMText implements ICustomDriverValueConver interface to extend custom types. For example, the TEXT type is mapped to dm.DmClob and cannot be directly received using string
type CustomDMText struct{}

// GetDriverValue Returns an instance of driver.Value, the struct attribute type, based on the database column type
// The structFieldType is passed nil because the map received or field does not exist
func (dmtext CustomDMText) GetDriverValue(ctx context.Context, columnType *sql.ColumnType, structFieldType *reflect.Type) (driver.Value, error) {
	// If you want to use the structFieldType, you need to determine if it is nil
	// if structFieldType != nil {
	// }

	return &dm.DmClob{}, nil
}

// ConverDriverValue database column type, temporary received Value of driver. value returned by GetDriverValue,struct attribute type
// The structFieldType is passed nil because the map received or field does not exist
// Returns a pointer, pointer, pointer that matches the received type value!!!!
func (dmtext CustomDMText) ConverDriverValue(ctx context.Context, columnType *sql.ColumnType, tempDriverValue driver.Value, structFieldType *reflect.Type) (interface{}, error) {
	// If you want to use the structFieldType, you need to determine if it is nil
	// if structFieldType != nil {
	// }

	// Type conversion
	dmClob, isok := tempDriverValue.(*dm.DmClob)
	if !isok {
		return tempDriverValue, errors.New("->ConverDriverValue--> Failed to convert to *dm.DmClob")
	}
	if dmClob == nil || !dmClob.Valid {
		return new(string), nil
	}
	// Get the length
	dmlen, errLength := dmClob.GetLength()
	if errLength != nil {
		return dmClob, errLength
	}

	// int64 is converted to an int
	strInt64 := strconv.FormatInt(dmlen, 10)
	dmlenInt, errAtoi := strconv.Atoi(strInt64)
	if errAtoi != nil {
		return dmClob, errAtoi
	}

	// Read the string
	str, errReadString := dmClob.ReadString(1, dmlenInt)

	// Handle EOF errors caused by empty strings or NULL value
	if errReadString == io.EOF {
		return new(string), nil
	}

	return &str, errReadString
}
// RegisterCustomDriverValueConver registered custom field processing logic, used to drive not directly convert scenarios, such as the TEXT of the dream cannot directly into a string
// It's usually registered in the init method
func init() {
	// dialectColumnType is a Dialect.FieldType, such as dm.TEXT
	zorm.RegisterCustomDriverValueConver("dm.TEXT", CustomDMText{})
}
```
### Kingbase
- Configure zorm.DataSourceConfig ```DriverName:kingbase ,Dialect:kingbase```
- Golden warehouse official drive: https://www.kingbase.com.cn/qd/index.htmhttps://bbs.kingbase.com.cn/thread-14457-1-1.html?_dsign=87f12756
- The Kingbase 8 core is based on PostgreSQL 9.6 and can be tested using https://github.com/lib/pq, and the official driver is recommended for production environments
- Note that ora_input_emptystr_isnull = false or ora_input_emptystr_isnull = on in the data/kingbase.conf of the database (according to the version), because golang does not have a null value, the general database is not null, golang's string defaults to '', if this is set to true, The database will set the value to null, which conflicts with the field property not null, so an error is reported.
  After the configuration file is modified, restart the database.
- Thanks to [@Jin](https://gitee.com/GOODJIN) for testing and suggestions.

### Shentong (shentong)
It is recommended to use official driver, configure zorm.DataSourceConfig ```DriverName:aci ,Dialect:shentong```  

### Nantong (gbase)
~~The official Go driver has not been found yet. Please configure it zorm.DataSourceConfig DriverName:gbase ,Dialect:gbase~~  
Use odbc driver for the time being, ```DriverName:odbc ,Dialect:gbase```

### TDengine
- Since the TDengine driver does not support transactions, you need to set this setting ```DisableTransaction=true```
- Configure zorm.DataSourceConfig ```DriverName:taosSql/taosRestful, Dialect:tdengine```
- zorm.DataSourceConfig```TDengineInsertsColumnName```TDengine batch insert statement whether there is a column name. The default false has no column name, and the insertion value and database column order are consistent, reducing the length of the statement
- Test case: https://www.yuque.com/u27016943/nrgi00/dnru3f
- TDengine is included: https://github.com/taosdata/awesome-tdengine/#orm

## Database scripts and entity classes  
Generate entity classes or write them manually, we recommend using a code generator https://gitee.com/zhou-a-xing/zorm-generate-struct  

```go 

package testzorm

import (
	"time"

	"gitee.com/chunanyong/zorm"
)

// Build a list sentence

/*

DROP TABLE IF EXISTS `t_demo`;
CREATE TABLE `t_demo` (
`id` varchar(50) NOT NULL COMMENT 'primary key ',
`userName` varchar(30) NOT NULL COMMENT 'name ',
`password` varchar(50) NOT NULL COMMENT 'password ',
`createTime` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
`active` int COMMENT 'Whether it is valid (0 no,1 yes)',
 PRIMARY KEY (`id`)
ENGINE = InnoDB CHARACTER SET = utf8mb4 COMMENT = 'example';

*/

// demoStructTableName Table name constant for direct call
const demoStructTableName = "t_demo"

// demoStruct example
type demoStruct struct {
	// Introduce the default struct to isolate method changes to the IEntityStruct
	zorm.EntityStruct

	// Id Primary key
	Id string `column:"id"`

	// UserName Specifies the name
	UserName string `column:"userName"`

	// Password Password
	Password string `column:"password"`

	// CreateTime <no value>
	CreateTime time.Time `column:"createTime"`

	// Active Whether it is valid (0 No,1 yes)
	// Active int `column:"active"`

	// -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- - end of the database field, custom fields to write in the following -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- // 
	// If the queried field is not found in the column tag, it is mapped to the struct property by name (case insensitive, support _ _ to _ hump)

	// Simulates the custom field Active
	Active int
}

// GetTableName Gets the table name
// IEntityStruct interface method, entity class needs to implement!!
func (entity *demoStruct) GetTableName() string {
	return demoStructTableName
}

// GetPKColumnName Gets the name of the primary key field of the database table. Because to be compatible with Map, can only be the database field name
// Joint primary key is not supported. It can be considered that there is no primary key and service control can be realized (hard choice).
// If you do not have a primary key, you need to implement this method as well
// IEntityStruct interface method, entity class needs to implement!!
func (entity *demoStruct) GetPKColumnName() string {
	// If there is no primary key
	// return ""
	return "id"
}

// newDemoStruct creates a default object
func newDemoStruct() demoStruct {
	demo := demoStruct{
		// if Id == ", "save zorm will call zorm.FuncGenerateStringID(ctx), the default time stamp + random number, also can define your own implementations, such as zorm.FuncGenerateStringID = funcmyId
		Id:		 zorm.FuncGenerateStringID(ctx),
		UserName:   "defaultUserName",
		Password:   "defaultPassword",
		Active:	 1,
		CreateTime: time.Now(),
	}
	return demo
}
```

## Test cases are documents
https://gitee.com/wuxiangege/zorm-examples  
```go  

// testzorm uses native sql statements with no restrictions on sql syntax. The statement uses Finder as the carrier
// Universal use of placeholders? zorm automatically replaces placeholders based on the database type, such as the postgresql database? Replace it with $1,$2...
// zorm uses the ctx context.Context parameter to propagate the transaction. ctx is passed in from the web layer. For example, gin's c.Request.Context()
// Transaction must be explicitly enabled using zorm.Transaction(ctx, func(ctx context.context) (interface{}, error) {})
package testzorm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gitee.com/chunanyong/zorm"

	// 00. Introduce the database driver
	_ "github.com/go-sql-driver/mysql"
)

// DBDAOs represent one database. If there are multiple databases, multiple DBDAOs are declared
var dbDao *zorm.DBDao

// 01. Initialize the DBDao
func init() {

	// Customize zorm log output
	// zorm.LogCallDepth = 4 // Level of log calls
	// zorm.FuncLogError = myFuncLogError // Function to log exceptions
	// zorm.FuncLogPanic = myFuncLogPanic // To log panic, the default is defaultLogError
	// zorm.FuncPrintSQL = myFuncPrintSQL // A function that prints sql

	// Reassign the FuncPrintSQL function to a custom log output format
	// log.SetFlags(log.LstdFlags)
	// zorm.FuncPrintSQL = zorm.FuncPrintSQL

	// Custom primary key generation
	// zorm.FuncGenerateStringID=funcmyId

	// Customize the Tag column name
	// zorm.FuncWrapFieldTagName=funcmyTagName

	// Custom decimal type implementation
	// zorm.FuncDecimalValue=funcmyDecimal

	// the Go database driver list: https://github.com/golang/go/wiki/SQLDrivers

	// dbDaoConfig Configure the database. This is just a simulation, the production should be reading the configuration configuration file and constructing the DataSourceConfig
	dbDaoConfig := zorm.DataSourceConfig{
		// DSN database connection string. parseTime=true is automatically converted to time format. The default query is the []byte array
		DSN: "root:root@tcp(127.0.0.1:3306)/zorm?charset=utf8&parseTime=true",
		// DriverName database driver name: mysql, postgres, oracle(go-ora), essentially, sqlite3, go_ibm_db, clickhouse, dm, kingbase, aci, taosSql | taosRestful Correspond to Dialect
		// sql.Open(DriverName,DSN) DriverName is the first string parameter of the sql.Open of the driver. The value can be obtained according to the actual conditions of the driver
		DriverName: "mysql",
		// the Dialect database Dialect: mysql, postgresql, oracle, MSSQL, sqlite, db2, clickhouse, dm, kingbase, shentong, tdengine and DriverName corresponding
		Dialect: "mysql",
		// MaxOpenConns The default maximum number of database connections is 50
		MaxOpenConns: 50,
		// MaxIdleConns The default maximum number of idle connections is 50
		MaxIdleConns: 50,
		// ConnMaxLifetimeSecond Connection survival seconds. Default 600(10 minutes) after the connection is destroyed and rebuilt. Prevent the database from voluntarily disconnecting, resulting in dead connections. MySQL default wait_timeout 28800 seconds (8 hours)
		ConnMaxLifetimeSecond: 600,
		// SlowSQLMillis slow sql time threshold, in milliseconds. A value less than 0 disables SQL statement output. If the value is equal to 0, only SQL statements are output and the execution time is not calculated. A value greater than 0 is used to calculate the SQL execution time and >=SlowSQLMillis value
		SlowSQLMillis: 0,
		// DefaultTxOptions Default configuration of transaction isolation level, which defaults to nil
		// DefaultTxOptions: nil,
		// If distributed transactions are used, the default configuration is recommended
		// DefaultTxOptions: &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false},

		// FuncGlobalTransaction seata/hptx An adaptation function of a globally distributed transaction that returns the implementation of the IGlobalTransaction interface
		// business must call ctx, _ = zorm.BindContextEnableGlobalTransaction (ctx) on the global distribution of transactions
		// FuncGlobalTransaction : MyFuncGlobalTransaction,

		// SQLDB uses an existing database connection and has a higher priority than DSN
		// SQLDB : nil,

		// DisableTransaction disables transactions. The default value is false. If DisableTransaction=true is set, the Transaction method becomes invalid and no transaction is required. Some databases, such as TDengine, do not support transactions
		// Disable transactions should have the driver forgery transaction API, there should be no orm implementation,clickhouse's driver does just that
		// DisableTransaction :false,

		// TDengineInsertsColumnName Whether there are column names in the TDengine batch insert statement. The default false has no column name, and the insertion value and database column order are consistent, reducing the length of the statement
		// TDengineInsertsColumnName :false,
	}

	// Create dbDao based on dbDaoConfig. Perform this operation once for each database. The first database is defaultDao and the subsequent zorm.xxx method uses defaultDao by default
	dbDao, _ = zorm.NewDBDao(&dbDaoConfig)
}

// TestInsert 02. Test save the Struct object
func TestInsert(t *testing.T) {
	// ctx is generally a request for one ctx, normally there should be a web layer in, such as gin's c. Request.Context()
	var ctx = context.Background()

	// You need to start the transaction manually. If the error returned by the anonymous function is not nil, the transaction will be rolled back. If the DisableTransaction=true parameter is set, the Transaction method becomes invalid and no transaction is required
	// if zorm.DataSourceConfig.DefaultTxOptions configuration does not meet the requirements, can be in zorm, Transaction before Transaction method set the Transaction isolation level
	// such as ctx, _ := dbDao BindContextTxOptions (ctx, & SQL TxOptions {Isolation: SQL LevelDefault, ReadOnly: False}), if txOptions is nil, the use of zorm.DataSourceConfig.DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		// Create a demo object
		demo := newDemoStruct()

		// Save the object. The parameter is a pointer to the object. If the primary key is increment, the value is assigned to the primary key property of the object
		_, err := zorm.Insert(ctx, &demo)

		// If err is not returned nil, the transaction is rolled back
		return nil, err
	})
	// Mark the test failed
	if err != nil {
		t.Errorf("Error:%v", err)
	}
}

// TestInsertSlice 03. Tests batch save Struct object Slice
// The primary key property in the Struct object cannot be assigned if the primary key is autoincrement
func TestInsertSlice(t *testing.T) {
	// ctx is generally a request for one ctx, normally there should be a web layer in, such as gin's c. Request.Context()
	var ctx = context.Background()

	// You need to start the transaction manually. If the error returned by the anonymous function is not nil, the transaction will be rolled back. If the DisableTransaction=true parameter is set, the Transaction method becomes invalid and no transaction is required
	// if zorm.DataSourceConfig.DefaultTxOptions configuration does not meet the requirements, can be in zorm, Transaction before Transaction method set the Transaction isolation level
	// such as ctx, _ := dbDao BindContextTxOptions (ctx, & SQL TxOptions {Isolation: SQL LevelDefault, ReadOnly: False}), if txOptions is nil, the use of zorm.DataSourceConfig.DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

		// slice stores the type zorm.IEntityStruct!!! Use the IEntityStruct interface, compatible with Struct entity classes
		demoSlice := make([]zorm.IEntityStruct,0)

		// Create object 1
		demo1 := newDemoStruct()
		demo1.UserName = "demo1"
		// Create object 2
		demo2 := newDemoStruct()
		demo2.UserName = "demo2"

		demoSlice = append(demoSlice, &demo1, &demo2)

		// Batch save objects. If the primary key is auto-increment, the auto-increment ID cannot be saved to the object.
		_, err := zorm.InsertSlice(ctx, demoSlice)

		// If err is not returned nil, the transaction is rolled back
		return nil, err
	})
	// Mark the test failed
	if err != nil {
		t.Errorf("Error:%v", err)
	}
}

// TestInsertEntityMap 04. Test to save an EntityMap object for scenarios where it is not convenient to use struct. Use Map as the carrier
func TestInsertEntityMap(t *testing.T) {
	// ctx is generally a request for one ctx, normally there should be a web layer in, such as gin's c. Request.Context()
	var ctx = context.Background()

	// You need to start the transaction manually. If the error returned by the anonymous function is not nil, the transaction will be rolled back. If the DisableTransaction=true parameter is set, the Transaction method becomes invalid and no transaction is required
	// if zorm.DataSourceConfig.DefaultTxOptions configuration does not meet the requirements, can be in zorm, Transaction before Transaction method set the Transaction isolation level
	// such as ctx, _ := dbDao BindContextTxOptions (ctx, & SQL TxOptions {Isolation: SQL LevelDefault, ReadOnly: False}), if txOptions is nil, the use of zorm.DataSourceConfig.DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		// To create an EntityMap, pass in the table name
		entityMap := zorm.NewEntityMap(demoStructTableName)
		// Set the primary key name
		entityMap.PkColumnName = "id"
		// If it is an increment sequence, set the value of the sequence
		// entityMap.PkSequence = "mySequence"

		// Set Sets the field values of the database
		// If the primary key is an increment or sequence, do not set the value of the entityMap.Set primary key
		entityMap.Set("id", zorm.FuncGenerateStringID(ctx))
		entityMap.Set("userName", "entityMap-userName")
		entityMap.Set("password", "entityMap-password")
		entityMap.Set("createTime", time.Now())
		entityMap.Set("active", 1)

		// Execute
		_, err := zorm.InsertEntityMap(ctx, entityMap)

		// If err is not returned nil, the transaction is rolled back
		return nil, err
	})
	// Mark the test failed
	if err != nil {
		t.Errorf("Error:%v", err)
	}
}


// TestInsertEntityMapSlice 05. Test batch save []IEntityMap for scenarios where it is not convenient to use struct, using Map as carrier
func TestInsertEntityMapSlice(t *testing.T) {
	// ctx is generally a request for one ctx, normally there should be a web layer in, such as gin's c. Request.Context()
	var ctx = context.Background()

	_, err := Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		entityMapSlice := make([]IEntityMap, 0)
		entityMap1 := NewEntityMap(demoStructTableName)
		entityMap1.PkColumnName = "id"
		entityMap1.Set("id", zorm.FuncGenerateStringID(ctx))
		entityMap1.Set("userName", "entityMap-userName1")
		entityMap1.Set("password", "entityMap-password1")
		entityMap1.Set("createTime", time.Now())
		entityMap1.Set("active", 1)

		entityMap2 := NewEntityMap(demoStructTableName)
		entityMap2.PkColumnName = "id"
		entityMap2.Set("id", zorm.FuncGenerateStringID(ctx))
		entityMap2.Set("userName", "entityMap-userName2")
		entityMap2.Set("password", "entityMap-password2")
		entityMap2.Set("createTime", time.Now())
		entityMap2.Set("active", 2)

		entityMapSlice = append(entityMapSlice, entityMap1 ,entityMap2)

		// Execute
		_, err := zorm.InsertEntityMapSlice(ctx, entityMapSlice)

		// If err is not returned nil, the transaction is rolled back
		return nil, err
	})
	// Mark the test failed
	if err != nil {
		t.Errorf("Error:%v", err)
	}
}

// TestQueryRow 06. Test query a struct object
func TestQueryRow(t *testing.T) {
	// ctx is generally a request for one ctx, normally there should be a web layer in, such as gin's c. Request.Context()
	var ctx = context.Background()

	// Declares a pointer to an object that holds the returned data
	demo := demoStruct{}

	// finder used to construct the query
	// finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo
	// finder := zorm.NewSelectFinder(demoStructTableName, "id,userName") // select id,userName from t_demo
	finder := zorm.NewFinder().Append("SELECT * FROM " + demoStructTableName) // select * from t_demo
	// finder by default, sql injection checking is enabled to disallow concatenation of 'single quotes in statements. You can set finder.injectioncheck = false to undo the restriction

	// finder.Append The first argument is the statement and the following arguments are the corresponding values in the correct order. Uniform use of statements? zorm handles database differences
	// in (?) Arguments must have () parentheses, not in?
	finder.Append("WHERE id=? and active in(?) ", "20210630163227149563000042432429", []int{0, 1})

	// How do I use like
	// finder.Append("WHERE id like ? ", "20210630163227149563000042432429%")

	// If the value of "has" is true, the database has data
	has, err := zorm.QueryRow(ctx, finder, &demo)

	if err != nil { // Mark the test failed
		t.Errorf("Error:%v", err)
	}
	// Print the result
	fmt.Println(demo)
}

// TestQueryRowMap 07. Test query map receives results. It is flexible for scenarios that are not suitable for structs
func TestQueryRowMap(t *testing.T) {
	// ctx is generally a request for one ctx, normally there should be a web layer in, such as gin's c. Request.Context()
	var ctx = context.Background()

	// finder used to construct the query
	// finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo
	finder := zorm.NewFinder().Append("SELECT * FROM " + demoStructTableName) // select * from t_demo
	// finder.Append The first argument is the statement and the following arguments are the corresponding values in the correct order. Uniform use of statements? zorm handles database differences
	// in (?) Arguments must have () parentheses, not in?
	finder.Append("WHERE id=? and active in(?) ", "20210630163227149563000042432429", []int{0, 1})
	// Run the query
	resultMap, err := zorm.QueryRowMap(ctx, finder)

	if err != nil { // Mark the test failed
		t.Errorf("Error:%v", err)
	}
	// Print the result
	fmt.Println(resultMap)
}

// TestQuery 08. Test the list of query objects
func TestQuery(t *testing.T) {
	// ctx is generally a request for one ctx, normally there should be a web layer in, such as gin's c. Request.Context()
	var ctx = context.Background()

	// Create a slice for receiving results
	list := make([]demoStruct, 0)

	// finder used to construct the query
	// finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo
	finder := zorm.NewFinder().Append("SELECT id FROM " + demoStructTableName) // select * from t_demo
	// Create a paging object. After the query is complete, the page object can be directly used by the front-end paging component
	page := zorm.NewPage()
	page.PageNo = 1   // Query page 1. The default value is 1
	page.PageSize = 20 // 20 per page. The default is 20

	// The total number of entries is not queried
	// finder.SelectTotalCount = false

	// You can manually specify paging statements if they are particularly complex statements that cause count statement construction to fail
	// countFinder := zorm.NewFinder().Append("select count(*) from (")
	// countFinder.AppendFinder(finder)
	// countFinder.Append(") tempcountfinder")
	// finder.CountFinder = countFinder

	// Run the query
	err := zorm.Query(ctx, finder, &list, page)
	if err != nil { // Mark the test failed
		t.Errorf("Error:%v", err)
	}
	// Print the result
	fmt.Println("Total number of items :", page.TotalCount, "List :", list)
}

// TestQueryMap 09. Test query map list. Used in the scenario where struct is not convenient
func TestQueryMap(t *testing.T) {
	// ctx is generally a request for one ctx, normally there should be a web layer in, such as gin's c. Request.Context()
	var ctx = context.Background()

	// finder used to construct the query
	// finder := zorm.NewSelectFinder(demoStructTableName) // select * from t_demo
	finder := zorm.NewFinder().Append("SELECT * FROM " + demoStructTableName) // select * from t_demo
	// Create a paging object. After the query is complete, the page object can be directly used by the front-end paging component
	page := zorm.NewPage()
	page.PageNo = 1   // Query page 1. The default value is 1
	page.PageSize = 20 // 20 per page. The default is 20

	// The total number of entries is not queried
	// finder.SelectTotalCount = false
	
	// You can manually specify paging statements if they are particularly complex statements that cause count statement construction to fail
	// countFinder := zorm.NewFinder().Append("select count(*) from (")
	// countFinder.AppendFinder(finder)
	// countFinder.Append(") tempcountfinder")
	// finder.CountFinder = countFinder

	// Run the query
	listMap, err := zorm.QueryMap(ctx, finder, page)
	if err != nil { // Mark the test failed
		t.Errorf("Error:%v", err)
	}
	// Print the result
	fmt.Println("Total number of items :", page.TotalCount, "List :", listMap)
}

// TestUpdateNotZeroValue 10. Update the struct object with only the non-zero fields. The primary key must have a value
func TestUpdateNotZeroValue(t *testing.T) {
	// ctx is generally a request for one ctx, normally there should be a web layer in, such as gin's c. Request.Context()
	var ctx = context.Background()

	// You need to start the transaction manually. If the error returned by the anonymous function is not nil, the transaction will be rolled back. If the DisableTransaction=true parameter is set, the Transaction method becomes invalid and no transaction is required
	// if zorm.DataSourceConfig.DefaultTxOptions configuration does not meet the requirements, can be in zorm, Transaction before Transaction method set the Transaction isolation level
	// such as ctx, _ := dbDao BindContextTxOptions (ctx, & SQL TxOptions {Isolation: SQL LevelDefault, ReadOnly: False}), if txOptions is nil, the use of zorm.DataSourceConfig.DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		// Declares a pointer to an object used to update data
		demo := demoStruct{}
		demo.Id = "20210630163227149563000042432429"
		demo.UserName = "UpdateNotZeroValue"

		// UPDATE "sql":"UPDATE t_demo SET userName=? WHERE id=?" ,"args":["UpdateNotZeroValue","20210630163227149563000042432429"]
		_, err := zorm.UpdateNotZeroValue(ctx, &demo)

		// If err is not returned nil, the transaction is rolled back
		return nil, err
	})
	if err != nil { // Mark the test failed
		t.Errorf("Error:%v", err)
	}

}

// TestUpdate 11. Update the struct object, updating all fields. The primary key must have a value
func TestUpdate(t *testing.T) {
	// ctx is generally a request for one ctx, normally there should be a web layer in, such as gin's c. Request.Context()
	var ctx = context.Background()

	// You need to start the transaction manually. If the error returned by the anonymous function is not nil, the transaction will be rolled back. If the DisableTransaction=true parameter is set, the Transaction method becomes invalid and no transaction is required
	// if zorm.DataSourceConfig.DefaultTxOptions configuration does not meet the requirements, can be in zorm, Transaction before Transaction method set the Transaction isolation level
	// such as ctx, _ := dbDao BindContextTxOptions (ctx, & SQL TxOptions {Isolation: SQL LevelDefault, ReadOnly: False}), if txOptions is nil, the use of zorm.DataSourceConfig.DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

		// Declares a pointer to an object used to update data
		demo := demoStruct{}
		demo.Id = "20210630163227149563000042432429"
		demo.UserName = "TestUpdate"

		_, err := zorm.Update(ctx, &demo)

		// If err is not returned nil, the transaction is rolled back
		return nil, err
	})
	if err != nil { // Mark the test failed
		t.Errorf("Error:%v", err)
	}
}

// TestUpdateFinder 12. With finder update,zorm's most flexible way of writing any update statement, even manually writing insert statements
func TestUpdateFinder(t *testing.T) {
	// ctx is generally a request for one ctx, normally there should be a web layer in, such as gin's c. Request.Context()
	var ctx = context.Background()

	// You need to start the transaction manually. If the error returned by the anonymous function is not nil, the transaction will be rolled back. If the DisableTransaction=true parameter is set, the Transaction method becomes invalid and no transaction is required
	// if zorm.DataSourceConfig.DefaultTxOptions configuration does not meet the requirements, can be in zorm, Transaction before Transaction method set the Transaction isolation level
	// such as ctx, _ := dbDao BindContextTxOptions (ctx, & SQL TxOptions {Isolation: SQL LevelDefault, ReadOnly: False}), if txOptions is nil, the use of zorm.DataSourceConfig.DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		// finder := zorm.NewUpdateFinder(demoStructTableName) // UPDATE t_demo SET
		// finder := zorm.NewDeleteFinder(demoStructTableName) // DELETE FROM t_demo
		finder := zorm.NewFinder().Append("UPDATE").Append(demoStructTableName).Append("SET") // UPDATE t_demo SET
		finder.Append("userName=? ,active=?", "TestUpdateFinder", 1).Append("WHERE id=?", "20210630163227149563000042432429")

		// UPDATE "sql":"UPDATE t_demo SET userName=? ,active=? WHERE id=?" ,"args":["TestUpdateFinder",1,"20210630163227149563000042432429"]
		_, err := zorm.UpdateFinder(ctx, finder)

		// If err is not returned nil, the transaction is rolled back
		return nil, err
	})
	if err != nil { // Mark the test failed
		t.Errorf("Error:%v", err)
	}

}

// TestUpdateEntityMap 13. Update an EntityMap. The primary key must have a value
func TestUpdateEntityMap(t *testing.T) {
	// ctx is generally a request for one ctx, normally there should be a web layer in, such as gin's c. Request.Context()
	var ctx = context.Background()

	// You need to start the transaction manually. If the error returned by the anonymous function is not nil, the transaction will be rolled back. If the DisableTransaction=true parameter is set, the Transaction method becomes invalid and no transaction is required
	// if zorm.DataSourceConfig.DefaultTxOptions configuration does not meet the requirements, can be in zorm, Transaction before Transaction method set the Transaction isolation level
	// such as ctx, _ := dbDao BindContextTxOptions (ctx, & SQL TxOptions {Isolation: SQL LevelDefault, ReadOnly: False}), if txOptions is nil, the use of zorm.DataSourceConfig.DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		// To create an EntityMap, pass in the table name
		entityMap := zorm.NewEntityMap(demoStructTableName)
		// Set the primary key name
		entityMap.PkColumnName = "id"
		// Set Sets the field value of the database. The primary key must have a value
		entityMap.Set("id", "20210630163227149563000042432429")
		entityMap.Set("userName", "TestUpdateEntityMap")
		// UPDATE "sql":"UPDATE t_demo SET userName=? WHERE id=?" ,"args":["TestUpdateEntityMap","20210630163227149563000042432429"]
		_, err := zorm.UpdateEntityMap(ctx, entityMap)

		// If err is not returned nil, the transaction is rolled back
		return nil, err
	})
	if err != nil { // Mark the test failed
		t.Errorf("Error:%v", err)
	}

}

// TestDelete 14. Delete a struct object. The primary key must have a value
func TestDelete(t *testing.T) {
	// ctx is generally a request for one ctx, normally there should be a web layer in, such as gin's c. Request.Context()
	var ctx = context.Background()

	// You need to start the transaction manually. If the error returned by the anonymous function is not nil, the transaction will be rolled back. If the DisableTransaction=true parameter is set, the Transaction method becomes invalid and no transaction is required
	// if zorm.DataSourceConfig.DefaultTxOptions configuration does not meet the requirements, can be in zorm, Transaction before Transaction method set the Transaction isolation level
	// such as ctx, _ := dbDao BindContextTxOptions (ctx, & SQL TxOptions {Isolation: SQL LevelDefault, ReadOnly: False}), if txOptions is nil, the use of zorm.DataSourceConfig.DefaultTxOptions
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		demo := demoStruct{}
		demo.Id = "20210630163227149563000042432429"

		// "sql":"DELETE FROM t_demo WHERE id=?" ,"args":["20210630163227149563000042432429"]
		_, err := zorm.Delete(ctx, &demo)

		// If err is not returned nil, the transaction is rolled back
		return nil, err
	})
	if err != nil { // Mark the test failed
		t.Errorf("Error:%v", err)
	}

}

// TestProc 15. Test calls the stored procedure
func TestProc(t *testing.T) {
	// ctx is generally a request for one ctx, normally there should be a web layer in, such as gin's c. Request.Context()
	var ctx = context.Background()

	demo := demoStruct{}
	finder := zorm.NewFinder().Append("call testproc(?)", "u_10001")
	zorm.QueryRow(ctx, finder, &demo)
	fmt.Println(demo)
}

// TestFunc 16. Test calls custom functions
func TestFunc(t *testing.T) {
	// ctx is generally a request for one ctx, normally there should be a web layer in, such as gin's c. Request.Context()
	var ctx = context.Background()

	userName := ""
	finder := zorm.NewFinder().Append("select testfunc(?)", "u_10001")
	zorm.QueryRow(ctx, finder, &userName)
	fmt.Println(userName)
}

// TestOther 17. Some other instructions. Thank you very much for seeing this line
func TestOther(t *testing.T) {
	// ctx is generally a request for one ctx, normally there should be a web layer in, such as gin's c. Request.Context()
	var ctx = context.Background()

	// Scenario 1. Multiple databases. The dbDao of the corresponding database calls BindContextDBConnection, binds the database connection to the returned ctx, and passes ctx to zorm's function
	// You can also rewrite the FuncReadWriteStrategy function to return the DBDao of the specified database by setting a different key via ctx
	newCtx, err := dbDao.BindContextDBConnection(ctx)
	if err != nil { // Mark the test failed
		t.Errorf("Error:%v", err)
	}

	finder := zorm.NewFinder().Append("SELECT * FROM " + demoStructTableName) // select * from t_demo
	// Pass the new newCtx to zorm's function
	list, _ := zorm.QueryMap(newCtx, finder, nil)
	fmt.Println(list)

	// Scenario 2. Read/write separation of a single database. Set the read-write separation policy function.
	zorm.FuncReadWriteStrategy = myReadWriteStrategy

	// Scenario 3. If multiple databases exist and read and write data are separated from each other, perform this operation according to Scenario 1.
	// You can also rewrite the FuncReadWriteStrategy function to return the DBDao of the specified database by setting a different key via ctx

}

// myReadWriteStrategy Database read-write strategy rwType=0 read,rwType=1 write
// You can also set different keys through ctx to return the DBDao of the specified database
func myReadWriteStrategy(ctx context.Context, rwType int) (*zorm.DBDao, error) {
	// Return the required read/write dao based on your business scenario. This function is called every time a database connection is needed
	// if rwType == 0 {
	// return dbReadDao
	// }
	// return dbWriteDao

	return dbDao, nil
}

// --------------------------------------------
// ICustomDriverValueConver interface, see examples of DaMeng

// --------------------------------------------
// OverrideFunc Rewrite the functions of ZORM, when you use this function, you have to know what you are doing

```  
## Global transaction
### seata-go CallbackWithCtx function mode
```go
// DataSourceConfig configures DefaultTxOptions
// DefaultTxOptions: &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false},

// Import the seata-go dependency package
import (
	"context"
	"fmt"
	"time"

	"github.com/seata/seata-go/pkg/client"
	"github.com/seata/seata-go/pkg/tm"
	seataSQL "github.com/seata/seata-go/pkg/datasource/sql" //Note: zorm's DriverName: seataSQL.SeataATMySQLDriver, !!!!
)

// Path of the configuration file
var configPath = "./conf/client.yml"

func main() {

	// Initialize the configuration
	conf := config.InitConf(configPath)
	// Initialize the zorm database
	// note: zorm DriverName: seataSQL SeataATMySQLDriver,!!!!!!!!!!
	initZorm()

	// Start distributed transactions
	tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
		Name:	"ATSampleLocalGlobalTx",
		Timeout: time.Second * 30,
	}, CallbackWithCtx)
	// CallbackWithCtx business callback definition
	// type CallbackWithCtx func(ctx context.Context) error


	// Get the XID after the transaction is started. This can be passed through gin's header, or otherwise
	// xid:=tm.GetXID(ctx)
	// tm.SetXID(ctx, xid)

	// If the gin framework is used, middleware binding parameters can be used
	// r.Use(ginmiddleware.TransactionMiddleware())
}

```

### seata-go transaction hosting mode

```go
// Do not use CallbackWithCtx function,zorm to achieve transaction management, no modification of business code, zero intrusion to achieve distributed transactions


// The distributed transaction must be started manually and must be invoked before the local transaction is started
ctx,_ = zorm.BindContextEnableGlobalTransaction(ctx)
// Distributed transaction sample code
_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

	// Get the XID of the current distributed transaction. Don't worry about how, if it is a distributed transaction environment, the value will be set automatically
	// xid := ctx.Value("XID").(string)

	// Pass the xid to the third party application
	// req.Header.Set("XID", xid)

	// If err is not returned nil, local and distributed transactions are rolled back
	return nil, err
})

// /---------- Third-party application -------/ //

	// Do not use the middleware provided by seata-go by default, just ctx binding XID!!!
	//// r.Use(ginmiddleware.TransactionMiddleware())
	xid := c.GetHeader(constant.XidKey)
	ctx = context.WithValue(ctx, "XID", xid)

	// The distributed transaction must be started manually and must be invoked before the local transaction is started
	ctx,_ = zorm.BindContextEnableGlobalTransaction(ctx)
	// ctx invokes the business transaction after binding the XID
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

	// Business code......

	// If err is not returned nil, local and distributed transactions are rolled back
	return nil, err
})

// It is recommended that the following code be placed in a separate file
// ... // 

// ZormGlobalTransaction packaging seata *tm.GlobalTransactionManager, zorm.IGlobalTransaction interface
type ZormGlobalTransaction struct {
	*tm.GlobalTransactionManager
}

// MyFuncGlobalTransaction zorm A function that ADAPTS a seata globally distributed transaction
// important!!!! Need to configure the zorm.DataSourceConfig.FuncGlobalTransaction = MyFuncGlobalTransaction important!!!!!!
func MyFuncGlobalTransaction(ctx context.Context) (zorm.IGlobalTransaction, context.Context, context.Context, error) {
	// Create a seata-go transaction
	globalTx := tm.GetGlobalTransactionManager()
	// Use the zorm.IGlobalTransaction interface object to wrap distributed transactions and isolate the seata-go dependencies
	globalTransaction := &ZormGlobalTransaction{globalTx}

	if tm.IsSeataContext(ctx) {
		return globalTransaction, ctx, ctx, nil
	}
	// open global transaction for the first time
	ctx = tm.InitSeataContext(ctx)
	// There is a request to come in, manually get the XID
	xidObj := ctx.Value("XID")
	if xidObj ! = nil {
		xid := xidObj.(string)
		tm.SetXID(ctx, xid)
	}
	tm.SetTxName(ctx, "ATSampleLocalGlobalTx")

	// use new context to process current global transaction.
	if tm.IsGlobalTx(ctx) {
		globalRootContext := transferTx(ctx)
		return globalTransaction, ctx, globalRootContext, nil
	}
	return globalTransaction, ctx, ctx, nil
}

// IGlobalTransaction managed global distributed transaction interface (zorm.IGlobalTransaction). seata and hptx currently implement the same code, only the reference implementation package is different

// BeginGTX Starts global distributed transactions
func (gtx *ZormGlobalTransaction) BeginGTX(ctx context.Context, globalRootContext context.Context) error {
	//tm.SetTxStatus(globalRootContext, message.GlobalStatusBegin)
	err := gtx.Begin(globalRootContext, time.Second*30)
	return err
}

// CommitGTX Commit global distributed transactions
func (gtx *ZormGlobalTransaction) CommitGTX(ctx context.Context, globalRootContext context.Context) error {
	gtr := tm.GetTx(globalRootContext)
	return gtx.Commit(globalRootContext, gtr)
}

// RollbackGTX rolls back globally distributed transactions
func (gtx *ZormGlobalTransaction) RollbackGTX(ctx context.Context, globalRootContext context.Context) error {
	gtr := tm.GetTx(globalRootContext)
	// If it is the Participant role, change it to the Launcher role to allow branch transactions to submit global transactions.
	if gtr.TxRole != tm.Launcher {
		gtr.TxRole = tm.Launcher
	}
	return gtx.Rollback(globalRootContext, gtr)
}
// GetGTXID Gets the XID of the globally distributed transaction
func (gtx *ZormGlobalTransaction) GetGTXID(ctx context.Context, globalRootContext context.Context) (string.error) {
	return tm.GetXID(globalRootContext), nil
}

// transferTx transfer the gtx into a new ctx from old ctx.
// use it to implement suspend and resume instead of seata java
func transferTx(ctx context.Context) context.Context {
	newCtx := tm.InitSeataContext(context.Background())
	tm.SetXID(newCtx, tm.GetXID(ctx))
	return newCtx
}

// ... // 
```


### hptx proxy mode
[in hptx proxy mode for zorm use example](https://github.com/CECTC/hptx-samples/tree/main/http_proxy_zorm)   
```go
// DataSourceConfig configures DefaultTxOptions
// DefaultTxOptions: &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false},

// Introduce the hptx dependency package
import (
	"github.com/cectc/hptx"
	"github.com/cectc/hptx/pkg/config"
	"github.com/cectc/hptx/pkg/resource"
	"github.com/cectc/mysql"
	"github.com/cectc/hptx/pkg/tm"

	gtxContext "github.com/cectc/hptx/pkg/base/context"
)

// Path of the configuration file
var configPath = "./conf/config.yml"

func main() {

	// Initialize the configuration
	hptx.InitFromFile(configPath)
	
	// Register the mysql driver
	mysql.RegisterResource(config.GetATConfig().DSN)
	resource.InitATBranchResource(mysql.GetDataSourceManager())
	// sqlDB, err := sql.Open("mysql", config.GetATConfig().DSN)


	// After the normal initialization of zorm, be sure to put it after the hptx mysql initialization!!

	// ... // 
	// tm register transaction service, refer to the official example (transaction hosting is mainly to remove proxy, zero intrusion on the business)
	tm.Implement(svc.ProxySvc)
	// ... // 


	// Get the hptx rootContext
	// rootContext := gtxContext.NewRootContext(ctx)
	// rootContext := ctx.(*gtxContext.RootContext)

	// Create an hptx transaction
	// globalTx := tm.GetCurrentOrCreate(rootContext)

	// Start the transaction
	// globalTx. BeginWithTimeoutAndName (int32 (6000), "name of the transaction," rootContext)

	// Get the XID after the transaction is started. This can be passed through the gin header, or otherwise
	// xid:=rootContext.GetXID()

	// If using gin frame, get ctx
	// ctx := c.Request.Context()

	// Accept the XID passed and bind it to the local ctx
	// ctx =context.WithValue(ctx,mysql.XID,xid)
}
```

### hptx transaction hosting mode
[zorm transaction hosting hptx example](https://github.com/CECTC/hptx-samples/tree/main/http_zorm)
```go
// Do not use proxy proxy mode,zorm to achieve transaction management, no modification of business code, zero intrusion to achieve distributed transactions
// tm.Implement(svc.ProxySvc)

// The distributed transaction must be started manually and must be invoked before the local transaction is started
ctx,_ = zorm.BindContextEnableGlobalTransaction(ctx)
// Distributed transaction sample code
_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

	// Get the XID of the current distributed transaction. Don't worry about how, if it is a distributed transaction environment, the value will be set automatically
	// xid := ctx.Value("XID").(string)

	// Pass the xid to the third party application
	// req.Header.Set("XID", xid)

	// If err is not returned nil, local and distributed transactions are rolled back
	return nil, err
})

// /---------- Third-party application -------// /

// Before third-party applications can start transactions,ctx needs to bind Xids, such as gin framework

// Accept the XID passed and bind it to the local ctx
// xid:=c.Request.Header.Get("XID")
// ctx is obtained
// ctx := c.Request.Context()
// ctx = context.WithValue(ctx,"XID",xid)

// The distributed transaction must be started manually and must be invoked before the local transaction is started
ctx,_ = zorm.BindContextEnableGlobalTransaction(ctx)
// ctx invokes the business transaction after binding the XID
_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

	// Business code......

	// If err is not returned nil, local and distributed transactions are rolled back
	return nil, err
})



// It is recommended that the following code be placed in a separate file
// ... // 

// ZormGlobalTransaction packaging hptx *tm.DefaultGlobalTransaction, zorm.IGlobalTransaction interface
type ZormGlobalTransaction struct {
	*tm.DefaultGlobalTransaction
}

// MyFuncGlobalTransaction zorm A function that ADAPTS a hptx globally distributed transaction
// important!!!! Need to configure the zorm.DataSourceConfig.FuncGlobalTransaction = MyFuncGlobalTransaction important!!!!!!
func MyFuncGlobalTransaction(ctx context.Context) (zorm.IGlobalTransaction, context.Context, context.Context, error) {
	// Obtain the hptx rootContext
	rootContext := gtxContext.NewRootContext(ctx)
	// Create a hptx transaction
	globalTx := tm.GetCurrentOrCreate(rootContext)
	// Use the zorm.IGlobalTransaction interface object to wrap distributed transactions and isolate hptx dependencies
	globalTransaction := &ZormGlobalTransaction{globalTx}

	return globalTransaction, ctx, rootContext, nil
}

// IGlobalTransaction managed global distributed transaction interface (zorm.IGlobalTransaction). seata and hptx currently implement the same code, only the reference implementation package is different

// BeginGTX Starts global distributed transactions
func (gtx *ZormGlobalTransaction) BeginGTX(ctx context.Context, globalRootContext context.Context) error {
	rootContext := globalRootContext.(*gtxContext.RootContext)
	return gtx.BeginWithTimeout(int32(6000), rootContext)
}

// CommitGTX Commit global distributed transactions
func (gtx *ZormGlobalTransaction) CommitGTX(ctx context.Context, globalRootContext context.Context) error {
	rootContext := globalRootContext.(*gtxContext.RootContext)
	return gtx.Commit(rootContext)
}

// RollbackGTX rolls back globally distributed transactions
func (gtx *ZormGlobalTransaction) RollbackGTX(ctx context.Context, globalRootContext context.Context) error {
	rootContext := globalRootContext.(*gtxContext.RootContext)
	// If it is the Participant role, change it to the Launcher role to allow branch transactions to submit global transactions.
	if gtx.Role != tm.Launcher {
		gtx.Role = tm.Launcher
	}
	return gtx.Rollback(rootContext)
}
// GetGTXID Gets the XID of the globally distributed transaction
func (gtx *ZormGlobalTransaction) GetGTXID(ctx context.Context, globalRootContext context.Context) (string.error) {
	rootContext := globalRootContext.(*gtxContext.RootContext)
	return rootContext.GetXID(), nil
}

// ... // 
```
###  dbpack distributed transactions 
```dbpack``` document: https://cectc.github.io/dbpack-doc/#/README deployment with a Mesh, the application integration is simple, just need to get xid, in a hint of SQL statements
```go
// Before starting dbpack transactions,ctx needs to bind sql hints, such as using the gin framework to obtain the xid passed by the header
xid := c.Request.Header.Get("xid")
// Generate sql hint content using xid, and then bind the hint to ctx
hint := fmt.Sprintf("/*+ XID('%s') */", xid)
// ctx is obtained
ctx := c.Request.Context()
// Bind the hint to ctx
ctx,_ = zorm.BindContextSQLHint(ctx, hint)

// After ctx binds the sql hint, the business transaction is invoked and ctx is transmitted to realize the propagation of the distributed transaction
_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

	// Business code......

	// If err is not returned nil, local and distributed transactions are rolled back
	return nil, err
})
```
