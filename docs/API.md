# Dix API 文档

## 📚 概述

Dix 提供了三个层次的 API：

1. **公共 API** (`dix` 包) - 推荐的主要 API
2. **全局容器 API** (`dixglobal` 包) - 便捷的全局操作
3. **内部 API** (`dixinternal` 包) - 高级用法和扩展

## 🚀 公共 API (`dix` 包)

### 容器管理

#### `New(opts ...Option) Container`

创建新的依赖注入容器。

```go
import "github.com/pubgo/dix"

// 创建默认容器
container := dix.New()

// 创建带选项的容器
container := dix.New(dix.WithValuesNull())
```

**参数：**
- `opts ...Option` - 可选的配置选项

**返回：**
- `Container` - 容器接口实例

#### `NewWithOptions(opts ...Option) Container`

Creates a new container with configuration options.

```go
container := dix.NewWithOptions(
    dix.WithLogger(logger),
    dix.WithDebug(true),
)
```

### 提供者注册

#### `Provide(providers ...interface{}) error`
Registers provider functions with the container.

**Provider Function Signatures:**
- `func() T` - Simple provider
- `func() (T, error)` - Provider with error handling
- `func(dep1 Dep1, dep2 Dep2) T` - Provider with dependencies
- `func(dep1 Dep1, dep2 Dep2) (T, error)` - Provider with dependencies and error handling

**Supported Types:**
- Pointer types: `*T`
- Interface types: `interface{}`
- Struct types: `struct{}`
- Map types: `map[K]V`
- Slice types: `[]T`
- Function types: `func(...) ...`

```go
// Simple provider
err := container.Provide(func() *Database {
    return &Database{Host: "localhost"}
})

// Provider with error handling
err := container.Provide(func() (*Config, error) {
    config, err := loadConfig()
    if err != nil {
        return nil, fmt.Errorf("failed to load config: %w", err)
    }
    return config, nil
})

// Provider with dependencies
err := container.Provide(func(config *Config) (*Database, error) {
    db, err := sql.Open("postgres", config.DatabaseURL)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }
    return &Database{DB: db}, nil
})

// Interface provider
err := container.Provide(func() Logger {
    return &ConsoleLogger{}
})

// Struct provider
err := container.Provide(func() Config {
    return Config{
        Host: "localhost",
        Port: 8080,
    }
})
```

**Error Handling in Providers:**
When a provider function returns an error as the second return value:
- If the error is `nil`, the first return value is used as the provided instance
- If the error is not `nil`, the provider invocation fails and the error is propagated
- The error will be wrapped with additional context about the provider type and location

### 依赖注入

#### `Inject[T any](container Container, target T, opts ...Option) T`

执行依赖注入到目标对象。

```go
// 结构体注入
type Service struct {
    Logger Logger
    DB     Database
}

var service Service
dix.Inject(container, &service)

// 函数注入
dix.Inject(container, func(logger Logger, db Database) {
    // 使用注入的依赖
    logger.Log("Database connected")
})

// 返回注入后的对象
service := dix.Inject(container, &Service{})
```

**参数：**
- `container Container` - 源容器
- `target T` - 注入目标
- `opts ...Option` - 可选配置

**返回：**
- `T` - 注入后的目标对象

### 实例获取

#### `Get[T any](container Container, opts ...Option) (T, error)`

获取指定类型的实例（带错误处理）。

```go
// 获取单个实例
logger, err := dix.Get[Logger](container)
if err != nil {
    log.Fatal(err)
}

// 获取切片
handlers, err := dix.Get[[]Handler](container)
if err != nil {
    log.Fatal(err)
}

// 获取映射
databases, err := dix.Get[map[string]Database](container)
if err != nil {
    log.Fatal(err)
}
```

**参数：**
- `container Container` - 源容器
- `opts ...Option` - 可选配置

**返回：**
- `T` - 请求的实例
- `error` - 错误信息

#### `MustGet[T any](container Container, opts ...Option) T`

获取指定类型的实例（失败时 panic）。

