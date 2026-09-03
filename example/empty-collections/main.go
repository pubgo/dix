// 【功能】空集合容错:没有任何 provider 时,缺失的 map/slice 依赖解析为空集合。
//
// 【原理】默认容器(AllowValuesNull=true,即 WithValuesNull 语义)下:
//   - 缺失的 map/slice 依赖解析为"非 nil 的空集合";
//   - 缺失的单值依赖(指针/接口/函数)仍然报错,不受该开关影响;
//   - dix.WithRejectEmptyCollections() 反转该语义:缺失集合依赖直接报错。
//
// 该语义由 dixinternal 的 TestPatternNilCollectionsTolerance
// 与本目录 main_test.go 的 TestInjectMissingCollections 锁定。
//
// 【运行】
//
//	cd example/empty-collections && go run .
//
// 【预期输出】(另有 "provider not found" 告警日志,时间戳省略)
//
//	map field: nil=false, len=0
//	slice field: nil=false, len=0
//	function inject: map=0, slice=0
package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
)

type Handler func() string

// App 同时声明缺失的 map 与 slice 字段。
type App struct {
	Errors   map[string]error
	Handlers []Handler
}

func main() {
	errs, handlers := injectMissingCollections()

	fmt.Printf("map field: nil=%v, len=%d\n", errs == nil, len(errs))
	fmt.Printf("slice field: nil=%v, len=%d\n", handlers == nil, len(handlers))

	// 函数参数路径同样注入空集合。
	di := dix.New(dix.WithValuesNull())
	dix.Inject(di, func(m map[string]error, hs []Handler) {
		fmt.Printf("function inject: map=%d, slice=%d\n", len(m), len(hs))
	})
}

// injectMissingContainers 演示:没有任何 provider 时,
// map/slice 字段都注入为非 nil 的空集合。
func injectMissingCollections() (map[string]error, []Handler) {
	// WithValuesNull:缺失的集合依赖解析为空集合,而不是报错。
	di := dix.New(dix.WithValuesNull())

	app := &App{}
	dix.Inject(di, app)
	return app.Errors, app.Handlers
}
