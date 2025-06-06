# Dix 迁移指南

## 🎯 概述

本指南帮助您从 Dix v1.x 迁移到 v2.0。新版本引入了现代化的模块化架构和泛型支持，同时保持了核心功能的向后兼容性。

## 📊 主要变化

### 架构变化

| 方面 | v1.x | v2.0 |
|------|------|------|
| **架构** | 单体设计 | 模块化设计 |
| **类型安全** | 运行时检查 | 编译时泛型 |
| **API 设计** | 方法链式 | 函数式 |
| **错误处理** | 简单错误 | 结构化错误 |
| **性能** | 基础优化 | 高度优化 |

### API 变化

| 操作 | v1.x API | v2.0 API |
|------|----------|----------|
| **容器创建** | `dix.NewDix()` | `dix.New()` |
| **提供者注册** | `container.Provide(fn)` | `dix.Provide(container, fn)` |
| **依赖注入** | `container.Inject(target)` | `dix.Inject(container, target)` |
| **实例获取** | `container.Get(target)` | `dix.Get[T](container)` |
| **图形查看** | `container.Graph()` | `dix.GetGraph(container)` |

## 🔄 迁移步骤

### 步骤 1：更新导入

**v1.x:**
```go
import "github.com/pubgo/dix"
```

**v2.0:**
```go
import (
    "github.com/pubgo/dix"
    "github.com/pubgo/dix/dixglobal" // 可选：全局容器
)
```

### 步骤 2：容器创建

**v1.x:**
```go
container := dix.NewDix()
```

**v2.0:**
```go
container := dix.New()

// 或使用全局容器（推荐简单场景）
// 无需创建容器，直接使用 dixglobal
```

### 步骤 3：提供者注册

**v1.x:**
```go
container.Provide(func() Logger {
    return &ConsoleLogger{}
})

container.Provide(func(logger Logger) *UserService {
    return &UserService{Logger: logger}
})
```

**v2.0:**
```go
// 使用容器
dix.Provide(container, func() Logger {
    return &ConsoleLogger{}
})

dix.Provide(container, func(logger Logger) *UserService {
    return &UserService{Logger: logger}
})

// 或使用全局容器
dixglobal.Provide(func() Logger {
    return &ConsoleLogger{}
})

dixglobal.Provide(func(logger Logger) *UserService {
    return &UserService{Logger: logger}
})
```

### 步骤 4：依赖注入

**v1.x:**
```go
// 结构体注入
var service UserService
container.Inject(&service)

// 函数注入
container.Inject(func(logger Logger) {
    logger.Log("Hello")
})
```

**v2.0:**
```go
// 使用容器
var service UserService
dix.Inject(container, &service)

dix.Inject(container, func(logger Logger) {
    logger.Log("Hello")
})

// 或使用全局容器
var service UserService
dixglobal.Inject(&service)

dixglobal.Inject(func(logger Logger) {
    logger.Log("Hello")
})
```

### 步骤 5：实例获取

**v1.x:**
```go
var logger Logger
err := container.Get(&logger)
if err != nil {
    log.Fatal(err)
}
```

**v2.0:**
```go
// 使用泛型 API（推荐）
logger, err := dix.Get[Logger](container)
if err != nil {
    log.Fatal(err)
}

// 或者使用 MustGet（确保不会失败时）
logger := dix.MustGet[Logger](container)

// 或使用全局容器
logger := dixglobal.Get[Logger]()
```

### 步骤 6：图形查看

**v1.x:**
```go
graph := container.Graph()
fmt.Println(graph)
```

**v2.0:**
```go
// 使用容器
graph := dix.GetGraph(container)
fmt.Printf("Providers: %s\n", graph.Providers)
fmt.Printf("Objects: %s\n", graph.Objects)

// 或使用全局容器
graph := dixglobal.Graph()
fmt.Printf("Providers: %s\n", graph.Providers)
```

## 📝 完整迁移示例

### v1.x 代码

```go
package main

import (
    "fmt"
    "github.com/pubgo/dix"
)

type Logger interface {
    Log(msg string)
}

type ConsoleLogger struct{}

func (c *ConsoleLogger) Log(msg string) {
    fmt.Println("LOG:", msg)
}

type UserService struct {
    Logger Logger
}

func main() {
    // 创建容器
    container := dix.NewDix()
    
    // 注册提供者
    container.Provide(func() Logger {
        return &ConsoleLogger{}
    })
    
    container.Provide(func(logger Logger) *UserService {
        return &UserService{Logger: logger}
    })
    
    // 依赖注入
    var service UserService
    err := container.Inject(&service)
    if err != nil {
        panic(err)
    }
    
    // 使用服务
    service.Logger.Log("Hello, Dix v1!")
    
    // 查看图形
    fmt.Println(container.Graph())
}
```

### v2.0 代码（容器方式）

```go
package main

import (
    "fmt"
    "github.com/pubgo/dix"
)

type Logger interface {
    Log(msg string)
}

type ConsoleLogger struct{}

func (c *ConsoleLogger) Log(msg string) {
    fmt.Println("LOG:", msg)
}

type UserService struct {
    Logger Logger
}

func main() {
    // 创建容器
    container := dix.New()
    
    // 注册提供者
    dix.Provide(container, func() Logger {
        return &ConsoleLogger{}
    })
    
    dix.Provide(container, func(logger Logger) *UserService {
        return &UserService{Logger: logger}
    })
    
    // 依赖注入
    var service UserService
    dix.Inject(container, &service)
    
    // 使用服务
    service.Logger.Log("Hello, Dix v2!")
    
    // 或者使用泛型获取
    userService := dix.MustGet[*UserService](container)
    userService.Logger.Log("Hello from generic API!")
    
    // 查看图形
    graph := dix.GetGraph(container)
    fmt.Printf("Providers: %s\n", graph.Providers)
}
```

