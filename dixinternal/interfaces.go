package dixinternal

import (
	"reflect"
)

// Container 依赖注入容器接口
//
// 负责注册和管理依赖提供者，执行依赖注入，并提供依赖关系图和配置选项。
type Container interface {
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
	//   - 函数类型：func(...) - 仅支持有入参无出参的函数
	//
	// 函数类型限制：
	//   - 当输出类型为函数时，该函数只能有入参，不能有出参
	//   - 函数入参支持的类型：指针(*T)、接口(interface{})、Map(map[K]V)、Slice([]T)、结构体(struct{})
	//   - 示例：func(logger Logger, db *Database, handlers []Handler) - 有效
	//   - 示例：func() string - 无效（有出参）
	//   - 示例：func(name string) - 无效（入参为基本类型）
	//
	// 不支持的类型：
	//   - 基本类型：string, int, bool 等（请使用指针类型替代）
	//
	// 错误处理：
	//   - 当提供者函数返回 (T, error) 时，如果 error 不为 nil，提供者调用失败
	//   - 错误会被包装并包含提供者类型和位置信息
	Provide(provider interface{}) error

	// Inject 统一的依赖注入方法
	//
	// 这是框架的核心方法，支持多种输入类型，提供了统一的依赖注入接口。
	// 它既可以进行依赖注入，也可以获取依赖实例。
	//
	// 支持的注入目标：
	//   - 函数：func(deps...) - 解析函数参数并调用函数
	//   - 结构体指针：&struct{} - 注入到结构体字段
	//   - 接口：interface{} - 支持接口类型注入
	//   - 切片：[]T - 注入所有匹配的实例
	//   - 映射：map[string]T - 注入带名称的实例
	//
	// 函数注入规则：
	//   - 函数只能有入参，不能有出参
	//   - 函数参数类型必须在容器中已注册
	//   - 支持的参数类型：指针(*T)、接口(interface{})、Map(map[K]V)、Slice([]T)、结构体(struct{})
	//   - 不支持基本类型参数：string, int, bool 等
	//   - 支持可变参数：func(handlers ...Handler)
	//   - 示例：func(logger Logger, db *Database, handlers []Handler) - 有效
	//   - 示例：func() error - 无效（有出参）
	//   - 示例：func(name string) - 无效（参数为基本类型）
	//
	// 结构体注入规则：
	//   - 字段必须是导出的（首字母大写）
	//   - 支持嵌套结构体注入
	//   - 支持方法注入（DixInject前缀的方法）
	//
	// 获取依赖实例的用法：
	//   - 获取单个依赖：var logger Logger; container.Inject(func(l Logger) { logger = l })
	//   - 批量获取依赖：var logger Logger; var db *DB; container.Inject(func(l Logger, d *DB) { logger, db = l, d })
	//
	// 这种设计使得一个方法就能处理所有的依赖注入需求，提供了更加统一和灵活的API。
	Inject(target interface{}, opts ...Option) error

	// Graph 获取依赖关系图
	//
	// 返回包含以下信息的图形：
	//   - Providers: 提供者之间的依赖关系（DOT格式）
	//   - Objects: 已创建对象的关系图（DOT格式）
	Graph() *Graph

	// Option 获取容器配置
	Option() Options
}

