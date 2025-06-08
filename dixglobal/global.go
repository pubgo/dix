package dixglobal

import (
	"reflect"

	"github.com/pubgo/funk/errors"

	"github.com/pubgo/dix/dixinternal"
)

var _container = dixinternal.New(dixinternal.WithValuesNull())

// Example:
//
//	Provide(func() *Config { return &Config{Endpoint: "localhost:..."} }) // Configuration
//	Provide(NewDB)                                                        // Database connection
//	Provide(NewHTTPServer)                                                // Server
//
//	Inject(func(server *http.Server) { // Application startup
//		server.ListenAndServe()
//	})
//
//	// 或者带错误处理的函数注入
//	Inject(func(server *http.Server, db *Database) error {
//		if err := db.Connect(); err != nil {
//			return err
//		}
//		return server.ListenAndServe()
//	})
//
//	// 或者使用结构体注入
//	type App struct {
//		Server *http.Server
//		DB     *Database
//	}
//	var app App
//	Inject(&app) // 结构体字段注入
//
//	// 获取依赖实例的用法
//	var logger Logger
//	Inject(func(l Logger) { logger = l }) // 获取单个依赖
//
// For more usage details, see the documentation for the Container type.

// Provide registers an object constructor
func Provide(provider any) {
	errors.Must(_container.Provide(provider))
}

// Inject 统一的依赖注入方法
//
// 支持多种注入目标类型：
//   - 函数：解析参数并调用函数
//   - 结构体指针：注入到结构体字段
//   - 接口、切片、映射等其他类型
//
// 这个方法既可以注入依赖，也可以获取实例，提供统一的 API。
//
// 获取依赖实例的用法：
//   - 获取单个依赖：var logger Logger; Inject(func(l Logger) { logger = l })
//   - 获取多个依赖：var logger Logger; var db *DB; Inject(func(l Logger, d *DB) { logger, db = l, d })
//
// 参数：
//   - target: 注入目标（函数、结构体指针等）
//   - opts: 可选配置
func Inject[T any](target T, opts ...dixinternal.Option) T {
	vp := reflect.ValueOf(target)
	if vp.Kind() == reflect.Struct {
		errors.Must(_container.Inject(&target, opts...))
	} else {
		errors.Must(_container.Inject(target, opts...))
	}
	return target
}

// Graph returns the dependency graph
func Graph() *dixinternal.Graph {
	return _container.Graph()
}

// Container returns the global container instance
func Container() dixinternal.Container {
	return _container
}
