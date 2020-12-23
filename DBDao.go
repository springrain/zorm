// Package zorm 使用原生的sql语句,没有对sql语法做限制.语句使用Finder作为载体
// 占位符统一使用?,zorm会根据数据库类型,语句执行前会自动替换占位符,postgresql 把?替换成$1,$2...;mssql替换成@P1,@p2...;orace替换成:1,:2...
// zorm使用 ctx context.Context 参数实现事务传播,ctx从web层传递进来即可,例如gin的c.Request.Context()
// zorm的事务操作需要显示使用zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {})开启
package zorm

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

//FuncReadWriteStrategy 单个数据库的读写分离的策略,用于外部复写实现自定义的逻辑,rwType=0 read,rwType=1 write
//不能归属到BaseDao里,BindContextDBConnection已经是指定数据库的连接了,和这个函数会冲突.就作为单数据库读写分离的处理方式
var FuncReadWriteStrategy func(rwType int) *DBDao = getDefaultDao

type wrapContextStringKey string

//context WithValue的key,不能是基础类型,例如字符串,包装一下
const contextDBConnectionValueKey = wrapContextStringKey("contextDBConnectionValueKey")

//NewContextDBConnectionValueKey 创建context中存放DBConnection的key
//故意使用一个公开方法,返回私有类型wrapContextStringKey,多库时禁止自定义contextKey,只能调用这个方法,不能接收也不能改变
//例如:ctx = context.WithValue(ctx, zorm.NewContextDBConnectionValueKey(), dbConnection)
//func NewContextDBConnectionValueKey() wrapContextStringKey {
//	return contextDBConnectionValueKey
//}

//bug(springrain) 还缺少1对1的属性嵌套对象,sql别名查询,直接赋值的功能.

//不再处理日期零值,会干扰反射判断零值
//默认的零时时间1970-01-01 00:00:00 +0000 UTC,兼容数据库,避免0001-01-01 00:00:00 +0000 UTC的零值.数据库不让存值,加上1秒,跪了
//因为mysql 5.7后,The TIMESTAMP data type is used for values that contain both date and time parts. TIMESTAMP has a range of '1970-01-01 00:00:01' UTC to '2038-01-19 03:14:07' UTC.
//var defaultZeroTime = time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC)

//var defaultZeroTime = time.Now()

//注释如果是 . 句号结尾,IDE的提示就截止了,注释结尾不要用 . 结束

//DBDao 数据库操作基类,隔离原生操作数据库API入口,所有数据库操作必须通过DBDao进行
type DBDao struct {
	config     *DataSourceConfig
	dataSource *dataSource
}

var defaultDao *DBDao = nil

// NewDBDao 创建dbDao,一个数据库要只执行一次,业务自行控制
//第一个执行的数据库为 defaultDao,后续zorm.xxx方法,默认使用的就是defaultDao
func NewDBDao(config *DataSourceConfig) (*DBDao, error) {
	dataSource, err := newDataSource(config)

	if err != nil {
		err = fmt.Errorf("创建dataSource失败:%w", err)
		FuncLogError(err)
		return nil, err
	}

	if FuncReadWriteStrategy(1) == nil {
		defaultDao = &DBDao{config, dataSource}
		return defaultDao, nil
	}
	return &DBDao{config, dataSource}, nil
}

//获取默认的Dao,用于隔离读写的Dao
func getDefaultDao(rwType int) *DBDao {
	return defaultDao
}

// newDBConnection 获取一个dbConnection
//如果参数dbConnection为nil,使用默认的datasource进行获取dbConnection
//如果是多库,Dao手动调用newDBConnection(),获得dbConnection,WithValue绑定到子context
func (dbDao *DBDao) newDBConnection() (*dataBaseConnection, error) {
	if dbDao == nil || dbDao.dataSource == nil {
		return nil, errors.New("请不要自己创建dbDao,使用NewDBDao方法进行创建")
	}
	dbConnection := new(dataBaseConnection)
	dbConnection.db = dbDao.dataSource.DB
	dbConnection.dbType = dbDao.config.DBType
	dbConnection.driverName = dbDao.config.DriverName
	dbConnection.printSQL = dbDao.config.PrintSQL
	return dbConnection, nil
}

//BindContextDBConnection 多库的时候,通过dbDao创建DBConnection绑定到子context,返回的context就有了DBConnection
//parent 不能为空
func (dbDao *DBDao) BindContextDBConnection(parent context.Context) (context.Context, error) {
	if parent == nil {
		return nil, errors.New("context的parent不能为nil")
	}
	dbConnection, errDBConnection := dbDao.newDBConnection()
	if errDBConnection != nil {
		return parent, errDBConnection
	}
	ctx := context.WithValue(parent, contextDBConnectionValueKey, dbConnection)
	return ctx, nil
}

