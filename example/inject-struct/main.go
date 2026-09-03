// 【功能】结构体注入:把依赖填充到结构体的导出字段,支持嵌套递归。
//
// 【原理】对结构体指针调用 Inject 时,dix 逐个解析导出字段:
//   - 指针/接口/函数字段:按类型从容器解析;
//   - 嵌套结构体字段:递归注入;
//   - 未导出字段与基础类型字段:跳过,不报错。
//
// 该语义由 dixinternal 的 TestPatternStructInNestedResolution
// 与本目录 main_test.go 的 TestBuildApp 锁定。
//
// 【运行】
//
//	cd example/inject-struct && go run .
//
// 【预期输出】
//
//	app.DB.Config.DSN = postgres://localhost/app
//	app.Metadata.Version = v1
package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
)

type Config struct {
	DSN string
}

type Database struct {
	Config *Config
}

type Metadata struct {
	Version string
}

type App struct {
	DB       *Database
	Metadata *Metadata
}

func main() {
	app := buildApp()

	fmt.Println("app.DB.Config.DSN =", app.DB.Config.DSN)
	fmt.Println("app.Metadata.Version =", app.Metadata.Version)
}

// buildApp 注册三个 provider 并注入结构体指针,
// 返回完成了嵌套字段解析(DB.Config)的 App。
func buildApp() *App {
	di := dix.New()

	dix.Provide(di, func() *Config {
		return &Config{DSN: "postgres://localhost/app"}
	})

	// provider 的输入同样来自容器:Database 的构造依赖 *Config。
	dix.Provide(di, func(cfg *Config) *Database {
		return &Database{Config: cfg}
	})

	dix.Provide(di, func() *Metadata {
		return &Metadata{Version: "v1"}
	})

	app := &App{}
	dix.Inject(di, app)
	return app
}
