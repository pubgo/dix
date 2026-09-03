// 【功能】函数注入:函数类型(func)作为依赖注册与注入。
//
// 【原理】函数类型是一等公民,可像指针一样作为 provider 产物:
//   - 同一类型的多个 provider 全部按注册顺序执行;
//   - 单值注入取最后注册者的产物(后者覆盖前者);
//   - 切片注入 []Greet 聚合全部产物,顺序即注册顺序。
//
// 该语义由 dixinternal 的 TestPatternSingleValueLastProviderWins
// 与本目录 main_test.go 的 TestBuildGreets 锁定。
//
// 【运行】
//
//	cd example/inject-func && go run .
//
// 【预期输出】
//
//	single func: world
//	all funcs: hello world
package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
)

// Greet 是一个可注入的函数类型依赖。
type Greet func() string

func main() {
	single, all := buildGreets()

	fmt.Println("single func:", single())
	fmt.Print("all funcs:")
	for _, fn := range all {
		fmt.Print(" ", fn())
	}
	fmt.Println()
}

// buildGreets 为同一函数类型注册多个 provider 并注入,
// 返回单值注入结果(最后注册者)与切片聚合结果(注册顺序)。
func buildGreets() (Greet, []Greet) {
	di := dix.New()

	dix.Provide(di, func() Greet {
		return func() string { return "hello" }
	})
	dix.Provide(di, func() Greet {
		return func() string { return "world" }
	})

	var single Greet
	var all []Greet
	dix.Inject(di, func(g Greet, gs []Greet) { single, all = g, gs })
	return single, all
}