/*
Transaction 的示例代码
  //匿名函数return的error如果不为nil,事务就会回滚
  zorm.Transaction(ctx context.Context,func(ctx context.Context) (interface{}, error) {

	  //业务代码


	  //return的error如果不为nil,事务就会回滚
      return nil, nil
  })
*/
// 事务方法,隔离dbConnection相关的API.必须通过这个方法进行事务处理,统一事务方式
// 如果入参ctx中没有dbConnection,使用defaultDao开启事务并最后提交
// 如果入参ctx有dbConnection且没有事务,调用dbConnection.begin()开启事务并最后提交
// 如果入参ctx有dbConnection且有事务,只使用不提交,有开启方提交事务
// 但是如果遇到错误或者异常,虽然不是事务的开启方,也会回滚事务,让事务尽早回滚
// 在多库的场景,手动获取dbConnection,然后绑定到一个新的context,传入进来
// 不要去掉匿名函数的context参数,因为如果Transaction的context中没有dbConnection,会新建一个context并放入dbConnection,此时的context指针已经变化,不能直接使用Transaction的context参数
// bug(springrain)如果有大神修改了匿名函数内的参数名,例如改为ctx2,这样业务代码实际使用的是Transaction的context参数,如果为没有dbConnection,会抛异常,如果有dbConnection,实际就是一个对象.影响有限.也可以把匿名函数抽到外部
// return的error如果不为nil,事务就会回滚
func Transaction(ctx context.Context, doTransaction func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	//是否是dbConnection的开启方,如果是开启方,才可以提交事务
	txOpen := false
	//如果dbConnection不存在,则会用默认的datasource开启事务
	var checkerr error
	var dbConnection *dataBaseConnection
	ctx, dbConnection, checkerr = checkDBConnection(ctx, false, 1)
	if checkerr != nil {
		return nil, checkerr
	}
	if dbConnection == nil || dbConnection.tx == nil {
		beginerr := dbConnection.beginTx(ctx)
		if beginerr != nil {
			beginerr = fmt.Errorf("事务开启失败:%w ", beginerr)
			FuncLogError(beginerr)
			return nil, beginerr
		}
		//本方法开启的事务,由本方法提交
		txOpen = true
	}

	defer func() {
		if r := recover(); r != nil {
			//err = fmt.Errorf("事务开启失败:%w ", err)
			//记录异常日志
			//if _, ok := r.(runtime.Error); ok {
			//	panic(r)
			//}
			err, errOk := r.(error)
			if errOk {
				err = fmt.Errorf("recover异常:%w", err)
				FuncLogPanic(err)
			}
			//if !txOpen { //如果不是开启方,也应该回滚事务,虽然可能造成日志不准确,但是回滚要尽早
			//	return
			//}
			rberr := dbConnection.rollback()
			if rberr != nil {
				rberr = fmt.Errorf("recover内事务回滚失败:%w", rberr)
				FuncLogError(rberr)
			}

		}
	}()

	info, err := doTransaction(ctx)
	if err != nil {
		err = fmt.Errorf("事务执行失败:%w", err)
		FuncLogError(err)
		//不是开启方回滚事务,有可能造成日志记录不准确,但是回滚最重要了,尽早回滚
		rberr := dbConnection.rollback()
		if rberr != nil {
			rberr = fmt.Errorf("事务回滚失败:%w", rberr)
			FuncLogError(rberr)
		}
		return info, err
	}
	if txOpen { //如果是事务开启方,提交事务
		commitError := dbConnection.commit()
		if commitError != nil {
			commitError = fmt.Errorf("事务提交失败:%w", commitError)
			FuncLogError(commitError)
			return info, commitError
		}
	}

	return nil, nil
}

//Query 不要偷懒调用QuerySlice返回第一条,问题1.需要构建一个selice,问题2.调用方传递的对象其他值会被抛弃或者覆盖.
//根据Finder和封装为指定的entity类型,entity必须是*struct类型或者基础类型的指针.把查询的数据赋值给entity,所以要求指针类型
//context必须传入,不能为空
func Query(ctx context.Context, finder *Finder, entity interface{}) error {

	typeOf, checkerr := checkEntityKind(entity)
	if checkerr != nil {
		checkerr = fmt.Errorf("类型检查错误:%w", checkerr)
		FuncLogError(checkerr)
		return checkerr
	}
	//从contxt中获取数据库连接,可能为nil
	dbConnection, errFromContxt := getDBConnectionFromContext(ctx)
	if errFromContxt != nil {
		return errFromContxt
	}
	//自己构建的dbConnection
	if dbConnection != nil && dbConnection.db == nil {
		return errDBConnection
	}

	var dbType string = ""
	if dbConnection == nil { //dbConnection为nil,使用defaultDao
		dbType = FuncReadWriteStrategy(0).config.DBType
	} else {
		dbType = dbConnection.dbType
	}

	//获取到sql语句
	sqlstr, err := wrapQuerySQL(dbType, finder, nil)
	if err != nil {
		err = fmt.Errorf("获取查询SQL语句错误:%w", err)
		FuncLogError(err)
		return err
	}

	//检查dbConnection.有可能会创建dbConnection或者开启事务,所以要尽可能的接近执行时检查.
	var dbConnectionerr error
	ctx, dbConnection, dbConnectionerr = checkDBConnection(ctx, false, 0)
	if dbConnectionerr != nil {
		return dbConnectionerr
	}

	//根据语句和参数查询
	rows, e := dbConnection.queryContext(ctx, sqlstr, finder.values...)
	defer rows.Close()

	if e != nil {
		e = fmt.Errorf("查询数据库错误:%w", e)
		FuncLogError(e)
		return e
	}

	//typeOf := reflect.TypeOf(entity).Elem()

	//数据库返回的列名
	columns, cne := rows.Columns()
	if cne != nil {
		cne = fmt.Errorf("数据库返回列名错误:%w", cne)
		FuncLogError(cne)
		return cne
	}

	//如果是基础类型,就查询一个字段
	if allowBaseTypeMap[typeOf.Kind()] && len(columns) == 1 {
		i := 0
		//循环遍历结果集
		for rows.Next() {
			if i > 1 {
				return errors.New("查询出多条数据")
			}
			i++
			scanerr := rows.Scan(entity)
			if scanerr != nil {
				scanerr = fmt.Errorf("rows.Scan异常:%w", scanerr)
				FuncLogError(scanerr)
				return scanerr
			}
		}

		return nil
	}

	valueOf := reflect.ValueOf(entity).Elem()
	//获取到类型的字段缓存
	dbColumnFieldMap, dbe := getDBColumnFieldMap(typeOf)
	if dbe != nil {
		dbe = fmt.Errorf("获取字段缓存错误:%w", dbe)
		FuncLogError(dbe)
		return dbe
	}
	//声明载体数组,用于存放struct的属性指针
	values := make([]interface{}, len(columns))
	i := 0
	//循环遍历结果集
	for rows.Next() {

		if i > 1 {
			return errors.New("查询出多条数据")
		}
		i++
		//遍历数据库的列名
		for i, column := range columns {
			//从缓存中获取列名的field字段
			field, fok := dbColumnFieldMap[column]
			if !fok { //如果列名不存在,就初始化一个空值
				values[i] = new(interface{})
				continue
			}
			//获取struct的属性值的指针地址,字段不会重名,不使用FieldByIndex()函数
			value := valueOf.FieldByName(field.Name).Addr().Interface()
			//把指针地址放到数组
			values[i] = value
		}
		//scan赋值.是一个指针数组,已经根据struct的属性类型初始化了,sql驱动能感知到参数类型,所以可以直接赋值给struct的指针.这样struct的属性就有值了
		scanerr := rows.Scan(values...)
		if scanerr != nil {
			scanerr = fmt.Errorf("rows.Scan错误:%w", scanerr)
			FuncLogError(scanerr)
			return scanerr
		}

	}

	return nil
}

