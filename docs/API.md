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

#### `Provide(container Container, provider any)`

注册依赖提供者到容器。

**支持的提供者函数签名：**
- `func() T` - 简单提供者
- `func() (T, error)` - 带错误处理的提供者
- `func(dep1 Dep1, dep2 Dep2) T` - 带依赖的提供者
- `func(dep1 Dep1, dep2 Dep2) (T, error)` - 带依赖和错误处理的提供者

**支持的类型：**
- 指针类型：`*T`
- 接口类型：`interface{}`
- 结构体类型：`struct{}`
- Map类型：`map[K]V`
- Slice类型：`[]T`
- 函数类型：`func(...) ...`

```go
// 简单提供者
dix.Provide(container, func() *Database {
    return &Database{Host: "localhost"}
})

// 带错误处理的提供者
dix.Provide(container, func() (*Config, error) {
    config, err := loadConfig()
    if err != nil {
        return nil, fmt.Errorf("failed to load config: %w", err)
    }
    return config, nil
})

// 带依赖的提供者
dix.Provide(container, func(config *Config) (*Database, error) {
    db, err := sql.Open("postgres", config.DatabaseURL)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }
    return &Database{DB: db}, nil
})

// Interface provider
dix.Provide(container, func() Logger {
    return &ConsoleLogger{}
})

// Struct provider
dix.Provide(container, func() Config {
    return Config{
        Host: "localhost",
        Port: 8080,
    }
})
```

**错误处理：**
当提供者函数返回错误作为第二个返回值时：
- 如果错误为 `nil`，第一个返回值将被用作提供的实例
- 如果错误不为 `nil`，提供者调用失败，错误会被传播
- 错误会被包装并包含提供者类型和位置信息以便调试

### 统一的依赖注入

#### `Inject(container Container, target interface{}, opts ...Option) error`

统一的依赖注入方法，这是 Dix 的核心方法。

**核心设计理念：**
`Inject` 方法支持多种输入类型，既可以进行依赖注入，也可以**获取依赖实例**，提供统一的 API 体验。

**支持的目标类型：**

1. **函数注入** - 解析函数参数并调用函数
2. **结构体注入** - 注入到结构体字段
3. **接口注入** - 支持接口类型注入
4. **集合注入** - 切片和映射类型注入

```go
// 1. 函数注入 - 直接使用依赖
dix.Inject(container, func(logger Logger, db *Database) {
    logger.Log("Database connected")
    // 直接使用注入的依赖
})

// 2. 结构体注入
type Service struct {
    Logger Logger
    DB     *Database
}
var service Service
dix.Inject(container, &service)

// 3. 方法注入（DixInject前缀方法会被自动调用）
type UserService struct {
    logger Logger
    db     *Database
}
func (s *UserService) DixInjectLogger(logger Logger) { s.logger = logger }
func (s *UserService) DixInjectDatabase(db *Database) { s.db = db }

var userService UserService
dix.Inject(container, &userService)

// 4. 获取依赖实例的用法
var logger Logger
var db *Database
dix.Inject(container, func(l Logger, d *Database) {
    logger = l   // 获取 Logger 实例
    db = d       // 获取 Database 实例
})

// 5. 批量获取多个依赖
var logger Logger
var database *Database
var handlers []Handler
var configMap map[string]*Config
dix.Inject(container, func(l Logger, db *Database, h []Handler, cm map[string]*Config) {
    logger = l
    database = db
    handlers = h
    configMap = cm
})
```

**参数：**
- `container Container` - 源容器
- `target interface{}` - 注入目标（函数、结构体指针、接口等）
- `opts ...Option` - 可选配置

**返回：**
- `error` - 注入失败时的错误信息

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

### 统一的依赖注入

#### `Inject[T any](target T, opts ...Option) T`

向目标对象注入依赖，支持所有类型的注入模式。

```go
// 结构体注入
type Service struct {
    Logger Logger
    DB     *Database
}
service := dixglobal.Inject(&Service{})

// 函数注入
dixglobal.Inject(func(logger Logger) {
    logger.Log("Hello from global container")
})

// 获取依赖实例的用法
var logger Logger
var database *Database
dixglobal.Inject(func(l Logger, db *Database) {
    logger = l
    database = db
})

// 批量获取依赖
var service *UserService
var handlers []Handler
var configMap map[string]*Config
dixglobal.Inject(func(s *UserService, h []Handler, cm map[string]*Config) {
    service = s
    handlers = h
    configMap = cm
})
```