// Provider 依赖提供者接口
//
// 提供者负责创建和管理特定类型的实例。
// 支持的提供者类型包括函数提供者、值提供者等。
//
// 重要设计改进：
//   - Provider 现在可以暴露多个类型，而不仅仅是单一类型
//   - 当 provider 返回结构体时，可以同时提供结构体类型及其字段类型
//   - 这解决了"结构体 provider 无法被类型匹配"的根本问题
type Provider interface {
	// ProvidedTypes 返回此提供者能够提供的所有类型
	//
	// 返回此提供者能够创建的所有实例类型列表。
	// 这是核心改进：从单一类型扩展为多类型支持。
	//
	// 类型提供规则：
	//   - 主要类型：provider 函数的直接返回类型
	//   - 字段类型：如果返回结构体，还包括结构体的所有导出字段类型
	//   - 嵌套类型：递归包括嵌套结构体的字段类型
	//
	// 示例：
	//   func() AppConfig { return AppConfig{Logger: logger, DB: db} }
	//   能提供的类型：[AppConfig, Logger, Database]
	//
	// 支持的类型：
	//   - 指针类型：*T
	//   - 接口类型：interface{}
	//   - 结构体类型：struct{}
	//   - Map类型：map[K]V
	//   - Slice类型：[]T
	//   - 函数类型：func(...) - 仅支持有入参无出参的函数
	//
	// 函数类型限制：
	//   - 函数只能有入参，不能有出参
	//   - 函数入参支持的类型：指针(*T)、接口(interface{})、Map(map[K]V)、Slice([]T)、结构体(struct{})
	//   - 不支持基本类型入参：string, int, bool 等
	ProvidedTypes() []reflect.Type

	// PrimaryType 返回主要提供的类型
	//
	// 返回 provider 函数的直接返回类型。
	// 这是为了保持向后兼容性而保留的方法。
	//
	// 对于结构体类型的 provider，这会返回结构体类型本身，
	// 而 ProvidedTypes() 会返回结构体及其所有字段类型。
	PrimaryType() reflect.Type

	// CanProvide 检查是否能提供指定类型
	//
	// 检查此提供者是否能够创建指定类型的实例。
	// 这比遍历 ProvidedTypes() 更高效。
	//
	// 参数：
	//   - typ: 要检查的类型
	//
	// 返回值：
	//   - bool: true 表示能提供，false 表示不能提供
	//
	// 检查逻辑：
	//   - 精确类型匹配
	//   - 接口类型匹配（实现关系）
	//   - 嵌入类型匹配（组合关系）
	CanProvide(typ reflect.Type) bool

	// Invoke 调用提供者函数
	//
	// 参数：
	//   - args: 依赖参数的反射值列表，顺序必须与 Dependencies() 返回的顺序一致
	//
	// 返回值：
	//   - []reflect.Value: 提供者函数的返回值列表
	//   - error: 调用失败时的错误信息
	//
	// 多类型支持：
	//   - 调用后，返回的主要结果可以用于提取不同类型的值
	//   - 如果主要结果是结构体，可以从中提取字段值来满足不同类型的需求
	//   - 这使得一次调用可以服务于多种类型的依赖请求
	//
	// 错误处理：
	//   - 如果提供者函数返回 (T, error)，当 error 不为 nil 时调用失败
	//   - 如果提供者函数发生 panic，会被捕获并转换为错误
	//   - 错误信息包含提供者类型、位置和调用栈信息
	//
	// 性能优化：
	//   - 调用时间会被记录用于性能分析
	//   - 支持并发调用（如果提供者函数是线程安全的）
	//   - 结果可以被缓存用于多种类型的请求
	Invoke(args []reflect.Value) ([]reflect.Value, error)

	// ProvideFor 为指定类型提供实例
	//
	// 这是新增的核心方法，用于为特定类型提供实例。
	// 它封装了 Invoke 调用和结果提取的逻辑。
	//
	// 参数：
	//   - typ: 要提供的目标类型
	//   - args: 依赖参数的反射值列表
	//
	// 返回值：
	//   - reflect.Value: 目标类型的实例
	//   - error: 提供失败时的错误信息
	//
	// 提供逻辑：
	//   - 如果目标类型是主要类型，直接返回 Invoke 的结果
	//   - 如果目标类型是字段类型，从结构体结果中提取对应字段
	//   - 如果目标类型是接口，检查结果是否实现该接口
	//   - 支持递归提取嵌套结构体的字段
	//
	// 错误情况：
	//   - 目标类型不在 ProvidedTypes() 列表中
	//   - Invoke 调用失败
	//   - 结果类型转换失败
	//   - 字段提取失败
	ProvideFor(typ reflect.Type, args []reflect.Value) (reflect.Value, error)

	// Dependencies 返回依赖的类型列表
	//
	// 返回此提供者函数所需的依赖类型列表。
	// 列表顺序与提供者函数的参数顺序一致。
	//
	// 支持的依赖类型：
	//   - 单个依赖：T
	//   - 切片依赖：[]T（注入所有 T 类型的实例）
	//   - 映射依赖：map[string]T（注入带名称的 T 类型实例）
	//
	// 依赖解析规则：
	//   - 接口类型会匹配所有实现该接口的类型
	//   - 结构体类型精确匹配
	//   - 指针类型匹配对应的指针实例
	Dependencies() []Dependency

	// IsInitialized 是否已初始化
	//
	// 返回此提供者是否已经被调用过。
	// 用于实现单例模式和避免重复初始化。
	IsInitialized() bool

	// SetInitialized 设置初始化状态
	//
	// 标记此提供者的初始化状态。
	// 通常在提供者被成功调用后设置为 true。
	//
	// 参数：
	//   - bool: true 表示已初始化，false 表示未初始化
	SetInitialized(bool)
}

