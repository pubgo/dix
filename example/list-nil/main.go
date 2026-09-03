// 【功能】列表空集合容错:即使没有任何 handler provider,注入也不失败。
//
// 【原理】默认容器(AllowValuesNull=true,即 WithValuesNull 语义)下:
//   - 缺失的 slice 依赖解析为"非 nil 的空切片",map 依赖同理;
//   - 缺失的单值依赖(指针/接口/函数)仍然报错,不受该开关影响;
//   - 若希望缺失集合依赖直接失败,改用 dix.WithRejectEmptyCollections()。
//
// 该语义由 dixinternal 的 TestPatternNilCollectionsTolerance
// 与本目录 main_test.go 的 TestInjectMissingHandlers 锁定。
//
// 【运行】
//
//	cd example/list-nil && go run .
//
// 【预期输出】(另有两行 "provider not found" 告警日志,时间戳省略)
//
//	Handlers is nil: false
//	Handlers length: 0
//	function inject length: 0
package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
)

type Handler func() string

func main() {
	handlers := injectMissingHandlers()

	fmt.Printf("Handlers is nil: %v\n", handlers == nil)
	fmt.Printf("Handlers length: %d\n", len(handlers))

	di := dix.New(dix.WithValuesNull())
	dix.Inject(di, func(hs []Handler) {
		fmt.Printf("function inject length: %d\n", len(hs))
	})
}

// injectMissingHandlers 演示:没有任何 provider 时,
// slice 字段与切片参数都注入为非 nil 的空切片。
func injectMissingHandlers() []Handler {
	di := dix.New(dix.WithValuesNull())

	type App struct {
		Handlers []Handler
	}

	app := &App{}
	dix.Inject(di, app)
	return app.Handlers
}