**参数：**
- `target T` - 注入目标
- `opts ...Option` - 可选配置

**返回：**
- `T` - 注入后的目标对象

### 图形查看

#### `Graph() *Graph`

获取全局容器的依赖关系图。

```go
graph := dixglobal.Graph()
fmt.Printf("Providers: %s\n", graph.Providers)
fmt.Printf("Objects: %s\n", graph.Objects)
```

## 🎯 高级用法

### 1. 接口注入

```go
type Logger interface {
    Log(msg string)
}

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

## 📊 统一 API 的优势

### 传统方式 vs Dix 方式

| 功能 | 传统方式 | Dix 统一方式 |
|------|---------|-------------|
| **获取单个实例** | `instance, err := container.Get(reflect.TypeOf((*Logger)(nil)).Elem())` | `var logger Logger; container.Inject(func(l Logger) { logger = l })` |
| **获取多个实例** | `logger, _ := container.Get(...)`<br>`db, _ := container.Get(...)` | `var logger Logger; var db *DB; container.Inject(func(l Logger, d *DB) { logger, db = l, d })` |
| **结构体注入** | `container.Inject(&target)` | `dix.Inject(container, &target)` |
| **函数调用** | `container.Call(fn)` | `dix.Inject(container, fn)` |

### 设计优势

1. **API 统一性** - 一个方法处理所有依赖注入需求
2. **类型安全** - 编译时类型检查，避免类型断言错误
3. **学习成本低** - 只需掌握一个方法的用法
4. **功能强大** - 支持复杂的依赖注入场景
5. **代码简洁** - 减少样板代码，提高开发效率

## ❌ 错误处理

### 错误类型

- `ErrProviderInvalid` - 无效的提供者函数
- `ErrCircularDependency` - 检测到循环依赖
- `ErrTypeNotFound` - 找不到指定类型的提供者
- `ErrInjectionFailed` - 注入操作失败

### 错误示例

```go
// 处理注入错误
err := dix.Inject(container, func(logger Logger) {
    logger.Log("Hello")
})
if err != nil {
    if errors.Is(err, dix.ErrTypeNotFound) {
        fmt.Println("Logger not registered")
    }
}

// 处理提供者错误
dix.Provide(container, func() (*Database, error) {
    return nil, errors.New("connection failed")
})

// 注入时会传播提供者错误
err = dix.Inject(container, func(db *Database) {
    // 这里会收到 "connection failed" 错误
})
```

## 🔧 最佳实践

### 1. 优先使用函数注入

```go
// 推荐：函数注入，直接使用依赖
dix.Inject(container, func(logger Logger, db *Database) {
    logger.Log("Starting application")
    // 直接使用依赖，无需额外变量
})

// 可选：当需要在函数外使用时
var logger Logger
dix.Inject(container, func(l Logger) { logger = l })
```

### 2. 合理使用全局容器

```go
// 简单应用：使用全局容器
dixglobal.Provide(func() Logger { return &ConsoleLogger{} })
dixglobal.Inject(func(logger Logger) {
    logger.Log("Simple and clean")
})

// 复杂应用：使用容器实例以避免全局状态
container := dix.New()
dix.Provide(container, func() Logger { return &ConsoleLogger{} })
```

### 3. 错误处理策略

```go
// 提供者中的错误处理
dix.Provide(container, func() (*Database, error) {
    db, err := connectToDatabase()
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }
    return db, nil
})

// 注入时的错误处理
if err := dix.Inject(container, myHandler); err != nil {
    log.Fatalf("Dependency injection failed: %v", err)
}
```

### 4. 性能优化

```go
// 延迟初始化重量级依赖
dix.Provide(container, func() *HeavyService {
    // 只有在需要时才会创建
    return NewHeavyService()
})

// 单例模式（默认行为）
dix.Provide(container, func() *Singleton {
    return &Singleton{} // 只会创建一次
})
```

## 📈 迁移指南

从传统 Get 方式迁移到统一 Inject 方式：

```go
// 旧方式
logger, err := container.Get(reflect.TypeOf((*Logger)(nil)).Elem())
if err != nil {
    return err
}
db, err := container.Get(reflect.TypeOf((*Database)(nil)).Elem())
if err != nil {
    return err
}

// 新方式
var logger Logger
var db *Database
err := dix.Inject(container, func(l Logger, d *Database) {
    logger = l
    db = d
})
if err != nil {
    return err
}
```

这种统一的设计大大简化了 API 的使用，提高了代码的可读性和维护性。 