// Dependency 依赖描述接口
//
// 描述提供者函数的单个依赖项。
// 包含类型信息和注入方式（单个、列表、映射）。
type Dependency interface {
	// Type 依赖类型
	//
	// 返回依赖的具体类型。
	// 对于集合类型（Map、Slice），返回元素类型。
	//
	// 示例：
	//   - 单个依赖 Logger -> reflect.TypeOf((*Logger)(nil)).Elem()
	//   - 切片依赖 []Handler -> reflect.TypeOf((*Handler)(nil)).Elem()
	//   - 映射依赖 map[string]Database -> reflect.TypeOf((*Database)(nil)).Elem()
	Type() reflect.Type

	// IsMap 是否为Map类型
	//
	// 返回此依赖是否应该以映射形式注入。
	// 当为 true 时，会注入 map[string]T 类型的实例。
	//
	// 映射注入规则：
	//   - 键为字符串类型，通常是提供者的名称或标识
	//   - 值为依赖类型的实例
	//   - 支持嵌套：map[string][]T
	IsMap() bool

	// IsList 是否为List类型
	//
	// 返回此依赖是否应该以切片形式注入。
	// 当为 true 时，会注入 []T 类型的实例。
	//
	// 列表注入规则：
	//   - 注入所有匹配类型的实例
	//   - 保持注册顺序
	//   - 支持空切片（如果没有匹配的提供者）
	IsList() bool

	// Validate 验证依赖是否有效
	//
	// 检查依赖类型是否被框架支持。
	// 在提供者注册时调用，确保依赖可以被正确解析。
	//
	// 验证规则：
	//   - 类型必须是支持的类型（指针、接口、结构体、Map、Slice、函数）
	//   - Map 和 Slice 的元素类型也必须是支持的类型
	//   - 不能是基本类型（string、int、bool 等）
	//
	// 返回值：
	//   - error: 验证失败时的错误信息，包含详细的类型信息
	Validate() error
}

// Resolver 依赖解析器接口
//
// 负责解析和创建依赖实例。
// 处理循环依赖检测、实例缓存和依赖图构建。
type Resolver interface {
	// Resolve 解析依赖
	//
	// 根据类型解析并返回对应的实例。
	// 支持单例模式和原型模式。
	//
	// 参数：
	//   - typ: 要解析的类型
	//   - opts: 解析选项（如是否允许 null 值）
	//
	// 返回值：
	//   - reflect.Value: 解析得到的实例
	//   - error: 解析失败时的错误信息
	//
	// 解析策略：
	//   - 优先使用已缓存的实例（单例模式）
	//   - 如果没有缓存，调用对应的提供者
	//   - 递归解析提供者的依赖
	//   - 检测并防止循环依赖
	Resolve(typ reflect.Type, opts Options) (reflect.Value, error)

	// ResolveAll 解析所有依赖
	//
	// 批量解析多个依赖，通常用于提供者函数的参数解析。
	//
	// 参数：
	//   - deps: 依赖列表
	//   - opts: 解析选项
	//
	// 返回值：
	//   - []reflect.Value: 解析得到的实例列表，顺序与输入一致
	//   - error: 任何一个依赖解析失败时的错误信息
	//
	// 优化特性：
	//   - 支持并行解析（如果依赖之间无关联）
	//   - 智能缓存和重用
	//   - 详细的错误上下文
	ResolveAll(deps []Dependency, opts Options) ([]reflect.Value, error)
}

