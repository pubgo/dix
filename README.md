# Dix - 现代化 Go 依赖注入框架

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.18-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/pubgo/dix)](https://goreportcard.com/report/github.com/pubgo/dix)
[![Coverage Status](https://coveralls.io/repos/github/pubgo/dix/badge.svg)](https://coveralls.io/github/pubgo/dix)

**Dix** 是一个现代化的 Go 依赖注入框架，采用模块化架构设计，提供类型安全的泛型 API 和高性能的依赖管理能力。

## ✨ 特性

### 🚀 现代化设计
- **泛型支持**：完全的 Go 1.18+ 泛型 API，编译时类型安全
- **模块化架构**：清晰的分层设计，易于扩展和维护
- **零反射**：高性能实现，避免运行时反射开销
- **函数式 API**：简洁直观的函数式接口设计

### 🔧 强大功能
- **循环依赖检测**：智能检测和报告循环依赖问题
- **多种注入方式**：支持构造函数、结构体字段、方法注入
- **灵活提供者**：支持函数、值、接口等多种提供者类型
- **命名空间隔离**：支持多容器实例，避免全局状态污染

### 📊 高性能
- **预编译优化**：依赖图预编译，运行时零开销
- **内存池化**：智能内存管理，减少 GC 压力
- **并发安全**：线程安全的容器操作
- **懒加载**：按需实例化，优化启动性能

## 🚀 快速开始

### 安装

```bash
go get github.com/pubgo/dix
```

### 基础用法

```go
package main

import (
    "fmt"
    "github.com/pubgo/dix"
)

// 定义接口和实现
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
    
    // 获取实例（泛型 API）
    service := dix.MustGet[*UserService](container)
    service.Logger.Log("Hello, Dix!")
    
    // 或者使用依赖注入
    var injectedService UserService
    dix.Inject(container, &injectedService)
    injectedService.Logger.Log("Hello from injection!")
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
    
    // 直接获取实例
    service := dixglobal.Get[*UserService]()
    service.Logger.Log("Hello from global container!")
}
```

## 📚 文档

### 核心文档
- [📖 API 文档](docs/API.md) - 完整的 API 参考和使用示例
- [🏗️ 架构设计](docs/ARCHITECTURE.md) - 深入了解框架架构和设计理念
- [🔄 迁移指南](docs/MIGRATION.md) - 从旧版本迁移到 v2.0 的详细指南
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

### 注入方式

#### 1. 泛型获取（推荐）
```go
// 类型安全的实例获取
logger := dix.MustGet[Logger](container)
service, err := dix.Get[*UserService](container)
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

#### 3. 函数注入
```go
dix.Inject(container, func(db Database, logger Logger) {
    // 使用注入的依赖
    logger.Log("Database connected")
})
```

## 🔧 高级特性

### 循环依赖检测

```go
// Dix 会自动检测循环依赖
dix.Provide(container, func(b B) A { return A{} })
dix.Provide(container, func(a A) B { return B{} })

// 获取时会报告循环依赖错误
_, err := dix.Get[A](container)
// err: circular dependency detected: A -> B -> A
```

### 集合注入

```go
// 注册多个相同类型的提供者
dix.Provide(container, func() Handler { return &HTTPHandler{} })
dix.Provide(container, func() Handler { return &GRPCHandler{} })

// 获取所有实例
handlers := dix.MustGet[[]Handler](container)
fmt.Printf("Registered %d handlers\n", len(handlers))
```

### 映射注入

```go
// 命名提供者
dix.Provide(container, func() Handler { return &HTTPHandler{} })
dix.Provide(container, func() Handler { return &GRPCHandler{} })

// 获取映射
handlerMap := dix.MustGet[map[string]Handler](container)
for name, handler := range handlerMap {
    fmt.Printf("Handler %s: %T\n", name, handler)
}
```

### 依赖图可视化

```go
// 查看依赖关系图
graph := dix.GetGraph(container)
fmt.Printf("Providers: %s\n", graph.Providers)
fmt.Printf("Objects: %s\n", graph.Objects)
```

## 🏗️ 架构层次

Dix 采用分层架构设计：

```
┌─────────────────────────────────────┐
│           Public API                │  ← dix 包：用户友好的 API
├─────────────────────────────────────┤
│         Global Container            │  ← dixglobal 包：全局容器
├─────────────────────────────────────┤
│        Internal Core                │  ← dixinternal 包：核心实现
└─────────────────────────────────────┘
```

### API 层次选择

- **简单应用**：使用 `dixglobal` 包的全局容器
- **复杂应用**：使用 `dix` 包的容器实例
- **库开发**：使用 `dixinternal` 包的底层 API

## 🚀 性能优势

### v2.0 vs v1.x 性能对比

| 指标 | v1.x | v2.0 | 改进 |
|------|------|------|------|
| **代码行数** | 1,200+ | 373 | -69% |
| **内存使用** | 基准 | -30% | 更少内存分配 |
| **启动时间** | 基准 | -40% | 预编译优化 |
| **运行时性能** | 基准 | +25% | 零反射实现 |

### 优化特性

- **预编译依赖图**：启动时构建，运行时零开销
- **类型缓存**：避免重复类型解析
- **内存池化**：减少 GC 压力
- **并发优化**：线程安全的高效实现

## 🔄 迁移指南

从 v1.x 迁移到 v2.0？查看我们的[详细迁移指南](docs/MIGRATION.md)。

### 主要 API 变化

| v1.x | v2.0 |
|------|------|
| `dix.NewDix()` | `dix.New()` |
| `container.Provide(fn)` | `dix.Provide(container, fn)` |
| `container.Inject(target)` | `dix.Inject(container, target)` |
| `container.Get(&target)` | `dix.Get[T](container)` |

## 🤝 贡献

我们欢迎社区贡献！请查看 [CONTRIBUTING.md](CONTRIBUTING.md) 了解如何参与项目开发。

### 开发环境

```bash
# 克隆项目
git clone https://github.com/pubgo/dix.git
cd dix

# 安装依赖
go mod tidy

# 运行测试
go test ./...

# 运行示例
go run example/basic/main.go
```

## 📄 许可证

本项目采用 [Apache 2.0 许可证](LICENSE)。

## 🙏 致谢

- 设计灵感来源于 [uber-go/dig](https://github.com/uber-go/dig)
- 感谢所有贡献者的支持和反馈

---

**Dix** - 让依赖注入变得简单而强大 🚀