//QuerySlice 不要偷懒调用QueryMapList,需要处理sql驱动支持的sql.Nullxxx的数据类型,也挺麻烦的
//根据Finder和封装为指定的entity类型,entity必须是*[]struct类型,已经初始化好的数组,此方法只Append元素,这样调用方就不需要强制类型转换了
//context必须传入,不能为空
func QuerySlice(ctx context.Context, finder *Finder, rowsSlicePtr interface{}, page *Page) error {

	if rowsSlicePtr == nil { //如果为nil
		return errors.New("数组必须是*[]struct类型或者基础类型数组的指针")
	}

	pv1 := reflect.ValueOf(rowsSlicePtr)
	if pv1.Kind() != reflect.Ptr { //如果不是指针
		return errors.New("数组必须是*[]struct类型或者基础类型数组的指针")
	}

	//获取数组元素
	sliceValue := reflect.Indirect(pv1)

	//如果不是数组
	if sliceValue.Kind() != reflect.Slice {
		return errors.New("数组必须是*[]struct类型或者基础类型数组的指针")
	}
	//获取数组内的元素类型
	sliceElementType := sliceValue.Type().Elem()

	//如果不是struct
	if !(sliceElementType.Kind() == reflect.Struct || allowBaseTypeMap[sliceElementType.Kind()]) {
		return errors.New("数组必须是*[]struct类型或者基础类型数组的指针")
	}
	//从contxt中获取数据库连接,可能为nil
	dbConnection, errFromContxt := getDBConnectionFromContext(ctx)
	if errFromContxt != nil {
		return errFromContxt
	}
	//自己构建的dbConnection
	if dbConnection != nil && dbConnection.db == nil {
		return errDBConnection
	}

	var dbType string = ""
	if dbConnection == nil { //dbConnection为nil,使用defaultDao
		dbType = FuncReadWriteStrategy(0).config.DBType
	} else {
		dbType = dbConnection.dbType
	}

	sqlstr, err := wrapQuerySQL(dbType, finder, page)
	if err != nil {
		err = fmt.Errorf("获取查询SQL语句错误:%w", err)
		FuncLogError(err)
		return err
	}

	//检查dbConnection.有可能会创建dbConnection或者开启事务,所以要尽可能的接近执行时检查.
	var dbConnectionerr error
	ctx, dbConnection, dbConnectionerr = checkDBConnection(ctx, false, 0)
	if dbConnectionerr != nil {
		return dbConnectionerr
	}

	//根据语句和参数查询
	rows, e := dbConnection.queryContext(ctx, sqlstr, finder.values...)
	defer rows.Close()
	if e != nil {
		e = fmt.Errorf("查询rows异常:%w", e)
		FuncLogError(e)
		return e
	}
	//数据库返回的列名
	columns, cne := rows.Columns()
	if cne != nil {
		cne = fmt.Errorf("数据库返回列名错误:%w", cne)
		FuncLogError(cne)
		return cne
	}

	//如果是基础类型,就查询一个字段
	if allowBaseTypeMap[sliceElementType.Kind()] {

		//循环遍历结果集
		for rows.Next() {
			//初始化一个基本类型,new出来的是指针.
			pv := reflect.New(sliceElementType)
			//把数据库值赋给指针
			scanerr := rows.Scan(pv.Interface())
			if scanerr != nil {
				scanerr = fmt.Errorf("rows.Scan异常:%w", scanerr)
				FuncLogError(scanerr)
				return scanerr
			}
			//通过反射给slice添加元素.添加指针下的真实元素
			sliceValue.Set(reflect.Append(sliceValue, pv.Elem()))
		}

		//查询总条数
		if page != nil && finder.SelectTotalCount {
			count, counterr := selectCount(ctx, finder)
			if counterr != nil {
				counterr = fmt.Errorf("查询总条数错误:%w", counterr)
				FuncLogError(counterr)
				return counterr
			}
			page.setTotalCount(count)
		}
		return nil
	}

	//获取到类型的字段缓存
	dbColumnFieldMap, dbe := getDBColumnFieldMap(sliceElementType)
	if dbe != nil {
		dbe = fmt.Errorf("获取字段缓存错误:%w", dbe)
		FuncLogError(dbe)
		return dbe
	}
	//声明载体数组,用于存放struct的属性指针
	values := make([]interface{}, len(columns))
	//循环遍历结果集
	for rows.Next() {
		//deepCopy(a, entity)
		//反射初始化一个数组内的元素
		//new 出来的为什么是个指针啊????
		pv := reflect.New(sliceElementType).Elem()
		//遍历数据库的列名
		for i, column := range columns {
			//从缓存中获取列名的field字段
			field, fok := dbColumnFieldMap[column]
			if !fok { //如果列名不存在,就初始化一个空值
				values[i] = new(interface{})
				continue
			}
			//获取struct的属性值的指针地址,字段不会重名,不使用FieldByIndex()函数
			value := pv.FieldByName(field.Name).Addr().Interface()
			//把指针地址放到数组
			values[i] = value
		}
		/*
			// fix:converting NULL to int is unsupported
			// 当读取数据库的值为NULL时，由于基本类型不支持为NULL，通过反射将未知driver.Value改为NullBool,基本类型会自动强转为默认值
				newValues := make([]interface{}, 0, len(values))
				empty := sql.NullBool{}
				queryValue := reflect.Indirect(reflect.ValueOf(rows))
				queryValue = queryValue.FieldByName("lastcols")
				cnt := queryValue.Len()
				for i := 0; i < cnt; i++ {
					v := queryValue.Index(i)
					if v.IsValid() {
						if v.InterfaceData()[0] != 0 {
							newValues = append(newValues, values[i])
						} else {
							newValues = append(newValues, &empty)
						}
					}
				}
		*/
		//scan赋值.是一个指针数组,已经根据struct的属性类型初始化了,sql驱动能感知到参数类型,所以可以直接赋值给struct的指针.这样struct的属性就有值了
		scanerr := rows.Scan(values...)
		if scanerr != nil {
			scanerr = fmt.Errorf("rows.Scan异常:%w", scanerr)
			FuncLogError(scanerr)
			return scanerr
		}

		//values[i] = f.Addr().Interface()
		//通过反射给slice添加元素
		sliceValue.Set(reflect.Append(sliceValue, pv))
	}

	//查询总条数
	if page != nil && finder.SelectTotalCount {
		count, counterr := selectCount(ctx, finder)
		if counterr != nil {
			counterr = fmt.Errorf("查询总条数错误:%w", counterr)
			FuncLogError(counterr)
			return counterr
		}
		page.setTotalCount(count)
	}

	return nil

}

