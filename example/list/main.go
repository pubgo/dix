// 【功能】列表聚合注入:同一类型的多个 provider 聚合进一个切片。
//
// 【原理】列表([]T)是聚合查询:
//   - 所有产出 T(或 []T)的 provider 全部执行,产物按注册顺序拼接;
//   - provider 内部返回的切片元素顺序保持不变;
//   - 单值注入与列表注入互不干扰:单值仍取最后注册者。
//
// 该语义由 dixinternal 的 TestPatternListAggregationOrder 锁定。
//
// 【运行】
//
//	cd example/list && go run .
//
// 【预期输出】
//
//	latest: [trace] request
//	chain size: 3
//	  middleware[0]: [auth] request
//	  middleware[1]: [log] request
//	  middleware[2]: [trace] request
package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
)

type Middleware func(string) string

func main() {
	di := dix.New()

	dix.Provide(di, func() []Middleware {
		return []Middleware{
			func(s string) string { return "[auth] " + s },
		}
	})

	dix.Provide(di, func() []Middleware {
		return []Middleware{
			func(s string) string { return "[log] " + s },
		}
	})

	// 单值 provider:单值注入取它(最后注册者),列表注入也会聚合它。
	dix.Provide(di, func() Middleware {
		return func(s string) string { return "[trace] " + s }
	})

	dix.Inject(di, func(latest Middleware, chain []Middleware) {
		fmt.Println("latest:", latest("request"))
		fmt.Println("chain size:", len(chain))
		for i, mw := range chain {
			fmt.Printf("  middleware[%d]: %s\n", i, mw("request"))
		}
	})
}
