// 【功能】dixcontext:把容器存入 context.Context,在调用链任意深处取用。
//
// 【原理】dixcontext.Create 把容器挂到 ctx 上:
//   - Get(ctx):取出容器,缺失时 panic(编程错误,快速失败);
//   - GetOrNil(ctx):缺失时返回 nil,适合可选场景;
//   - 适合中间件/请求处理链等不便显式传参的场合。
//
// 该语义由 dixcontext 包的 TestCreateAndGet/TestGetOrNil
// 与本目录 main_test.go 的 TestHandleRequestWithContainer 锁定。
//
// 【运行】
//
//	cd example/context-container && go run .
//
// 【预期输出】
//
//	handle request with dsn: postgres://localhost/app
//	optional lookup without container: nil=false
package main

import (
	"context"
	"fmt"

	"github.com/pubgo/dix/v2"
	"github.com/pubgo/dix/v2/dixcontext"
)

type Config struct{ DSN string }

func main() {
	fmt.Println("handle request with", handleRequest(context.Background()))

	// 无容器的 ctx:GetOrNil 返回 nil,不 panic。
	if dixcontext.GetOrNil(context.Background()) == nil {
		fmt.Println("optional lookup without container: nil=false")
	}
}

// handleRequest 模拟一次请求处理:容器经 ctx 传递,
// 中间件/处理器在任意深度都能取到容器并完成注入。
func handleRequest(ctx context.Context) string {
	di := dix.New()
	dix.Provide(di, func() *Config { return &Config{DSN: "postgres://localhost/app"} })

	// 业务入口处把容器挂上 ctx。
	ctx = dixcontext.Create(ctx, di)

	return loadConfigFromContext(ctx)
}

// loadConfigFromContext 模拟调用链深处:从 ctx 取容器并注入。
func loadConfigFromContext(ctx context.Context) string {
	var cfg *Config
	if err := dixcontext.Get(ctx).TryInject(func(c *Config) { cfg = c }); err != nil {
		panic(err)
	}
	return "dsn: " + cfg.DSN
}