//QueryMap 根据Finder查询,封装Map
//context必须传入,不能为空
func QueryMap(ctx context.Context, finder *Finder) (map[string]interface{}, error) {

	if finder == nil {
		return nil, errors.New("QueryMap的finder参数不能为nil")
	}
	resultMapList, listerr := QueryMapSlice(ctx, finder, nil)
	if listerr != nil {
		listerr = fmt.Errorf("QueryMapList查询错误:%w", listerr)
		FuncLogError(listerr)
		return nil, listerr
	}
	if resultMapList == nil {
		return nil, nil
	}
	if len(resultMapList) > 1 {
		return resultMapList[0], errors.New("查询出多条数据")
	} else if len(resultMapList) == 0 { //数据库不存在值
		return nil, nil
	}
	return resultMapList[0], nil
}

//QueryMapSlice 根据Finder查询,封装Map数组
//根据数据库字段的类型,完成从[]byte到golang类型的映射,理论上其他查询方法都可以调用此方法,但是需要处理sql.Nullxxx等驱动支持的类型
//context必须传入,不能为空
func QueryMapSlice(ctx context.Context, finder *Finder, page *Page) ([]map[string]interface{}, error) {

	if finder == nil {
		return nil, errors.New("QueryMap的finder参数不能为nil")
	}
	//从contxt中获取数据库连接,可能为nil
	dbConnection, errFromContxt := getDBConnectionFromContext(ctx)
	if errFromContxt != nil {
		return nil, errFromContxt
	}
	//自己构建的dbConnection
	if dbConnection != nil && dbConnection.db == nil {
		return nil, errDBConnection
	}

	var dbType string = ""
	if dbConnection == nil { //dbConnection为nil,使用defaultDao
		dbType = FuncReadWriteStrategy(0).config.DBType
	} else {
		dbType = dbConnection.dbType
	}

	sqlstr, err := wrapQuerySQL(dbType, finder, page)
	if err != nil {
		err = fmt.Errorf("QueryMapList查询SQL语句错误:%w", err)
		FuncLogError(err)
		return nil, err
	}

	//检查dbConnection.有可能会创建dbConnection或者开启事务,所以要尽可能的接近执行时检查.
	var dbConnectionerr error
	ctx, dbConnection, dbConnectionerr = checkDBConnection(ctx, false, 0)
	if dbConnectionerr != nil {
		return nil, dbConnectionerr
	}

	//根据语句和参数查询
	rows, e := dbConnection.queryContext(ctx, sqlstr, finder.values...)
	defer rows.Close()
	if e != nil {
		e = fmt.Errorf("查询rows错误:%w", e)
		FuncLogError(e)
		return nil, e
	}

	//数据库返回的列类型
	//columns, cne := rows.Columns()
	//columnType.scanType返回的类型都是[]byte,使用columnType.databaseType挨个判断
	columnTypes, cne := rows.ColumnTypes()
	if cne != nil {
		cne = fmt.Errorf("数据库返回列名错误:%w", cne)
		FuncLogError(cne)
		return nil, cne
	}
	resultMapList := make([]map[string]interface{}, 0)
	//循环遍历结果集
	for rows.Next() {
		//接收数据库返回的数据,需要使用指针接收
		values := make([]interface{}, len(columnTypes))
		//使用指针类型接收字段值,需要使用interface{}包装一下
		result := make(map[string]interface{})
		//给数据赋值初始化变量
		for i := range values {
			values[i] = new(interface{})
		}
		//scan赋值
		scanerr := rows.Scan(values...)
		if scanerr != nil {
			scanerr = fmt.Errorf("rows.Scan异常:%w", scanerr)
			FuncLogError(scanerr)
			return nil, scanerr
		}
		//获取每一列的值
		for i, columnType := range columnTypes {

			//取到指针下的值,[]byte格式
			v := *(values[i].(*interface{}))
			//从[]byte转化成实际的类型值,例如string,int
			v = converValueColumnType(v, columnType)
			//赋值到Map
			result[columnType.Name()] = v

		}

		//添加Map到数组
		resultMapList = append(resultMapList, result)

	}

	//bug(springrain) 还缺少查询总条数的逻辑
	//查询总条数
	if page != nil && finder.SelectTotalCount {
		count, counterr := selectCount(ctx, finder)
		if counterr != nil {
			counterr = fmt.Errorf("查询总条数错误:%w", counterr)
			FuncLogError(counterr)
			return resultMapList, counterr
		}
		page.setTotalCount(count)
	}

	return resultMapList, nil
}

