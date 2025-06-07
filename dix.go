package dix

import (
	"github.com/pubgo/dix/dixinternal"
	"github.com/pubgo/funk/assert"
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
// 这是框架的核心方法，支持多种输入类型，提供了统一的依赖注入接口。
// 一个方法处理所有依赖注入需求，包括获取实例和注入依赖。
//
// 支持的输入类型：
//   - 函数：func(deps...) - 解析函数参数并调用函数
//   - 结构体指针：&struct{} - 注入到结构体字段
//   - 接口：interface{} - 支持接口类型注入
//   - 切片：[]T - 注入所有匹配的实例
//   - 映射：map[string]T - 注入带名称的实例
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
//	// 函数注入（替代传统启动函数）
//	err := dix.Inject(container, func(logger Logger, db *Database, handlers []Handler) {
//	    // 使用注入的依赖启动服务器
//	    startServer(logger, db, handlers)
//	})
//
//	// 结构体注入
//	type Service struct {
//	    Logger Logger
//	    DB     *Database
//	}
//	var service Service
//	err := dix.Inject(container, &service)
//
//	// 获取单个依赖实例
//	var logger Logger
//	err := dix.Inject(container, func(l Logger) {
//	    logger = l
//	})
//
//	// 批量获取多个依赖实例
//	var logger Logger
//	var db *Database
//	var handlers []Handler
//	err := dix.Inject(container, func(l Logger, d *Database, h []Handler) {
//	    logger, db, handlers = l, d, h
//	})
//
//	// 方法注入示例
//	type UserService struct {
//	    logger Logger
//	    db     *Database
//	}
//	func (s *UserService) DixInjectLogger(logger Logger) { s.logger = logger }
//	func (s *UserService) DixInjectDatabase(db *Database) { s.db = db }
//
//	var service UserService
//	err := dix.Inject(container, &service)
//
// 参数：
//   - container: 依赖注入容器
//   - target: 注入目标（函数、结构体指针、接口等）
//   - opts: 注入选项（可选）
//
// 返回值：
//   - error: 注入失败时的错误信息
func Inject(container Container, target interface{}, opts ...Option) error {
	return container.Inject(target, opts...)
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
//	        return nil, fmt.Errorf("failed to load config: %w", err)
//	    }
//	    return config, nil
//	})
//
//	// 带依赖的提供者
//	dix.Provide(container, func(config *Config) (*Database, error) {
//	    db, err := sql.Open("postgres", config.DatabaseURL)
//	    if err != nil {
//	        return nil, fmt.Errorf("failed to connect: %w", err)
//	    }
//	    return &Database{DB: db}, nil
//	})
//
// 参数：
//   - container: 依赖注入容器
//   - provider: 提供者函数
func Provide(container Container, provider any) {
	assert.Must(container.Provide(provider))
}

// GetGraph 获取依赖关系图
//
//	container: 依赖注入容器
func GetGraph(container Container) *Graph {
	return container.Graph()
}

// 为了向后兼容，保留旧的类型别名
// Deprecated: 使用 Container 替代
type Dix = Container
