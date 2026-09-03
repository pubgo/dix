// 【功能】函数注入:函数类型(func)作为依赖注册与注入。
//
// 【原理】函数类型是一等公民,可像指针一样作为 provider 产物:
//   - 同一类型的多个 provider 全部按注册顺序执行;
//   - 单值注入取最后注册者的产物(后者覆盖前者);
//   - 切片注入 []Greet 聚合全部产物,顺序即注册顺序。
//
// 该语义由 dixinternal 的 TestPatternSingleValueLastProviderWins 锁定。
//
// 【运行】
//
//	cd example/func && go run .
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
	di := dix.New()

	// 1) 为同一函数类型注册多个 provider:两者都会在解析时执行。
	dix.Provide(di, func() Greet {
		return func() string { return "hello" }
	})
	dix.Provide(di, func() Greet {
		return func() string { return "world" }
	})

	// 2) 单值注入:取最后注册的 provider 产物(后者覆盖前者)。
	dix.Inject(di, func(g Greet) {
		fmt.Println("single func:", g())
	})

	// 3) 切片注入:聚合全部产物,顺序与注册顺序一致。
	dix.Inject(di, func(all []Greet) {
		fmt.Print("all funcs:")
		for _, fn := range all {
			fmt.Print(" ", fn())
		}
		fmt.Println()
	})
}