//UpdateFinder 更新Finder语句
//ctx不能为nil,参照使用zorm.Transaction方法传入ctx.也不要自己构建DBConnection
//affected影响的行数,如果异常或者驱动不支持,返回-1
func UpdateFinder(ctx context.Context, finder *Finder) (int, error) {
	affected := -1
	if finder == nil {
		return affected, errors.New("finder不能为空")
	}
	sqlstr, err := finder.GetSQL()
	if err != nil {
		err = fmt.Errorf("finder.GetSQL()错误:%w", err)
		FuncLogError(err)
		return affected, err
	}

	//从contxt中获取数据库连接,可能为nil
	dbConnection, errFromContxt := getDBConnectionFromContext(ctx)
	if errFromContxt != nil {
		return affected, errFromContxt
	}

	//自己构建的dbConnection
	if dbConnection != nil && dbConnection.db == nil {
		return affected, errDBConnection
	}

	var dbType string = ""
	if dbConnection == nil { //dbConnection为nil,使用defaultDao
		dbType = FuncReadWriteStrategy(1).config.DBType
	} else {
		dbType = dbConnection.dbType
	}

	sqlstr, err = wrapSQL(dbType, sqlstr)
	if err != nil {
		err = fmt.Errorf("UpdateFinder-->wrapSQL获取SQL语句错误:%w", err)
		FuncLogError(err)
		return affected, err
	}

	//必须要有dbConnection和事务.有可能会创建dbConnection放入ctx或者开启事务,所以要尽可能的接近执行时检查
	var dbConnectionerr error
	ctx, dbConnection, dbConnectionerr = checkDBConnection(ctx, true, 1)
	if dbConnectionerr != nil {
		return affected, dbConnectionerr
	}

	res, errexec := dbConnection.execContext(ctx, sqlstr, finder.values...)

	if errexec != nil {
		errexec = fmt.Errorf("执行更新错误:%w", errexec)
		FuncLogError(errexec)
		return affected, errexec
	}
	//影响的行数
	rowsAffected, errAffected := res.RowsAffected()
	if errAffected == nil {
		affected, _ = typeConvertInt64toInt(rowsAffected)
	}

	return affected, nil
}

//Insert 保存Struct对象,必须是IEntityStruct类型
//ctx不能为nil,参照使用zorm.Transaction方法传入ctx.也不要自己构建DBConnection
//affected影响的行数,如果异常或者驱动不支持,返回-1
func Insert(ctx context.Context, entity IEntityStruct) (int, error) {
	affected := -1
	if entity == nil {
		return affected, errors.New("对象不能为空")
	}
	typeOf, columns, values, columnAndValueErr := columnAndValue(entity)
	if columnAndValueErr != nil {
		columnAndValueErr = fmt.Errorf("Insert-->columnAndValue获取实体类的列和值异常:%w", columnAndValueErr)
		FuncLogError(columnAndValueErr)
		return affected, columnAndValueErr
	}
	if len(columns) < 1 {
		return affected, errors.New("没有tag信息,请检查struct中 column 的tag")
	}
	//从contxt中获取数据库连接,可能为nil
	dbConnection, errFromContxt := getDBConnectionFromContext(ctx)
	if errFromContxt != nil {
		return affected, errFromContxt
	}
	//自己构建的dbConnection
	if dbConnection != nil && dbConnection.db == nil {
		return affected, errDBConnection
	}

	var dbType string = ""
	if dbConnection == nil { //dbConnection为nil,使用defaultDao
		dbType = FuncReadWriteStrategy(1).config.DBType
	} else {
		dbType = dbConnection.dbType
	}

	//SQL语句
	sqlstr, autoIncrement, err := wrapInsertStructSQL(dbType, typeOf, entity, &columns, &values)
	if err != nil {
		err = fmt.Errorf("Insert-->wrapInsertStructSQL获取保存语句错误:%w", err)
		FuncLogError(err)
		return affected, err
	}

	//必须要有dbConnection和事务.有可能会创建dbConnection放入ctx或者开启事务,所以要尽可能的接近执行时检查
	var dbConnectionerr error
	ctx, dbConnection, dbConnectionerr = checkDBConnection(ctx, true, 1)
	if dbConnectionerr != nil {
		return affected, dbConnectionerr
	}

	//流弊的...,把数组展开变成多个参数的形式
	res, errexec := dbConnection.execContext(ctx, sqlstr, values...)

	if errexec != nil {
		errexec = fmt.Errorf("Insert执行保存错误:%w", errexec)
		FuncLogError(errexec)
		return affected, errexec
	}
	//影响的行数
	rowsAffected, errAffected := res.RowsAffected()
	if errAffected == nil {
		affected, _ = typeConvertInt64toInt(rowsAffected)
	}
	//如果是自增主键
	if autoIncrement {
		//需要数据库支持,获取自增主键
		autoIncrementIDInt64, e := res.LastInsertId()
		if e != nil { //数据库不支持自增主键,不再赋值给struct属性
			e = fmt.Errorf("数据库不支持自增主键,不再赋值给struct属性:%w", e)
			FuncLogError(e)
			return affected, nil
		}
		pkName := entity.GetPKColumnName()
		//int64 转 int
		autoIncrementIDInt, _ := typeConvertInt64toInt(autoIncrementIDInt64)
		//设置自增主键的值
		seterr := setFieldValueByColumnName(entity, pkName, autoIncrementIDInt)
		if seterr != nil {
			seterr = fmt.Errorf("反射赋值数据库返回的自增主键错误:%w", seterr)
			FuncLogError(seterr)
			return affected, seterr
		}
	}

	return affected, nil

}

