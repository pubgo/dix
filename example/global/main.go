// 【功能】dixglobal:进程级全局容器,免创建直接注册与注入。
//
// 【原理】dixglobal 内部持有一个全局 *dix.Dix(默认开启 WithValuesNull):
//   - Provide/Inject 直接操作全局容器,适合简单应用;
//   - 容器非线程安全:全局容器只应在启动阶段(单协程)注册;
//   - 复杂应用仍建议显式创建 dix.New() 并传递容器。
//
// 该语义由 dixglobal 包的 TestProvideAndInject/TestInjectT
// 与本目录 main_test.go 的 TestBuildGlobalApp 锁定。
//
// 【运行】
//
//	cd example/global && go run .
//
// 【预期输出】
//
//	dix version: v2.0.2
//	endpoint: localhost:8080
package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
	"github.com/pubgo/dix/v2/dixglobal"
)

type Server struct{ Endpoint string }

func main() {
	fmt.Println("dix version:", dix.Version())
	fmt.Println("endpoint:", buildGlobalApp().Endpoint)
}

// buildGlobalApp 在全局容器上注册并注入(启动阶段单协程)。
func buildGlobalApp() *Server {
	dixglobal.Provide(func() *Server {
		return &Server{Endpoint: "localhost:8080"}
	})

	// 函数注入:回调参数由全局容器解析。
	var srv *Server
	dixglobal.Inject(func(s *Server) { srv = s })
	return srv
}
