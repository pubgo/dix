// 【功能】Map 空集合容错:即使没有任何 map 依赖的 provider,注入也不失败。
//
// 【原理】默认容器(AllowValuesNull=true,即 WithValuesNull 语义)下:
//   - 缺失的 map 依赖解析为"非 nil 的空 map",缺失的 slice 依赖同理;
//   - 缺失的单值依赖(指针/接口/函数)仍然报错,不受该开关影响;
//   - 若希望缺失集合依赖直接失败,改用 dix.WithRejectEmptyCollections()。
//
// 该语义由 dixinternal 的 TestPatternNilCollectionsTolerance
// 与本目录 main_test.go 的 TestInjectMissingErrors 锁定。
//
// 【运行】
//
//	cd example/map-nil && go run .
//
// 【预期输出】(另有两行 "provider not found" 告警日志,时间戳省略)
//
//	Errors is nil: false
//	Errors length: 0
//	function inject length: 0
package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
)

func main() {
	errs := injectMissingErrors()

	fmt.Printf("Errors is nil: %v\n", errs == nil)
	fmt.Printf("Errors length: %d\n", len(errs))

	di := dix.New(dix.WithValuesNull())
	dix.Inject(di, func(m map[string]error) {
		fmt.Printf("function inject length: %d\n", len(m))
	})
}

// injectMissingErrors 演示:没有任何 provider 时,
// map 字段注入为非 nil 的空 map。
func injectMissingErrors() map[string]error {
	// WithValuesNull:缺失的集合依赖解析为空集合,而不是报错。
	di := dix.New(dix.WithValuesNull())

	type App struct {
		Errors map[string]error
	}

	app := &App{}
	dix.Inject(di, app)
	return app.Errors
}