//InsertSlice 批量保存Struct Slice 数组对象,必须是[]IEntityStruct类型,golang目前没有泛型,使用IEntityStruct接口,兼容Struct实体类
//如果是自增主键,无法对Struct对象里的主键属性赋值
//ctx不能为nil,参照使用zorm.Transaction方法传入ctx.也不要自己构建DBConnection
//affected影响的行数,如果异常或者驱动不支持,返回-1
func InsertSlice(ctx context.Context, entityStructSlice []IEntityStruct) (int, error) {
	affected := -1
	if entityStructSlice == nil || len(entityStructSlice) < 1 {
		return affected, errors.New("对象数组不能为空")
	}
	//第一个对象,获取第一个Struct对象,用于获取数据库字段,也获取了值
	entity := entityStructSlice[0]
	typeOf, columns, values, columnAndValueErr := columnAndValue(entity)
	if columnAndValueErr != nil {
		columnAndValueErr = fmt.Errorf("InsertSlice-->columnAndValue获取实体类的列和值异常:%w", columnAndValueErr)
		FuncLogError(columnAndValueErr)
		return affected, columnAndValueErr
	}
	if len(columns) < 1 {
		return affected, errors.New("没有tag信息,请检查struct中 column 的tag")
	}
	//从contxt中获取数据库连接,可能为nil
	dbConnection, errFromContxt := getDBConnectionFromContext(ctx)
	if errFromContxt != nil {
		return affected, errFromContxt
	}
	//自己构建的dbConnection
	if dbConnection != nil && dbConnection.db == nil {
		return affected, errDBConnection
	}

	var dbType string = ""
	if dbConnection == nil { //dbConnection为nil,使用defaultDao
		dbType = FuncReadWriteStrategy(1).config.DBType
	} else {
		dbType = dbConnection.dbType
	}

	//SQL语句
	sqlstr, _, err := wrapInsertSliceStructSQL(dbType, typeOf, entityStructSlice, &columns, &values)
	if err != nil {
		err = fmt.Errorf("InsertSlice-->wrapInsertSliceStructSQL获取保存语句错误:%w", err)
		FuncLogError(err)
		return affected, err
	}

	//必须要有dbConnection和事务.有可能会创建dbConnection放入ctx或者开启事务,所以要尽可能的接近执行时检查
	var dbConnectionerr error
	ctx, dbConnection, dbConnectionerr = checkDBConnection(ctx, true, 1)
	if dbConnectionerr != nil {
		return affected, dbConnectionerr
	}

	//流弊的...,把数组展开变成多个参数的形式
	res, errexec := dbConnection.execContext(ctx, sqlstr, values...)

	if errexec != nil {
		errexec = fmt.Errorf("InsertSlice执行保存错误:%w", errexec)
		FuncLogError(errexec)
		return affected, errexec
	}
	//影响的行数
	rowsAffected, errAffected := res.RowsAffected()
	if errAffected == nil {
		affected, _ = typeConvertInt64toInt(rowsAffected)
	}
	return affected, nil

}

//Update 更新struct所有属性,必须是*IEntityStruct类型
//ctx不能为nil,参照使用zorm.Transaction方法传入ctx.也不要自己构建DBConnection
func Update(ctx context.Context, entity IEntityStruct) (int, error) {
	affected, err := updateStructFunc(ctx, entity, false)
	if err != nil {
		err = fmt.Errorf("Update-->updateStructFunc更新错误:%w", err)
		return affected, err
	}
	return affected, nil
}

//UpdateNotZeroValue 更新struct不为默认零值的属性,必须是*IEntityStruct类型,主键必须有值
//ctx不能为nil,参照使用zorm.Transaction方法传入ctx.也不要自己构建DBConnection
func UpdateNotZeroValue(ctx context.Context, entity IEntityStruct) (int, error) {
	affected, err := updateStructFunc(ctx, entity, true)
	if err != nil {
		err = fmt.Errorf("UpdateNotZeroValue-->updateStructFunc更新错误:%w", err)
		return affected, err
	}
	return affected, nil
}

//Delete 根据主键删除一个对象.必须是*IEntityStruct类型
//ctx不能为nil,参照使用zorm.Transaction方法传入ctx.也不要自己构建DBConnection
//affected影响的行数,如果异常或者驱动不支持,返回-1
func Delete(ctx context.Context, entity IEntityStruct) (int, error) {
	affected := -1
	typeOf, checkerr := checkEntityKind(entity)
	if checkerr != nil {
		return affected, checkerr
	}

	pkName, pkNameErr := entityPKFieldName(entity, typeOf)

	if pkNameErr != nil {
		pkNameErr = fmt.Errorf("Delete-->entityPKFieldName获取主键名称错误:%w", pkNameErr)
		FuncLogError(pkNameErr)
		return affected, pkNameErr
	}

	value, e := structFieldValue(entity, pkName)
	if e != nil {
		e = fmt.Errorf("Delete-->structFieldValue获取主键值错误:%w", e)
		FuncLogError(e)
		return affected, e
	}
	//从contxt中获取数据库连接,可能为nil
	dbConnection, errFromContxt := getDBConnectionFromContext(ctx)
	if errFromContxt != nil {
		return affected, errFromContxt
	}
	//自己构建的dbConnection
	if dbConnection != nil && dbConnection.db == nil {
		return affected, errDBConnection
	}

	var dbType string = ""
	if dbConnection == nil { //dbConnection为nil,使用defaultDao
		dbType = FuncReadWriteStrategy(1).config.DBType
	} else {
		dbType = dbConnection.dbType
	}

	//SQL语句
	sqlstr, err := wrapDeleteStructSQL(dbType, entity)
	if err != nil {
		err = fmt.Errorf("Delete-->wrapDeleteStructSQL获取SQL语句错误:%w", err)
		FuncLogError(err)
		return affected, err
	}

	//必须要有dbConnection和事务.有可能会创建dbConnection放入ctx或者开启事务,所以要尽可能的接近执行时检查
	var dbConnectionerr error
	ctx, dbConnection, dbConnectionerr = checkDBConnection(ctx, true, 1)
	if dbConnectionerr != nil {
		return affected, dbConnectionerr
	}

	res, errexec := dbConnection.execContext(ctx, sqlstr, value)

	if errexec != nil {
		errexec = fmt.Errorf("Delete执行删除错误:%w", errexec)
		FuncLogError(errexec)
		return affected, errexec
	}

	//影响的行数
	rowsAffected, errAffected := res.RowsAffected()
	if errAffected == nil {
		affected, _ = typeConvertInt64toInt(rowsAffected)
	}

	return affected, nil

}

