package zorm

import "context"

// ISeataGlobalTransaction seata-golang的包装接口,隔离seata-golang的依赖
// 声明一个struct,实现这个接口,并配置实现 FuncSeataGlobalTransaction 函数
/**
// ZormSeataGlobalTransaction 包装seata的*tm.DefaultGlobalTransaction,实现zorm.ISeataGlobalTransaction接口
type ZormSeataGlobalTransaction struct {
	*tm.DefaultGlobalTransaction
}

// MyFuncSeataGlobalTransaction zorm适配seata分布式事务的函数,配置zorm.DataSourceConfig.FuncSeataGlobalTransaction=MyFuncSeataGlobalTransaction
func MyFuncSeataGlobalTransaction(ctx context.Context) (zorm.ISeataGlobalTransaction, context.Context, error) {
	//获取seata的rootContext
	rootContext := seataContext.NewRootContext(ctx)
	//创建seata事务
	seataTx := tm.GetCurrentOrCreate(rootContext)
	//使用zorm.ISeataGlobalTransaction接口对象包装seata事务,隔离seata-golang依赖
	seataGlobalTransaction := ZormSeataGlobalTransaction{seataTx}

	return seataGlobalTransaction, rootContext, nil
}

//实现zorm.ISeataGlobalTransaction接口
func (gtx ZormSeataGlobalTransaction) SeataBegin(ctx context.Context) error {
	rootContext := ctx.(*seataContext.RootContext)
	return gtx.BeginWithTimeout(int32(6000), rootContext)
}

func (gtx ZormSeataGlobalTransaction) SeataCommit(ctx context.Context) error {
	rootContext := ctx.(*seataContext.RootContext)
	return gtx.Commit(rootContext)
}

func (gtx ZormSeataGlobalTransaction) SeataRollback(ctx context.Context) error {
	rootContext := ctx.(*seataContext.RootContext)
	return gtx.SeataRollback(rootContext)
}

func (gtx ZormSeataGlobalTransaction) GetSeataXID(ctx context.Context) string {
	rootContext := ctx.(*seataContext.RootContext)
	return rootContext.GetXID()
}
**/

type ISeataGlobalTransaction interface {
	//开启seata全局事务
	SeataBegin(ctx context.Context) error

	//提交seata全局事务
	SeataCommit(ctx context.Context) error

	//回滚seata全局事务
	SeataRollback(ctx context.Context) error

	//获取seata事务的XID
	GetSeataXID(ctx context.Context) string
}
