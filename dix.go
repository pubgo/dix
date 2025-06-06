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

// Inject 注入函数
//
// 解析函数参数并调用函数。
// 通常用于启动函数或回调函数的依赖注入。
//
// 参数：
//   - container: 依赖注入容器
//   - fn: 目标函数，必须是函数类型
//   - opts: 注入选项（可选）
//
// 函数注入规则：
//   - 函数只能有入参，不能有出参
//   - 函数参数类型必须在容器中已注册
//   - 支持的参数类型：
//   - 指针类型：*T
//   - 接口类型：interface{}
//   - 结构体类型：struct{}
//   - 切片类型：[]T（注入所有匹配的实例）
//   - 映射类型：map[string]T（注入带名称的实例）
//   - 不支持基本类型参数：string, int, bool 等
//   - 支持可变参数：func(handlers ...Handler)
//
// 函数限制：
//   - 函数不能有返回值（包括 error）
//   - 函数参数不能是基本类型
//
// 错误处理：
//   - 参数解析失败时返回详细错误信息
//   - 函数调用 panic 会被捕获并转换为错误
//   - 如果函数有返回值，注册时会被拒绝
//
// 示例：
//
//	// 有效的启动函数
//	func StartServer(logger Logger, db *Database, handlers []Handler) {
//	    // 使用注入的依赖启动服务器
//	}
//
//	// 有效的回调函数
//	func ProcessRequest(ctx Context, service *UserService) {
//	    // 处理请求
//	}
//
//	// 无效的函数（有返回值）
//	func InvalidFunc(logger Logger) error {
//	    return nil // 不允许有返回值
//	}
//
//	// 无效的函数（基本类型参数）
//	func InvalidFunc2(name string, port int) {
//	    // 不允许基本类型参数
//	}
//
//	// 使用示例
//	container := dix.New()
//	container.Provide(NewLogger)
//	container.Provide(NewDatabase)
//	container.Provide(NewUserService)
//
//	// 注入并调用启动函数
//	err := dix.Inject(container, StartServer)
//	if err != nil {
//	    log.Fatal(err)
//	}
func Inject(container Container, fn interface{}, opts ...Option) error {
	return container.Inject(fn, opts...)
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

// Get 获取指定类型的实例（泛型版本）
//
// 支持获取的类型：
//   - 单个实例：T（直接指定类型）
//   - 切片：[]T（获取所有 T 类型的实例）
//   - 映射：map[string]T（获取带名称的 T 类型实例）
//
// 类型解析规则：
//   - 接口类型会匹配所有实现该接口的类型
//   - 结构体类型精确匹配
//   - 指针类型匹配对应的指针实例
//
// 错误情况：
//   - 类型未注册：返回 ErrTypeNotFound
//   - 循环依赖：返回 ErrCircularDependency
//   - 提供者调用失败：返回包装后的错误
//
// 示例：
//
//	// 获取单个实例
//	logger, err := dix.Get[Logger](container)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 获取切片
//	handlers, err := dix.Get[[]Handler](container)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 获取映射
//	databases, err := dix.Get[map[string]Database](container)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// 参数：
//   - container: 依赖注入容器
//   - opts: 可选配置
//
// 返回值：
//   - T: 请求的实例
//   - error: 获取失败时的错误信息
func Get[T any](container Container, opts ...Option) (T, error) {
	return dixinternal.Get[T](container, opts...)
}

// MustGet 获取指定类型的实例，失败时panic（泛型版本）
//
// 功能与 Get 相同，但在获取失败时会 panic 而不是返回错误。
// 适用于确信实例一定存在的场景，如应用启动阶段。
//
// 支持获取的类型：
//   - 单个实例：T（直接指定类型）
//   - 切片：[]T（获取所有 T 类型的实例）
//   - 映射：map[string]T（获取带名称的 T 类型实例）
//
// 使用场景：
//   - 应用启动阶段，确信依赖已正确注册
//   - 测试代码中，简化错误处理
//   - 配置阶段，依赖缺失应该立即失败
//
// 示例：
//
//	// 获取单个实例（确信存在）
//	logger := dix.MustGet[Logger](container)
//
//	// 获取切片（可能为空）
//	handlers := dix.MustGet[[]Handler](container)
//
//	// 在应用启动中使用
//	func main() {
//	    container := setupContainer()
//
//	    // 这些依赖必须存在，否则应用无法启动
//	    logger := dix.MustGet[Logger](container)
//	    db := dix.MustGet[*Database](container)
//
//	    startServer(logger, db)
//	}
//
// 参数：
//   - container: 依赖注入容器
//   - opts: 可选配置
//
// 返回值：
//   - T: 请求的实例
//
// 注意：失败时会 panic，请确保在适当的场景下使用
func MustGet[T any](container Container, opts ...Option) T {
	return dixinternal.MustGet[T](container, opts...)
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