//InsertEntityMap 保存*IEntityMap对象.使用Map保存数据,用于不方便使用struct的场景,如果主键是自增或者序列,不要entityMap.Set主键的值
//ctx不能为nil,参照使用zorm.Transaction方法传入ctx.也不要自己构建DBConnection
//affected影响的行数,如果异常或者驱动不支持,返回-1
func InsertEntityMap(ctx context.Context, entity IEntityMap) (int, error) {
	affected := -1
	//检查是否是指针对象
	_, checkerr := checkEntityKind(entity)
	if checkerr != nil {
		return affected, checkerr
	}

	//从contxt中获取数据库连接,可能为nil
	dbConnection, errFromContxt := getDBConnectionFromContext(ctx)
	if errFromContxt != nil {
		return affected, errFromContxt
	}

	//自己构建的dbConnection
	if dbConnection != nil && dbConnection.db == nil {
		return affected, errDBConnection
	}

	var dbType string = ""
	if dbConnection == nil { //dbConnection为nil,使用defaultDao
		dbType = FuncReadWriteStrategy(1).config.DBType
	} else {
		dbType = dbConnection.dbType
	}

	//SQL语句
	sqlstr, values, autoIncrement, err := wrapSaveMapSQL(dbType, entity)
	if err != nil {
		err = fmt.Errorf("SaveMap-->wrapSaveMapSQL获取SQL语句错误:%w", err)
		FuncLogError(err)
		return affected, err
	}

	//必须要有dbConnection和事务.有可能会创建dbConnection放入ctx或者开启事务,所以要尽可能的接近执行时检查
	var dbConnectionerr error
	ctx, dbConnection, dbConnectionerr = checkDBConnection(ctx, true, 1)
	if dbConnectionerr != nil {
		return affected, dbConnectionerr
	}

	//流弊的...,把数组展开变成多个参数的形式
	res, errexec := dbConnection.execContext(ctx, sqlstr, values...)
	if errexec != nil {
		errexec = fmt.Errorf("SaveMap执行保存错误:%w", errexec)
		FuncLogError(errexec)
		return affected, errexec
	}

	//影响的行数
	rowsAffected, errAffected := res.RowsAffected()
	if errAffected == nil {
		affected, _ = typeConvertInt64toInt(rowsAffected)
	}

	//如果是自增主键
	if autoIncrement {
		//需要数据库支持,获取自增主键
		autoIncrementIDInt64, e := res.LastInsertId()
		if e != nil { //数据库不支持自增主键,不再赋值给struct属性
			e = fmt.Errorf("数据库不支持自增主键,不再赋值给IEntityMap:%w", e)
			FuncLogError(e)
			return affected, nil
		}
		//int64 转 int
		strInt64 := strconv.FormatInt(autoIncrementIDInt64, 10)
		autoIncrementIDInt, _ := strconv.Atoi(strInt64)
		//设置自增主键的值
		entity.Set(entity.GetPKColumnName(), autoIncrementIDInt)
	}

	return affected, nil

}

//UpdateEntityMap 更新*IEntityMap对象.用于不方便使用struct的场景,主键必须有值
//ctx不能为nil,参照使用zorm.Transaction方法传入ctx.也不要自己构建DBConnection
//affected影响的行数,如果异常或者驱动不支持,返回-1
func UpdateEntityMap(ctx context.Context, entity IEntityMap) (int, error) {
	affected := -1
	//检查是否是指针对象
	_, checkerr := checkEntityKind(entity)
	if checkerr != nil {
		return affected, checkerr
	}
	//从contxt中获取数据库连接,可能为nil
	dbConnection, errFromContxt := getDBConnectionFromContext(ctx)
	if errFromContxt != nil {
		return affected, errFromContxt
	}
	//自己构建的dbConnection
	if dbConnection != nil && dbConnection.db == nil {
		return affected, errDBConnection
	}

	var dbType string = ""
	if dbConnection == nil { //dbConnection为nil,使用defaultDao
		dbType = FuncReadWriteStrategy(1).config.DBType
	} else {
		dbType = dbConnection.dbType
	}

	//SQL语句
	sqlstr, values, err := wrapUpdateMapSQL(dbType, entity)
	if err != nil {
		err = fmt.Errorf("UpdateMap-->wrapUpdateMapSQL获取SQL语句错误:%w", err)
		FuncLogError(err)
		return affected, err
	}

	//必须要有dbConnection和事务.有可能会创建dbConnection放入ctx或者开启事务,所以要尽可能的接近执行时检查
	var dbConnectionerr error
	ctx, dbConnection, dbConnectionerr = checkDBConnection(ctx, true, 1)
	if dbConnectionerr != nil {
		return affected, dbConnectionerr
	}

	//流弊的...,把数组展开变成多个参数的形式
	res, errexec := dbConnection.execContext(ctx, sqlstr, values...)

	if errexec != nil {
		errexec = fmt.Errorf("UpdateMap执行更新错误:%w", errexec)
		FuncLogError(errexec)
		return affected, errexec
	}
	//影响的行数
	rowsAffected, errAffected := res.RowsAffected()
	if errAffected == nil {
		affected, _ = typeConvertInt64toInt(rowsAffected)
	}
	return affected, nil

}