### v2.0 代码（全局容器方式）

```go
package main

import (
    "fmt"
    "github.com/pubgo/dix/dixglobal"
)

type Logger interface {
    Log(msg string)
}

type ConsoleLogger struct{}

func (c *ConsoleLogger) Log(msg string) {
    fmt.Println("LOG:", msg)
}

type UserService struct {
    Logger Logger
}

func main() {
    // 注册提供者到全局容器
    dixglobal.Provide(func() Logger {
        return &ConsoleLogger{}
    })
    
    dixglobal.Provide(func(logger Logger) *UserService {
        return &UserService{Logger: logger}
    })
    
    // 依赖注入
    var service UserService
    dixglobal.Inject(&service)
    
    // 使用服务
    service.Logger.Log("Hello, Dix v2 Global!")
    
    // 或者使用泛型获取
    userService := dixglobal.Get[*UserService]()
    userService.Logger.Log("Hello from global generic API!")
    
    // 查看图形
    graph := dixglobal.Graph()
    fmt.Printf("Providers: %s\n", graph.Providers)
}
```

## 🔧 常见迁移问题

### 1. 编译错误：方法不存在

**问题：**
```go
// v1.x 代码
container.Provide(provider) // 编译错误
```

**解决方案：**
```go
// v2.0 代码
dix.Provide(container, provider)
```

### 2. 类型获取方式变化

**问题：**
```go
// v1.x 代码
var logger Logger
err := container.Get(&logger)
```

**解决方案：**
```go
// v2.0 代码
logger, err := dix.Get[Logger](container)
// 或
logger := dix.MustGet[Logger](container)
```

### 3. 错误处理变化

**问题：**
```go
// v1.x 代码
err := container.Inject(target)
if err != nil {
    // 处理错误
}
```

**解决方案：**
```go
// v2.0 代码 - 自动错误处理
dix.Inject(container, target) // 内部使用 assert.Must

// 或者手动错误处理
if err := container.Inject(target); err != nil {
    // 处理错误
}
```

### 4. 图形输出格式变化

**问题：**
```go
// v1.x 代码
fmt.Println(container.Graph()) // 简单字符串
```

**解决方案：**
```go
// v2.0 代码
graph := dix.GetGraph(container)
fmt.Printf("Providers: %s\n", graph.Providers)
fmt.Printf("Objects: %s\n", graph.Objects)
```

## 🚀 利用新特性

### 1. 泛型 API

```go
// 类型安全的实例获取
logger := dix.MustGet[Logger](container)
handlers := dix.MustGet[[]Handler](container)
databases := dix.MustGet[map[string]Database](container)
```

### 2. 全局容器

```go
// 简化的全局操作
dixglobal.Provide(newLogger)
dixglobal.Provide(newUserService)

service := dixglobal.Get[*UserService]()
```

### 3. 增强的错误处理

```go
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

## 📋 迁移检查清单

### 基础迁移

- [ ] 更新导入语句
- [ ] 替换 `dix.NewDix()` 为 `dix.New()`
- [ ] 替换 `container.Provide()` 为 `dix.Provide(container, ...)`
- [ ] 替换 `container.Inject()` 为 `dix.Inject(container, ...)`
- [ ] 替换 `container.Get()` 为 `dix.Get[T](container)`
- [ ] 替换 `container.Graph()` 为 `dix.GetGraph(container)`

### 优化迁移

- [ ] 考虑使用全局容器简化代码
- [ ] 利用泛型 API 提高类型安全
- [ ] 更新错误处理逻辑
- [ ] 优化提供者函数设计
- [ ] 添加适当的配置选项

### 测试验证

- [ ] 运行现有测试确保功能正常
- [ ] 添加新的泛型 API 测试
- [ ] 验证错误处理行为
- [ ] 检查性能是否有改善
- [ ] 确认依赖图输出正确

## 🔄 渐进式迁移策略

### 阶段 1：基础兼容

1. 更新到 v2.0
2. 使用类型别名保持兼容：`type Dix = Container`
3. 最小化代码变更
4. 验证功能正常

### 阶段 2：API 现代化

1. 逐步替换旧 API 调用
2. 引入泛型 API
3. 优化错误处理
4. 更新测试代码

### 阶段 3：架构优化

1. 考虑使用全局容器
2. 重构提供者函数
3. 利用新的配置选项
4. 性能优化和监控

## 💡 最佳实践

### 1. 选择合适的 API 层次

```go
// 简单应用：使用全局容器
dixglobal.Provide(provider)
dixglobal.Inject(target)

// 复杂应用：使用容器实例
container := dix.New()
dix.Provide(container, provider)
dix.Inject(container, target)

// 库开发：使用内部 API
container := dixinternal.New()
```

### 2. 错误处理策略

```go
// 应用启动阶段：使用 MustGet
logger := dix.MustGet[Logger](container)

// 运行时阶段：处理错误
logger, err := dix.Get[Logger](container)
if err != nil {
    return fmt.Errorf("failed to get logger: %w", err)
}
```

### 3. 性能优化

```go
// 预先获取常用依赖
logger := dix.MustGet[Logger](container)
db := dix.MustGet[Database](container)

// 避免重复解析
handlers := dix.MustGet[[]Handler](container)
for _, handler := range handlers {
    // 使用 handler
}
```

---

通过遵循这个迁移指南，您可以顺利地从 Dix v1.x 迁移到 v2.0，并充分利用新版本的现代化特性和性能改进。 