```go
// 获取实例，失败时 panic
logger := dix.MustGet[Logger](container)
handlers := dix.MustGet[[]Handler](container)
databases := dix.MustGet[map[string]Database](container)
```

**参数：**
- `container Container` - 源容器
- `opts ...Option` - 可选配置

**返回：**
- `T` - 请求的实例

### 图形查看

#### `GetGraph(container Container) *Graph`

获取容器的依赖关系图。

```go
graph := dix.GetGraph(container)
fmt.Printf("Providers: %s\n", graph.Providers)
fmt.Printf("Objects: %s\n", graph.Objects)
```

**参数：**
- `container Container` - 目标容器

**返回：**
- `*Graph` - 依赖关系图

### 配置选项

#### `WithValuesNull() Option`

允许注入 null 值的配置选项。

```go
container := dix.New(dix.WithValuesNull())
```

## 🌍 全局容器 API (`dixglobal` 包)

全局容器提供便捷的单例容器操作，无需手动管理容器实例。

### 提供者注册

#### `Provide(provider any)`

向全局容器注册提供者。

```go
import "github.com/pubgo/dix/dixglobal"

dixglobal.Provide(func() Logger {
    return &ConsoleLogger{}
})

dixglobal.Provide(func(logger Logger) *UserService {
    return &UserService{Logger: logger}
})
```

### 依赖注入

#### `Inject(target any, opts ...Option)`

向目标对象注入依赖。

```go
// 结构体注入
var service UserService
dixglobal.Inject(&service)

// 函数注入
dixglobal.Inject(func(logger Logger) {
    logger.Log("Hello from global container")
})
```

### 实例获取

#### `Get[T any](opts ...Option) T`

从全局容器获取实例。

```go
logger := dixglobal.Get[Logger]()
handlers := dixglobal.Get[[]Handler]()
databases := dixglobal.Get[map[string]Database]()
```

#### `MustGet[T any](opts ...Option) T`

从全局容器获取实例（失败时 panic）。

```go
logger := dixglobal.MustGet[Logger]()
```

### 图形查看

#### `Graph() *Graph`

获取全局容器的依赖关系图。

```go
graph := dixglobal.Graph()
fmt.Println(graph.Providers)
```

## 🔧 内部 API (`dixinternal` 包)

内部 API 提供更底层的控制和扩展能力。

### 容器接口

#### `Container` 接口

```go
type Container interface {
    Provide(provider any) error
    Inject(target any, opts ...Option) error
    Graph() *Graph
}
```

### 提供者接口

#### `Provider` 接口

```go
type Provider interface {
    Type() reflect.Type
    Call(resolver Resolver) (reflect.Value, error)
}
```

### 解析器接口

#### `Resolver` 接口

```go
type Resolver interface {
    Resolve(typ reflect.Type, opts ...Option) (reflect.Value, error)
}
```

### 注入器接口

#### `Injector` 接口

```go
type Injector interface {
    Inject(target any, opts ...Option) error
}
```

### 内部函数

#### `New(opts ...Option) Container`

创建新容器（内部实现）。

```go
container := dixinternal.New()
```

#### `Get[T any](container Container, opts ...Option) (T, error)`

泛型获取函数（内部实现）。

```go
instance, err := dixinternal.Get[Logger](container)
```

#### `MustGet[T any](container Container, opts ...Option) T`

泛型获取函数，失败时 panic（内部实现）。

```go
instance := dixinternal.MustGet[Logger](container)
```

## 📋 类型和结构

### Graph 结构

```go
type Graph struct {
    Providers string // 提供者信息
    Objects   string // 对象信息
}
```

### Option 类型

```go
type Option func(*Options)
```

### Options 结构

```go
type Options struct {
    AllowNullValues bool
    // 其他配置选项...
}
```

## 🎯 使用模式

### 1. 基础依赖注入