//更新对象
//ctx不能为nil,参照使用zorm.Transaction方法传入ctx.也不要自己构建DBConnection
//affected影响的行数,如果异常或者驱动不支持,返回-1
func updateStructFunc(ctx context.Context, entity IEntityStruct, onlyUpdateNotZero bool) (int, error) {
	affected := -1
	if entity == nil {
		return affected, errors.New("对象不能为空")
	}
	//从contxt中获取数据库连接,可能为nil
	dbConnection, errFromContxt := getDBConnectionFromContext(ctx)
	if errFromContxt != nil {
		return affected, errFromContxt
	}
	//自己构建的dbConnection
	if dbConnection != nil && dbConnection.db == nil {
		return affected, errDBConnection
	}

	var dbType string = ""
	if dbConnection == nil { //dbConnection为nil,使用defaultDao
		dbType = FuncReadWriteStrategy(1).config.DBType
	} else {
		dbType = dbConnection.dbType
	}

	typeOf, columns, values, columnAndValueErr := columnAndValue(entity)
	if columnAndValueErr != nil {
		return affected, columnAndValueErr
	}

	//SQL语句
	sqlstr, err := wrapUpdateStructSQL(dbType, typeOf, entity, &columns, &values, onlyUpdateNotZero)
	if err != nil {
		return affected, err
	}

	//必须要有dbConnection和事务.有可能会创建dbConnection放入ctx或者开启事务,所以要尽可能的接近执行时检查
	var dbConnectionerr error
	ctx, dbConnection, dbConnectionerr = checkDBConnection(ctx, true, 1)
	if dbConnectionerr != nil {
		return affected, dbConnectionerr
	}

	res, errexec := dbConnection.execContext(ctx, sqlstr, values...)

	if errexec != nil {
		return affected, errexec
	}

	//影响的行数
	rowsAffected, errAffected := res.RowsAffected()
	if errAffected == nil {
		affected, _ = typeConvertInt64toInt(rowsAffected)
	}

	return affected, nil

}

//selectCount 根据finder查询总条数
//context必须传入,不能为空
func selectCount(ctx context.Context, finder *Finder) (int, error) {

	if finder == nil {
		return -1, errors.New("参数为nil")
	}
	//自定义的查询总条数Finder,主要是为了在group by等复杂情况下,为了性能,手动编写总条数语句
	if finder.CountFinder != nil {
		count := -1
		err := Query(ctx, finder.CountFinder, &count)
		if err != nil {
			return -1, err
		}
		return count, nil
	}

	countsql, counterr := finder.GetSQL()
	if counterr != nil {
		return -1, counterr
	}

	//查询order by 的位置
	locOrderBy := findOrderByIndex(countsql)
	if len(locOrderBy) > 0 { //如果存在order by
		countsql = countsql[:locOrderBy[0]]
	}
	s := strings.ToLower(countsql)
	gbi := -1
	locGroupBy := findGroupByIndex(countsql)
	if len(locGroupBy) > 0 {
		gbi = locGroupBy[0]
	}
	//特殊关键字,包装SQL
	if strings.Index(s, " distinct ") > -1 || strings.Index(s, " union ") > -1 || gbi > -1 {
		countsql = "SELECT COUNT(*)  frame_row_count FROM (" + countsql + ") temp_frame_noob_table_name WHERE 1=1 "
	} else {
		locFrom := findFromIndex(countsql)
		//没有找到FROM关键字,认为是异常语句
		if len(locFrom) < 0 {
			return -1, errors.New("没有FROM关键字,语句错误")
		}
		countsql = "SELECT COUNT(*) " + countsql[locFrom[0]:]
	}

	countFinder := NewFinder()
	countFinder.Append(countsql)
	countFinder.values = finder.values

	count := -1
	cerr := Query(ctx, countFinder, &count)
	if cerr != nil {
		return -1, cerr
	}
	return count, nil

}

//getDBConnectionFromContext 从Conext中获取数据库连接
func getDBConnectionFromContext(ctx context.Context) (*dataBaseConnection, error) {
	if ctx == nil {
		return nil, errors.New("context不能为空")
	}
	//获取数据库连接
	value := ctx.Value(contextDBConnectionValueKey)
	if value == nil {
		return nil, nil
	}
	dbConnection, isdb := value.(*dataBaseConnection)
	if !isdb { //不是数据库连接
		return nil, errors.New("context传递了错误的*DBConnection类型值")
	}
	return dbConnection, nil

}

//变量名建议errFoo这样的驼峰
var errDBConnection = errors.New("更新操作需要使用zorm.Transaction开启事务.  读取操作如果ctx没有dbConnection,使用FuncReadWriteStrategy(rwType).newDBConnection(),如果dbConnection有事务,就使用事务查询")

//检查dbConnection.有可能会创建dbConnection或者开启事务,所以要尽可能的接近执行时检查.
//context必须传入,不能为空.rwType=0 read,rwType=1 write
func checkDBConnection(ctx context.Context, hastx bool, rwType int) (context.Context, *dataBaseConnection, error) {

	dbConnection, errFromContext := getDBConnectionFromContext(ctx)
	if errFromContext != nil {
		return ctx, nil, errFromContext
	}

	if dbConnection == nil { //dbConnection为空

		if hastx { //如果要求有事务,事务需要手动zorm.Transaction显示开启.如果自动开启,就会为了偷懒,每个操作都自动开启,事务就失去意义了
			return ctx, nil, errDBConnection
		}

		//如果要求没有事务,实例化一个默认的dbConnection
		var errGetDBConnection error
		dbConnection, errGetDBConnection = FuncReadWriteStrategy(rwType).newDBConnection()
		if errGetDBConnection != nil {
			return ctx, nil, errGetDBConnection
		}
		//把dbConnection放入context
		ctx = context.WithValue(ctx, contextDBConnectionValueKey, dbConnection)

	} else { //如果dbConnection存在

		if dbConnection.db == nil { //禁止外部构建
			return ctx, dbConnection, errDBConnection
		}
		tx := dbConnection.tx
		if tx == nil && hastx { //如果要求有事务,事务需要手动zorm.Transaction显示开启.如果自动开启,就会为了偷懒,每个操作都自动开启,事务就失去意义了
			return ctx, dbConnection, errDBConnection
		}
	}

	return ctx, dbConnection, nil

}
