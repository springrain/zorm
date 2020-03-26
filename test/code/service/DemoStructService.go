package permservice

import (
	"context"
	"errors"
	"fmt"
	"gitee.com/chunanyong/logger"
	"gitee.com/chunanyong/zorm"
	permstruct "gitee.com/chunanyong/zorm/test/code/struct"
)

//SaveDemoStruct 保存栗子
//如果入参ctx中没有dbConnection,使用defaultDao开启事务并最后提交
//如果入参ctx有dbConnection且没有事务,调用dbConnection.begin()开启事务并最后提交
//如果入参ctx有dbConnection且有事务,只使用不提交,有开启方提交事务
//但是如果遇到错误或者异常,虽然不是事务的开启方,也会回滚事务,让事务尽早回滚
func SaveDemoStruct(ctx context.Context, demoStruct *permstruct.DemoStruct) error {

    // demoStruct对象指针不能为空
	if  demoStruct == nil {
		return errors.New("demoStruct对象指针不能为空")
	}
    //匿名函数return的error如果不为nil,事务就会回滚
	_, errSaveDemoStruct := zorm.Transaction(ctx,func(ctx context.Context) (interface{}, error) {

		//事务下的业务代码开始

        //赋值主键Id
		if len(demoStruct.Id) < 1 {
			demoStruct.Id = zorm.FuncGenerateStringID()
		}


		errSaveDemoStruct := zorm.SaveStruct(ctx, demoStruct)


		if errSaveDemoStruct != nil {
			return nil, errSaveDemoStruct
		}

		return nil, nil
		//事务下的业务代码结束

	})

	//记录错误
	if errSaveDemoStruct != nil {
		errSaveDemoStruct := fmt.Errorf("permservice.SaveDemoStruct错误:%w", errSaveDemoStruct)
		logger.Error(errSaveDemoStruct)
		return errSaveDemoStruct
	}

	return nil
}

//UpdateDemoStruct 更新栗子
//如果入参ctx中没有dbConnection,使用defaultDao开启事务并最后提交
//如果入参ctx有dbConnection且没有事务,调用dbConnection.begin()开启事务并最后提交
//如果入参ctx有dbConnection且有事务,只使用不提交,有开启方提交事务
//但是如果遇到错误或者异常,虽然不是事务的开启方,也会回滚事务,让事务尽早回滚
func UpdateDemoStruct(ctx context.Context, demoStruct *permstruct.DemoStruct) error {


	// demoStruct对象指针或主键Id不能为空
	if  demoStruct == nil || len(demoStruct.Id) < 1 {
		return errors.New("demoStruct对象指针或主键Id不能为空")
	}
	
    //匿名函数return的error如果不为nil,事务就会回滚
	_, errUpdateDemoStruct := zorm.Transaction(ctx,func(ctx context.Context) (interface{}, error) {

		//事务下的业务代码开始
		errUpdateDemoStruct := zorm.UpdateStruct(ctx, demoStruct)


		if errUpdateDemoStruct != nil {
			return nil, errUpdateDemoStruct
		}

		return nil, nil
		//事务下的业务代码结束

	})

	//记录错误
	if errUpdateDemoStruct != nil {
		errUpdateDemoStruct := fmt.Errorf("permservice.UpdateDemoStruct错误:%w", errUpdateDemoStruct)
		logger.Error(errUpdateDemoStruct)
		return errUpdateDemoStruct
	}

	return nil
}

//DeleteDemoStructById 根据Id删除栗子
//如果入参ctx中没有dbConnection,使用defaultDao开启事务并最后提交
//如果入参ctx有dbConnection且没有事务,调用dbConnection.begin()开启事务并最后提交
//如果入参ctx有dbConnection且有事务,只使用不提交,有开启方提交事务
//但是如果遇到错误或者异常,虽然不是事务的开启方,也会回滚事务,让事务尽早回滚
func DeleteDemoStructById(ctx context.Context, id string) error {
	
	
	//id不能为空
	if len(id) < 1 {
		return errors.New("id不能为空")
	}
	
    //匿名函数return的error如果不为nil,事务就会回滚
	_, errDeleteDemoStruct := zorm.Transaction(ctx,func(ctx context.Context) (interface{}, error) {

		//事务下的业务代码开始
		finder := zorm.NewDeleteFinder(permstruct.DemoStructTableName).Append(" WHERE id=?", id)
		errDeleteDemoStruct := zorm.UpdateFinder(ctx, finder)


		if errDeleteDemoStruct != nil {
			return nil, errDeleteDemoStruct
		}

		return nil, nil
		//事务下的业务代码结束

	})

    //记录错误
	if errDeleteDemoStruct != nil {
		errDeleteDemoStruct := fmt.Errorf("permservice.DeleteDemoStruct错误:%w", errDeleteDemoStruct)
		logger.Error(errDeleteDemoStruct)
		return errDeleteDemoStruct
	}

	return nil
}

//FindDemoStructById 根据Id查询栗子信息
//ctx中如果没有dbConnection,则会使用默认的datasource进行无事务查询
func FindDemoStructById(ctx context.Context, id string) (*permstruct.DemoStruct, error) {
	//id不能为空
	if len(id) < 1 {
		return nil, errors.New("id不能为空")
	}

	//根据Id查询
	finder := zorm.NewSelectFinder(permstruct.DemoStructTableName).Append(" WHERE id=?", id)
	demoStruct := permstruct.DemoStruct{}
	errFindDemoStructById := zorm.QueryStruct(ctx, finder, &demoStruct)

	//记录错误
	if errFindDemoStructById != nil {
		errFindDemoStructById := fmt.Errorf("permservice.FindDemoStructById错误:%w", errFindDemoStructById)
		logger.Error(errFindDemoStructById)
		return nil, errFindDemoStructById
	}

	return &demoStruct, nil

}

//FindDemoStructList 根据Finder查询栗子列表
//ctx中如果没有dbConnection,则会使用默认的datasource进行无事务查询
func FindDemoStructList(ctx context.Context, finder *zorm.Finder, page *zorm.Page) ([]permstruct.DemoStruct, error) {
	
	//finder不能为空
	if finder == nil {
		return nil, errors.New("finder不能为空")
	}

	demoStructList := make([]permstruct.DemoStruct, 0)
	errFindDemoStructList := zorm.QueryStructList(ctx, finder, &demoStructList, page)

	//记录错误
	if errFindDemoStructList != nil {
		errFindDemoStructList := fmt.Errorf("permservice.FindDemoStructList错误:%w", errFindDemoStructList)
		logger.Error(errFindDemoStructList)
		return nil, errFindDemoStructList
	}

	return demoStructList, nil
}