```go
// 定义接口
type Logger interface {
    Log(msg string)
}

// 实现
type ConsoleLogger struct{}
func (c *ConsoleLogger) Log(msg string) {
    fmt.Println("LOG:", msg)
}

// 注册
dix.Provide(container, func() Logger {
    return &ConsoleLogger{}
})

// 使用
dix.Inject(container, func(logger Logger) {
    logger.Log("Hello, Dix!")
})
```

### 2. 结构体注入

```go
type UserService struct {
    Logger Logger
    DB     Database
}

// 注册依赖
dix.Provide(container, func() Logger { return &ConsoleLogger{} })
dix.Provide(container, func() Database { return &MySQL{} })

// 注入到结构体
var service UserService
dix.Inject(container, &service)
```

### 3. 集合类型注入

```go
// 注册多个同类型提供者
dix.Provide(container, func() Handler { return &Handler1{} })
dix.Provide(container, func() Handler { return &Handler2{} })

// 注入为切片
dix.Inject(container, func(handlers []Handler) {
    for _, h := range handlers {
        h.Handle()
    }
})
```

### 4. 映射类型注入

```go
// 注册映射提供者
dix.Provide(container, func() map[string]Database {
    return map[string]Database{
        "primary": &MySQL{},
        "cache":   &Redis{},
    }
})

// 注入映射
dix.Inject(container, func(dbs map[string]Database) {
    primary := dbs["primary"]
    cache := dbs["cache"]
})
```

### 5. 方法注入

```go
type Service struct {
    logger Logger
    db     Database
}

// DixInject 前缀的方法会被自动调用
func (s *Service) DixInjectLogger(logger Logger) {
    s.logger = logger
}

func (s *Service) DixInjectDatabase(db Database) {
    s.db = db
}

// 注入
var service Service
dix.Inject(container, &service)
```

## ❌ 错误处理

### 错误类型

- `ErrProviderInvalid` - 无效的提供者函数
- `ErrCircularDependency` - 检测到循环依赖
- `ErrTypeNotFound` - 找不到指定类型的提供者
- `ErrInjectionFailed` - 注入操作失败

### 错误示例

```go
// 处理获取错误
logger, err := dix.Get[Logger](container)
if err != nil {
    switch {
    case errors.Is(err, dixinternal.ErrTypeNotFound):
        log.Println("Logger not registered")
    case errors.Is(err, dixinternal.ErrCircularDependency):
        log.Println("Circular dependency detected")
    default:
        log.Printf("Injection failed: %v", err)
    }
}
```

## 🔄 最佳实践

### 1. 接口优先

```go
// 好的做法：依赖接口
type UserService struct {
    Logger Logger    // 接口
    DB     Database  // 接口
}

// 避免：依赖具体实现
type UserService struct {
    Logger *ConsoleLogger // 具体实现
    DB     *MySQL         // 具体实现
}
```

### 2. 提供者函数设计

```go
// 好的做法：简单的提供者函数
dix.Provide(container, func() Logger {
    return &ConsoleLogger{}
})

// 好的做法：带依赖的提供者函数
dix.Provide(container, func(config Config) Database {
    return &MySQL{
        Host: config.Database.Host,
        Port: config.Database.Port,
    }
})

// 避免：复杂的提供者函数
dix.Provide(container, func() Logger {
    // 大量初始化逻辑...
    // 应该拆分为多个提供者
})
```

### 3. 错误处理

```go
// 好的做法：处理错误
logger, err := dix.Get[Logger](container)
if err != nil {
    return fmt.Errorf("failed to get logger: %w", err)
}

// 或者使用 MustGet（确保不会失败的场景）
logger := dix.MustGet[Logger](container)
```

### 4. 容器生命周期

```go
// 好的做法：在应用启动时注册所有提供者
func setupContainer() Container {
    container := dix.New()
    
    // 注册所有提供者
    dix.Provide(container, newLogger)
    dix.Provide(container, newDatabase)
    dix.Provide(container, newUserService)
    
    return container
}

// 在应用运行时使用
func main() {
    container := setupContainer()
    
    // 使用容器...
}
```

---

这个 API 文档提供了 Dix 框架的完整 API 参考，包括使用示例、最佳实践和错误处理指南。 