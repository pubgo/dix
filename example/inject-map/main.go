// 【功能】Map 命名空间注入:用 map key 给同类型依赖分组隔离。
//
// 【原理】provider 返回 map[string]T 时,每个 key 成为独立命名空间:
//   - 多个 provider 可向同一个 map[string]T 贡献不同 key,注入时合并;
//   - 相同 key 由后注册的 provider 覆盖;
//   - provider 返回 map 中的 nil 值会被跳过;
//   - 空字符串 key 归入 default 分组,注入后表现为 key "default"。
//
// 该语义由 dixinternal 的 TestPatternMapNamespaceAggregation
// 与本目录 main_test.go 的 TestBuildDatabases 锁定。
//
// 【运行】
//
//	cd example/map && go run .
//
// 【预期输出】(另有一条 "provider value already exists" 提示日志,可忽略)
//
//	master: postgres://master/db
//	slave: postgres://slave/db
//	analytics: postgres://analytics/db
package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
)

type Database struct {
	DSN string
}

func main() {
	dbs := buildDatabases()

	fmt.Println("master:", dbs["master"].DSN)
	fmt.Println("slave:", dbs["slave"].DSN)
	fmt.Println("analytics:", dbs["analytics"].DSN)
}

// buildDatabases 用两个 provider 向同一个 map[string]*Database
// 贡献命名空间,注入后返回合并结果。
func buildDatabases() map[string]*Database {
	di := dix.New()

	dix.Provide(di, func() map[string]*Database {
		return map[string]*Database{
			"master": {DSN: "postgres://master/db"},
			"slave":  {DSN: "postgres://slave/db"},
		}
	})

	dix.Provide(di, func() map[string]*Database {
		return map[string]*Database{
			"analytics": {DSN: "postgres://analytics/db"},
		}
	})

	var dbs map[string]*Database
	dix.Inject(di, func(m map[string]*Database) { dbs = m })
	return dbs
}
