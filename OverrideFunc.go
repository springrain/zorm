package zorm

import (
	"context"
	"errors"
	"reflect"
)

/*
// OverrideFunc 重写ZORM的函数,当你使用这个函数时,你必须知道自己在做什么

//oldInsertFunc 默认的Insert实现
var oldInsertFunc func(ctx context.Context, entity zorm.IEntityStruct) (int, error)

//newInsertFunc 新的Insert实现
var newInsertFunc = func(ctx context.Context, entity zorm.IEntityStruct) (int, error) {
	fmt.Println("Insert前")
	i, err := oldInsertFunc(ctx, entity)
	fmt.Println("Insert后")
	return i, err
}

// 在init函数中注册覆盖老的函数
func init() {
	ok, oldFunc, err := zorm.OverrideFunc("Insert", newInsertFunc)
	if ok && err == nil {
		oldInsertFunc = oldFunc.(func(ctx context.Context, entity zorm.IEntityStruct) (int, error))
	}
}
*/
// OverrideFunc 重写ZORM的函数,用于风险监控,只要查看这个函数的调用,就知道哪些地方重写了函数,避免项目混乱.当你使用这个函数时,你必须知道自己在做什么
// funcName 是需要重写的方法命,funcObject是对应的函数. 返回值bool是否重写成功,interface{}是重写前的函数
// 一般是在init里调用重写
func OverrideFunc(funcName string, funcObject interface{}) (bool, interface{}, error) {
	if funcName == "" {
		return false, nil, errors.New("->OverrideFunc-->funcName不能为空")
	}

	// oldFunc 老的函数
	var oldFunc interface{} = nil
	switch funcName {
	case "Transaction":
		newFunc, ok := funcObject.(func(ctx context.Context, doTransaction func(ctx context.Context) (interface{}, error)) (interface{}, error))
		if ok {
			oldFunc = transaction
			transaction = newFunc
		}
	case "QueryRow":
		newFunc, ok := funcObject.(func(ctx context.Context, finder *Finder, entity interface{}) (bool, error))
		if ok {
			oldFunc = queryRow
			queryRow = newFunc
		}
	case "Query":
		newFunc, ok := funcObject.(func(ctx context.Context, finder *Finder, rowsSlicePtr interface{}, page *Page) error)
		if ok {
			oldFunc = query
			query = newFunc
		}

	case "QueryRowMap":
		newFunc, ok := funcObject.(func(ctx context.Context, finder *Finder) (map[string]interface{}, error))
		if ok {
			oldFunc = queryRowMap
			queryRowMap = newFunc
		}
	case "QueryMap":
		newFunc, ok := funcObject.(func(ctx context.Context, finder *Finder, page *Page) ([]map[string]interface{}, error))
		if ok {
			oldFunc = queryMap
			queryMap = newFunc
		}
	case "UpdateFinder":
		newFunc, ok := funcObject.(func(ctx context.Context, finder *Finder) (int, error))
		if ok {
			oldFunc = updateFinder
			updateFinder = newFunc
		}
	case "Insert":
		newFunc, ok := funcObject.(func(ctx context.Context, entity IEntityStruct) (int, error))
		if ok {
			oldFunc = insert
			insert = newFunc
		}
	case "InsertSlice":
		newFunc, ok := funcObject.(func(ctx context.Context, entityStructSlice []IEntityStruct) (int, error))
		if ok {
			oldFunc = insertSlice
			insertSlice = newFunc
		}
	case "Update":
		newFunc, ok := funcObject.(func(ctx context.Context, entity IEntityStruct) (int, error))
		if ok {
			oldFunc = update
			update = newFunc
		}
	case "UpdateNotZeroValue":
		newFunc, ok := funcObject.(func(ctx context.Context, entity IEntityStruct) (int, error))
		if ok {
			oldFunc = updateNotZeroValue
			updateNotZeroValue = newFunc
		}
	case "Delete":
		newFunc, ok := funcObject.(func(ctx context.Context, entity IEntityStruct) (int, error))
		if ok {
			oldFunc = delete
			delete = newFunc
		}

	case "InsertEntityMap":
		newFunc, ok := funcObject.(func(ctx context.Context, entity IEntityMap) (int, error))
		if ok {
			oldFunc = insertEntityMap
			insertEntityMap = newFunc
		}
	case "InsertEntityMapSlice":
		newFunc, ok := funcObject.(func(ctx context.Context, entity []IEntityMap) (int, error))
		if ok {
			oldFunc = insertEntityMapSlice
			insertEntityMapSlice = newFunc
		}
	case "UpdateEntityMap":
		newFunc, ok := funcObject.(func(ctx context.Context, entity IEntityMap) (int, error))
		if ok {
			oldFunc = updateEntityMap
			updateEntityMap = newFunc
		}
	case "reBuildSQL": //重建语句,用于占位符替换等
		newFunc, ok := funcObject.(func(ctx context.Context, config *DataSourceConfig, sqlstr *string, args *[]interface{}) (*string, *[]interface{}, error))
		if ok {
			oldFunc = reBuildSQL
			reBuildSQL = newFunc
		}
	case "reBuildUpdateSQL": //重建更新语句,处理特殊场景,例如 clickhouse的 UPDATE 和 DELETE 等
		newFunc, ok := funcObject.(func(ctx context.Context, config *DataSourceConfig, sqlstr *string) error)
		if ok {
			oldFunc = reBuildUpdateSQL
			reBuildUpdateSQL = newFunc
		}
	case "wrapPageSQL": //分页SQL
		newFunc, ok := funcObject.(func(ctx context.Context, config *DataSourceConfig, sqlstr *string, page *Page) error)
		if ok {
			oldFunc = wrapPageSQL
			wrapPageSQL = newFunc
		}

	case "wrapInsertSQL": //Insert IEntityStruct SQL
		newFunc, ok := funcObject.(func(ctx context.Context, config *DataSourceConfig, typeOf *reflect.Type, entity IEntityStruct, columns *[]reflect.StructField, values *[]interface{}) (*string, int, string, error))
		if ok {
			oldFunc = wrapInsertSQL
			wrapInsertSQL = newFunc
		}

	case "wrapAutoIncrementInsertSQL": //Insert IEntityStruct 主键自增值的SQL
		newFunc, ok := funcObject.(func(ctx context.Context, config *DataSourceConfig, pkColumnName string, sqlstr *string, values *[]interface{}) (*int64, *int64))
		if ok {
			oldFunc = wrapAutoIncrementInsertSQL
			wrapAutoIncrementInsertSQL = newFunc
		}

	case "wrapInsertSliceSQL": //批量插入 IEntityStruct 的SQL
		newFunc, ok := funcObject.(func(ctx context.Context, config *DataSourceConfig, typeOf *reflect.Type, entityStructSlice []IEntityStruct, columns *[]reflect.StructField, values *[]interface{}) (*string, int, error))
		if ok {
			oldFunc = wrapInsertSliceSQL
			wrapInsertSliceSQL = newFunc
		}
	case "wrapInsertEntityMapSQL": //插入 IEntityMap 的SQL
		newFunc, ok := funcObject.(func(ctx context.Context, config *DataSourceConfig, entity IEntityMap) (string, *[]interface{}, bool, error))
		if ok {
			oldFunc = wrapInsertEntityMapSQL
			wrapInsertEntityMapSQL = newFunc
		}

	case "wrapInsertEntityMapSliceSQL": //批量插入 IEntityMap 的SQL
		newFunc, ok := funcObject.(func(ctx context.Context, config *DataSourceConfig, entityMapSlice []IEntityMap) (*string, *[]interface{}, error))
		if ok {
			oldFunc = wrapInsertEntityMapSliceSQL
			wrapInsertEntityMapSliceSQL = newFunc
		}

	case "wrapDeleteSQL": //删除 IEntityStruct 的SQL
		newFunc, ok := funcObject.(func(ctx context.Context, entity IEntityStruct) (string, error))
		if ok {
			oldFunc = wrapDeleteSQL
			wrapDeleteSQL = newFunc
		}

	case "wrapUpdateSQL": //更新 IEntityStruct 的SQL
		newFunc, ok := funcObject.(func(ctx context.Context, typeOf *reflect.Type, entity IEntityStruct, columns *[]reflect.StructField, values *[]interface{}) (string, error))
		if ok {
			oldFunc = wrapUpdateSQL
			wrapUpdateSQL = newFunc
		}

	case "wrapUpdateEntityMapSQL": //更新 IEntityMap 的SQL
		newFunc, ok := funcObject.(func(ctx context.Context, entity IEntityMap) (*string, *[]interface{}, error))
		if ok {
			oldFunc = wrapUpdateEntityMapSQL
			wrapUpdateEntityMapSQL = newFunc
		}

	default:
		return false, oldFunc, errors.New("->OverrideFunc-->函数" + funcName + "暂不支持重写或不存在")
	}
	if oldFunc == nil {
		return false, oldFunc, errors.New("->OverrideFunc-->请检查传入的" + funcName + "函数实现,断言转换失败.")
	}
	return true, oldFunc, nil
}
