// 【功能】map[string][]T 分组聚合:每个 provider 按 key 贡献一组列表,注入时合并。
//
// 【原理】map 的 value 是切片时,聚合规则是两层组合:
//   - 每个 key 是独立命名空间;
//   - 同一 key 下,多个 provider 的切片产物按注册顺序拼接;
//   - provider 切片内的元素顺序保持不变;nil 元素被跳过;
//   - 聚合元素必须是指针/接口/函数(dix 只管理可寻址类型,
//     裸 struct 元素不支持聚合查询)。
//
// 该语义由 dixinternal 的 TestMapOfLists
// 与本目录 main_test.go 的 TestBuildRoutes 锁定。
//
// 【运行】
//
//	cd example/inject-map-list && go run .
//
// 【预期输出】
//
//	api routes: [/users /orders /health]
//	web routes: [/home]
package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
)

type Route struct{ Path string }

func main() {
	routes := buildRoutes()

	fmt.Println("api routes:", paths(routes["api"]))
	fmt.Println("web routes:", paths(routes["web"]))
}

// buildRoutes 用两个 provider 向 map[string][]*Route 贡献路由:
// 同 key("api")的切片按注册顺序拼接,不同 key("web")独立分组。
func buildRoutes() map[string][]*Route {
	di := dix.New()

	dix.Provide(di, func() map[string][]*Route {
		return map[string][]*Route{
			"api": {{Path: "/users"}, {Path: "/orders"}},
		}
	})

	dix.Provide(di, func() map[string][]*Route {
		return map[string][]*Route{
			"api": {{Path: "/health"}},
			"web": {{Path: "/home"}},
		}
	})

	var routes map[string][]*Route
	dix.Inject(di, func(m map[string][]*Route) { routes = m })
	return routes
}

func paths(rs []*Route) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.Path)
	}
	return out
}
