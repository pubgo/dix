package dix

import (
	"fmt"
	"reflect"

	"github.com/pubgo/funk/errors"

	"github.com/pubgo/dix/dixinternal"
)

const (
	InjectMethodPrefix = dixinternal.InjectMethodPrefix
)

type (
	Option    = dixinternal.Option
	Options   = dixinternal.Options
	Container = dixinternal.Container
	Graph     = dixinternal.Graph
)

// WithValuesNull 配置选项：允许值为null
func WithValuesNull() Option {
	return dixinternal.WithValuesNull()
}

// New 创建新的依赖注入容器
func New(opts ...Option) Container {
	return dixinternal.New(opts...)
}

// Inject 统一的依赖注入方法
//
// 这是框架的核心方法，按照最原始的设计，只支持函数和结构体类型。
// 提供了统一的依赖注入接口，简化API设计。
//
// 支持的输入类型：
//   - 函数：func(deps...) - 解析函数参数并调用函数
//   - 结构体指针：&struct{} - 注入到结构体字段
//
// 函数注入规则：
//   - 函数只能有入参，不能有出参
//   - 函数参数类型必须在容器中已注册
//   - 支持的参数类型：指针(*T)、接口(interface{})、结构体(struct{})、切片([]T)、映射(map[string]T)
//   - 不支持基本类型参数：string, int, bool 等
//   - 支持可变参数：func(handlers ...Handler)
//
// 结构体注入规则：
//   - 字段必须是导出的（首字母大写）
//   - 支持嵌套结构体注入
//   - 支持方法注入（DixInject前缀的方法）
//
// 获取实例的用法：
//   - 通过函数参数获取：var logger Logger; Inject(container, func(l Logger) { logger = l })
//   - 直接在函数中使用：Inject(container, func(l Logger) { l.Log("message") })
//   - 批量获取：var logger Logger; var db *Database; Inject(container, func(l Logger, d *Database) { logger, db = l, d })
//
// 错误处理：
//   - 参数解析失败时返回详细错误信息
//   - 函数调用 panic 会被捕获并转换为错误
//   - 循环依赖会被检测并报错
//
// 示例：
//
//	// 函数注入
//	_, err := dix.Inject(container, func(logger Logger, db *Database, handlers []Handler) {
//	    // 使用注入的依赖启动服务器
//	    startServer(logger, db, handlers)
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 结构体注入
//	type Service struct {
//	    Logger Logger
//	    DB     *Database
//	}
//	service, err := dix.Inject(container, &Service{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 获取单个依赖实例
//	var logger Logger
//	_, err := dix.Inject(container, func(l Logger) {
//	    logger = l
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 批量获取多个依赖实例
//	var logger Logger
//	var db *Database
//	var handlers []Handler
//	_, err := dix.Inject(container, func(l Logger, d *Database, h []Handler) {
//	    logger, db, handlers = l, d, h
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 方法注入示例
//	type UserService struct {
//	    logger Logger
//	    db     *Database
//	}
//	func (s *UserService) DixInjectLogger(logger Logger) { s.logger = logger }
//	func (s *UserService) DixInjectDatabase(db *Database) { s.db = db }
//
//	service, err := dix.Inject(container, &UserService{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// 参数：
//   - container: 依赖注入容器
//   - target: 注入目标（函数或结构体指针）
//   - opts: 注入选项（可选）
//
// 返回值：
//   - T: 如果target是函数，返回零值（nil）；如果是结构体，返回注入后的结构体
//   - error: 注入失败时的错误信息
func Inject[T any](container Container, target T, opts ...Option) (result T, err error) {
	defer func() {
		if r := recover(); r != nil {
			var panicErr error
			if e, ok := r.(error); ok {
				panicErr = e
			} else {
				panicErr = fmt.Errorf("panic: %v", r)
			}
			err = errors.Wrap(panicErr, "dix: inject failed with panic")
		}
	}()

	vp := reflect.ValueOf(target)
	if vp.Kind() == reflect.Struct {
		if err := container.Inject(&target, opts...); err != nil {
			return target, errors.Wrap(err, "dix: inject failed")
		}
		return target, nil
	} else {
		return target, container.Inject(target, opts...)
	}
}

func InjectMust[T any](container Container, target T, opts ...Option) T {
	return errors.Must1(Inject(container, target, opts...))
}

// Provide 注册依赖提供者
//
// 支持的提供者函数签名：
//   - func() T                    - 简单提供者
//   - func() (T, error)           - 带错误处理的提供者
//   - func(dep1 D1, dep2 D2) T    - 带依赖的提供者
//   - func(dep1 D1, dep2 D2) (T, error) - 带依赖和错误处理的提供者
//
// 支持的输出类型：
//   - 指针类型：*T
//   - 接口类型：interface{}
//   - 结构体类型：struct{}
//   - Map类型：map[K]V
//   - Slice类型：[]T
//   - 函数类型：func(...)
//
// 不支持的类型：
//   - 基本类型：string, int, bool 等（请使用指针类型替代）
//
// 错误处理：
//   - 当提供者函数返回 (T, error) 时，如果 error 不为 nil，提供者调用失败
//   - 错误会被包装并包含提供者类型和位置信息
//   - 提供者注册失败时会 panic（使用 assert.Must）
//
// 示例：
//
//	// 简单提供者
//	dix.Provide(container, func() *Database {
//	    return &Database{Host: "localhost"}
//	})
//
//	// 带错误处理的提供者
//	dix.Provide(container, func() (*Config, error) {
//	    config, err := loadConfig()
//	    if err != nil {
//	        return nil, errors.Wrap(err, "failed to load config")
//	    }
//	    return config, nil
//	})
//
//	// 带依赖的提供者
//	dix.Provide(container, func(config *Config) (*Database, error) {
//	    db, err := sql.Open("postgres", config.DatabaseURL)
//	    if err != nil {
//	        return nil, errors.Wrap(err, "failed to connect")
//	    }
//	    return &Database{DB: db}, nil
//	})
//
// 参数：
//   - container: 依赖注入容器
//   - provider: 提供者函数
func Provide(container Container, provider any) {
	defer func() {
		if r := recover(); r != nil {
			var panicErr error
			if e, ok := r.(error); ok {
				panicErr = e
			} else {
				panicErr = fmt.Errorf("panic: %v", r)
			}
			panic(errors.Wrap(panicErr, "dix: provide failed with panic"))
		}
	}()

	errors.Must(container.Provide(provider))
}

// GetGraph 获取依赖关系图
//
//	container: 依赖注入容器
func GetGraph(container Container) *Graph {
	return container.Graph()
}