// Injector 注入器接口
//
// 负责将解析得到的依赖注入到目标对象中。
// 支持结构体字段注入、方法注入和函数调用注入。
type Injector interface {
	// InjectStruct 注入结构体
	//
	// 将依赖注入到结构体的字段中。
	// 支持嵌套结构体和方法注入。
	//
	// 参数：
	//   - target: 目标结构体的反射值（必须是指针类型）
	//   - opts: 注入选项
	//
	// 注入规则：
	//   - 只注入导出字段（首字母大写）
	//   - 字段类型必须在容器中已注册
	//   - 支持以下字段类型：
	//     * 指针类型：*T
	//     * 接口类型：interface{}
	//     * 结构体类型：struct{}（会递归注入）
	//     * 切片类型：[]T（注入所有匹配的实例）
	//     * 映射类型：map[string]T（注入带名称的实例）
	//
	// 方法注入：
	//   - 查找以 "DixInject" 开头的方法
	//   - 方法参数会被自动解析和注入
	//   - 方法必须是导出的（首字母大写）
	//
	// 错误处理：
	//   - 如果字段类型未注册，根据选项决定是否报错
	//   - 如果允许 null 值，未注册的字段保持零值
	//   - 循环依赖会被检测并报错
	//
	// 示例：
	//   type Service struct {
	//       Logger   Logger    // 接口注入
	//       DB       *Database // 指针注入
	//       Handlers []Handler // 切片注入
	//       Config   Config    // 结构体注入（递归）
	//   }
	//
	//   func (s *Service) DixInjectCache(cache Cache) {
	//       s.cache = cache // 方法注入
	//   }
	InjectStruct(target reflect.Value, opts Options) error

	// InjectFunc 注入函数
	//
	// 解析函数参数并调用函数。
	// 通常用于启动函数或回调函数的依赖注入。
	//
	// 参数：
	//   - fn: 目标函数的反射值
	//   - opts: 注入选项
	//
	// 函数注入规则：
	//   - 函数可以没有返回值，或者有一个 error 返回值
	//   - 函数参数类型必须在容器中已注册
	//   - 支持的参数类型：
	//     * 指针类型：*T
	//     * 接口类型：interface{}
	//     * 结构体类型：struct{}
	//     * 切片类型：[]T（注入所有匹配的实例）
	//     * 映射类型：map[string]T（注入带名称的实例）
	//   - 不支持基本类型参数：string, int, bool 等
	//   - 支持可变参数：func(handlers ...Handler)
	//
	// 函数返回值规则：
	//   - 无返回值：func(deps...) - 常规注入函数
	//   - 一个 error 返回值：func(deps...) error - 可以报告执行错误
	//   - 不能有其他类型的返回值
	//
	// 错误处理：
	//   - 参数解析失败时返回详细错误信息
	//   - 函数调用 panic 会被捕获并转换为错误
	//   - 如果函数返回 error，该错误会被包装并向上传播
	//   - 如果函数有非 error 类型的返回值，注册时会被拒绝
	//
	// 示例：
	//   // 有效的启动函数（无返回值）
	//   func StartServer(logger Logger, db *Database, handlers []Handler) {
	//       // 使用注入的依赖启动服务器
	//   }
	//
	//   // 有效的回调函数（error 返回值）
	//   func ProcessRequest(ctx Context, service *UserService) error {
	//       // 处理请求，可能返回错误
	//       return service.Process(ctx)
	//   }
	//
	//   // 无效的函数（非 error 返回值）
	//   func InvalidFunc(logger Logger) string {
	//       return "not allowed" // 只允许 error 返回值
	//   }
	//
	//   // 无效的函数（基本类型参数）
	//   func InvalidFunc2(name string, port int) {
	//       // 不允许基本类型参数
	//   }
	InjectFunc(fn reflect.Value, opts Options) error
}

// CycleDetector 循环依赖检测器接口
type CycleDetector interface {
	// DetectCycle 检测循环依赖
	DetectCycle(providers map[reflect.Type][]Provider) ([]reflect.Type, error)
}

// GraphRenderer 图形渲染器接口
type GraphRenderer interface {
	// RenderProviders 渲染提供者图
	RenderProviders(providers map[reflect.Type][]Provider) string

	// RenderObjects 渲染对象图
	RenderObjects(objects map[reflect.Type]map[string][]reflect.Value) string
}

// Graph 依赖关系图
type Graph struct {
	Objects   string `json:"objects"`
	Providers string `json:"providers"`
}
