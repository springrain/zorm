package zorm

import "context"

// ISeataGlobalTransaction seata-golang的包装接口,隔离seata-golang的依赖
// 声明一个struct,实现这个接口,并配置实现 FuncSeataGlobalTransaction 函数
/**

//不使用proxy代理模式,全局托管,不修改业务代码,零侵入实现分布式事务
//tm.Implement(svc.ProxySvc)

// 业务代码中获取当前分布式事务的XID
//   ctx.Value("XID")

// 建议以下代码放到单独的文件里
//................//

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
	return gtx.Rollback(rootContext)
}

func (gtx ZormSeataGlobalTransaction) GetSeataXID(ctx context.Context) string {
	rootContext := ctx.(*seataContext.RootContext)
	return rootContext.GetXID()
}
//................//
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

	//重新包装为seata的context.RootContext
	//context.RootContext 如果后续使用了 context.WithValue,类型就是context.valueCtx 就会造成无法再类型断言为 context.RootContext
	//所以DBDao里使用了 seataRootContext变量,区分业务的ctx和seata的RootContext
	//SeataNewRootContext(ctx context.Context) context.Context
}
