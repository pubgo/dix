[![Go Doc](https://godoc.org/github.com/pubgo/dix?status.svg)](https://godoc.org/github.com/pubgo/dix)
[![Build Status](https://travis-ci.com/pubgo/dix.svg?branch=master)](https://travis-ci.com/pubgo/dix)
[![Go Report Card](https://goreportcard.com/badge/github.com/pubgo/dix)](https://goreportcard.com/report/github.com/pubgo/dix)

# Dix - 现代化的 Go 依赖注入框架

## 🎯 核心设计理念

Dix 采用**统一的 Inject 方法设计**，通过单一接口支持多种依赖注入模式：

- ✅ **函数注入** - 解析函数参数并调用
- ✅ **结构体注入** - 注入到结构体字段  
- ✅ **方法注入** - 自动调用 DixInject 前缀方法
- ✅ **获取依赖实例** - 通过函数参数获取依赖实例

> **设计优势**: `Inject` 方法的入参可以是函数、指针、接口、map、list 等，**一个方法涵盖所有依赖注入需求**，提供更加统一和灵活的 API。

## 🚀 快速开始

### 安装

```bash
go get github.com/pubgo/dix
```

### 基本用法

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
    
    // 方式1: 结构体注入
    var service UserService
    dix.Inject(container, &service)
    service.Logger.Log("Hello, Dix!")
    
    // 方式2: 函数注入
    dix.Inject(container, func(service *UserService) {
        service.Logger.Log("Hello from function injection!")
    })
    
    // 方式3: 获取依赖实例的用法
    var logger Logger
    var userService *UserService
    dix.Inject(container, func(l Logger, us *UserService) {
        logger = l
        userService = us
    })
    logger.Log("Hello from unified injection!")
}
```

### 全局容器用法

```go
package main

import (
    "github.com/pubgo/dix/dixglobal"
)

func main() {
    // 使用全局容器，更简洁
    dixglobal.Provide(func() Logger {
        return &ConsoleLogger{}
    })
    
    dixglobal.Provide(func(logger Logger) *UserService {
        return &UserService{Logger: logger}
    })
    
    // 统一的注入方式
    dixglobal.Inject(func(service *UserService) {
        service.Logger.Log("Hello from global container!")
    })
    
    // 获取依赖实例的用法
    var service *UserService
    dixglobal.Inject(func(s *UserService) {
        service = s
    })
    service.Logger.Log("Got service via injection!")
}
```

## 📚 文档

### 核心文档
- [📖 API 文档](docs/API.md) - 完整的 API 参考和使用示例
- [🏗️ 架构设计](docs/ARCHITECTURE.md) - 深入了解框架架构和设计理念
- [🔄 迁移指南](docs/MIGRATION.md) - 从旧版本迁移的详细指南
- [📋 更新日志](docs/CHANGELOG.md) - 版本更新历史和变更记录

### 示例代码
- [基础示例](example/) - 各种使用场景的完整示例
- [循环依赖处理](example/cycle/) - 循环依赖检测和处理
- [列表注入](example/list/) - 集合类型的依赖注入
- [方法注入](example/inject_method/) - 方法级别的依赖注入
- [结构体输出](example/struct-out/) - 复杂结构体的依赖管理

## 🎯 核心概念

### 容器 (Container)
容器是依赖管理的核心，负责存储提供者和解析依赖关系：

```go
// 创建新容器
container := dix.New()

// 或使用全局容器
dixglobal.Provide(provider)
```

### 提供者 (Provider)
提供者定义如何创建和配置依赖项：

```go
// 函数提供者
dix.Provide(container, func() Database {
    return &PostgresDB{Host: "localhost"}
})

// 带依赖的提供者
dix.Provide(container, func(db Database, logger Logger) *UserService {
    return &UserService{DB: db, Logger: logger}
})

// 值提供者
dix.Provide(container, &Config{Port: 8080})
```

### 统一的注入方式

#### 1. 函数注入（推荐）
```go
// 直接使用依赖
dix.Inject(container, func(db Database, logger Logger) {
    // 使用注入的依赖
    logger.Log("Database connected")
})

// 获取依赖实例的用法
var logger Logger
var service *UserService
dix.Inject(container, func(l Logger, s *UserService) {
    logger = l    // 获取 Logger 实例
    service = s   // 获取 UserService 实例
})
```

#### 2. 结构体注入
```go
type Handler struct {
    DB     Database `dix:""`
    Logger Logger   `dix:""`
}

var handler Handler
dix.Inject(container, &handler)
```

#### 3. 方法注入
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

var service Service
dix.Inject(container, &service)
```

## 🔧 高级特性

### 循环依赖检测

```go
// Dix 会自动检测循环依赖
dix.Provide(container, func(b B) A { return A{} })
dix.Provide(container, func(a A) B { return B{} })

// 注入时会报告循环依赖错误
err := dix.Inject(container, func(a A) {
    // 这里会触发循环依赖错误
})
// err: circular dependency detected: A -> B -> A
```

### 集合注入

```go
// 注册多个相同类型的提供者
dix.Provide(container, func() Handler { return &HTTPHandler{} })
dix.Provide(container, func() Handler { return &GRPCHandler{} })

// 获取所有实例
dix.Inject(container, func(handlers []Handler) {
    fmt.Printf("Registered %d handlers\n", len(handlers))
    for i, handler := range handlers {
        fmt.Printf("Handler %d: %T\n", i, handler)
    }
})
```

### 映射注入

```go
// 命名提供者
dix.Provide(container, func() Handler { return &HTTPHandler{} })
dix.Provide(container, func() Handler { return &GRPCHandler{} })

// 获取映射
dix.Inject(container, func(handlerMap map[string]Handler) {
    for name, handler := range handlerMap {
        fmt.Printf("Handler %s: %T\n", name, handler)
    }
})
```

### 依赖图可视化

```go
// 查看依赖关系图
graph := dix.GetGraph(container)
fmt.Printf("Providers:\n%s\n", graph.Providers)
fmt.Printf("Objects:\n%s\n", graph.Objects)
```

## 📋 API 对比

### 统一设计的优势

| 传统方式 | Dix 统一方式 |
|---------|-------------|
| `container.Get(&target)` | `dix.Inject(container, func(t Target) { target = t })` |
| `container.Inject(target)` | `dix.Inject(container, target)` |
| `container.Call(fn)` | `dix.Inject(container, fn)` |

**统一的 Inject 方法支持:**
- ✅ 函数：`func(deps...) { ... }`
- ✅ 结构体指针：`&struct{}`
- ✅ 接口类型：`interface{}`
- ✅ 切片类型：`[]T`
- ✅ 映射类型：`map[string]T`

## 🌟 特性亮点

- **🎯 统一 API**: 一个 `Inject` 方法处理所有依赖注入场景
- **🔒 类型安全**: 编译时类型检查，运行时错误详细
- **⚡ 高性能**: 优化的依赖解析和缓存机制
- **🔍 循环检测**: 自动检测和报告循环依赖
- **📊 可视化**: 依赖关系图生成和分析
- **🧩 模块化**: 清晰的架构分层和组件解耦
- **🛡️ 错误友好**: 详细的错误信息和调试支持

## 💡 设计思路

Dix 的核心设计理念是**简化和统一**：

1. **统一接口**: `Inject` 方法可以处理所有类型的依赖注入需求
2. **类型灵活**: 支持函数、指针、接口、集合等多种类型
3. **功能全面**: 既能注入依赖，也能获取实例，满足所有需求
4. **使用简单**: 学习成本低，API 直观易懂

这种设计让开发者只需要掌握一个方法，就能处理所有的依赖注入场景，大大简化了框架的使用复杂度。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## �� 许可证

MIT License
