// 【功能】泛型注入:InjectT / InjectTContext 直接构造并填充结构体。
//
// 【原理】泛型助手省去"声明变量 + 取地址 + Inject"三步:
//   - InjectT[T](di):分配零值 T 并注入其导出字段,返回填充后的值;
//   - InjectTContext[T](ctx, di):同上,且携带 trace 上下文;
//   - T 必须是结构体类型,否则 panic(fail-fast)。
//
// 该语义由根包的 TestInjectT/TestInjectTContext
// 与本目录 main_test.go 的 TestInjectGenericApp 锁定。
//
// 【运行】
//
//	cd example/inject-generic && go run .
//
// 【预期输出】
//
//	dsn: postgres://localhost/app
//	version: v1
package main

import (
	"context"
	"fmt"

	"github.com/pubgo/dix/v2"
)

type Config struct{ DSN string }

type Database struct{ Config *Config }

type Metadata struct{ Version string }

// App 声明全部依赖字段,由 InjectT 自动构造并填充。
type App struct {
	DB       *Database
	Metadata *Metadata
}

func main() {
	app := buildApp(context.Background())

	fmt.Println("dsn:", app.DB.Config.DSN)
	fmt.Println("version:", app.Metadata.Version)
}

// buildApp 用泛型助手完成"构造 + 注入"一步到位。
func buildApp(ctx context.Context) App {
	di := dix.New()

	dix.Provide(di, func() *Config { return &Config{DSN: "postgres://localhost/app"} })
	dix.Provide(di, func(c *Config) *Database { return &Database{Config: c} })
	dix.Provide(di, func() *Metadata { return &Metadata{Version: "v1"} })

	// InjectTContext:分配 App、注入字段,并携带 ctx 用于 trace 传播。
	// T 无法从返回位置推断,需显式指定。
	return dix.InjectTContext[App](ctx, di)
}
