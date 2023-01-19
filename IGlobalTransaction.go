/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package zorm

import "context"

// IGlobalTransaction 托管全局分布式事务接口
type IGlobalTransaction interface {
	// BeginGTX 开启全局分布式事务
	BeginGTX(ctx context.Context, globalRootContext context.Context) error

	// CommitGTX 提交全局分布式事务.不能命名为 Commit,不然就和gtx的Commit一致了,就递归调用自己了.......
	CommitGTX(ctx context.Context, globalRootContext context.Context) error

	// RollbackGTX 回滚全局分布式事务
	RollbackGTX(ctx context.Context, globalRootContext context.Context) error

	// GetGTXID 获取全局分布式事务的XID
	GetGTXID(ctx context.Context, globalRootContext context.Context) (string, error)

	// 重新包装为 seata/hptx 的context.RootContext
	// context.RootContext 如果后续使用了 context.WithValue,类型就是context.valueCtx 就会造成无法再类型断言为 context.RootContext
	// 所以DBDao里使用了 globalRootContext变量,区分业务的ctx和分布式事务的RootContext
	// NewRootContext(ctx context.Context) context.Context
}
