package zorm

import "context"

// ISeataGlobalTransaction seata-golang的包装接口,隔离seata-golang的依赖
// 声明一个struct,实现这个接口,并配置实现 FuncSeataGlobalTransaction 函数
/**
//实现zorm.ISeataGlobalTransaction接口
func (gtx ZormSeataGlobalTransaction) SeataBegin(ctx context.Context) error {
	rootContext, ok := ctx.(*seataContext.RootContext)
	if !ok {
		rootContext = seataContext.NewRootContext(ctx)
	}
	return gtx.BeginWithTimeout(int32(6000), rootContext)
}

func (gtx ZormSeataGlobalTransaction) SeataCommit(ctx context.Context) error {
	rootContext, ok := ctx.(*seataContext.RootContext)
	if !ok {
		rootContext = seataContext.NewRootContext(ctx)
	}
	return gtx.Commit(rootContext)
}

func (gtx ZormSeataGlobalTransaction) SeataRollback(ctx context.Context) error {
	rootContext, ok := ctx.(*seataContext.RootContext)
	if !ok {
		rootContext = seataContext.NewRootContext(ctx)
	}
	return gtx.Rollback(rootContext)
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

	//重新包装为seata的context.RootContext
	//context.RootContext 如果后续使用了 context.WithValue,类型就是context.valueCtx 就会造成无法再类型断言为 context.RootContext
	//SeataNewRootContext(ctx context.Context) context.Context
}
