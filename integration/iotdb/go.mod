module zorm-iotdb-integration

go 1.25

require (
	gitee.com/chunanyong/zorm v0.0.0
	github.com/apache/iotdb-client-go/v2 v2.0.10
)

require (
	github.com/apache/thrift v0.24.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
)

replace gitee.com/chunanyong/zorm => ../..
