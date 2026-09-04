# dix

[![Go Reference](https://pkg.go.dev/badge/github.com/pubgo/dix/v2.svg)](https://pkg.go.dev/github.com/pubgo/dix/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/pubgo/dix)](https://goreportcard.com/report/github.com/pubgo/dix)

> **dix** 是一个轻量、强大的 Go 依赖注入框架

参考了 [uber-go/dig](https://github.com/uber-go/dig) 的设计，支持更复杂的依赖管理和 namespace 隔离。

[English](./README.md)

## 目录

- [适用场景](#适用场景)
- [功能特性](#-功能特性)
- [安装](#-安装)
- [快速开始](#-快速开始)
- [核心 API](#-核心-api)
- [注入模式](#-注入模式)
- [模块](#-模块)
- [诊断与排障](#-诊断与排障)
- [开发](#️-开发)
- [示例](#-示例)
- [文档](#-文档)

## 适用场景

- 需要**运行时**注册依赖（插件、动态模块、按条件装配）。
- 希望内置**诊断能力**：结构化 trace 日志、JSONL 导出、HTTP 依赖图可视化。
- 偏好 **dig 风格 API**，并需要 `Try*` 安全接口、map/list 分组、方法注入等能力。

若追求编译期装配、最小运行时开销，可参考 [google/wire](https://github.com/google/wire)。若使用 Uber fx 生态，可参考 [uber-go/dig](https://github.com/uber-go/dig)。

## ✨ 功能特性

| 特性           | 说明                                              |
| -------------- | ------------------------------------------------- |
| 🔄 **循环检测** | 自动检测依赖循环，避免死循环                      |
| 📦 **多种注入** | 支持 func、struct、map、list 作为注入参数         |
| 🏷️ **命名空间** | 通过 map key 实现依赖隔离                         |
| 🎯 **多输出**   | struct 可对外提供多组依赖对象                     |
| 🪆 **嵌套支持** | 支持 struct 依赖嵌套                              |
| 🔧 **无侵入**   | 对原对象零侵入                                    |
| 🛡️ **安全 API** | 提供 `TryProvide`/`TryInject` 不 panic 的安全版本 |
| 🌐 **可视化**   | HTTP 模块图形化展示依赖关系                       |

## 📦 安装

```bash
go get github.com/pubgo/dix/v2
```

## 🚀 快速开始

```go
package main

import (
    "fmt"
    "github.com/pubgo/dix/v2"
)

type Config struct {
    DSN string
}

type Database struct {
    Config *Config
}

type UserService struct {
    DB *Database
}

func main() {
    // 创建容器
    di := dix.New()

    // 注册 Provider
    dix.Provide(di, func() *Config {
        return &Config{DSN: "postgres://localhost/mydb"}
    })

    dix.Provide(di, func(c *Config) *Database {
        return &Database{Config: c}
    })

    dix.Provide(di, func(db *Database) *UserService {
        return &UserService{DB: db}
    })

    // 注入使用
    dix.Inject(di, func(svc *UserService) {
        fmt.Println("DSN:", svc.DB.Config.DSN)
    })
}
```

生产环境启动建议优先使用 `TryProvide` / `TryInject`，避免 panic 并保留进程用于诊断：

```go
if err := dix.TryProvide(di, NewDatabase); err != nil {
    log.Fatal(err)
}
if err := dix.TryInject(di, Run); err != nil {
    log.Fatal(err)
}
```

## 📖 核心 API

| API | 失败是否 panic | 说明 |
| --- | --- | --- |
| `New(...Option)` | — | 创建容器 |
| `Provide(di, fn)` | 是 | 注册 provider |
| `TryProvide(di, fn)` | 否 | 注册 provider，返回 `error` |
| `Inject(di, target)` | 是 | 向函数或结构体注入依赖 |
| `TryInject(di, target)` | 否 | 注入依赖，返回 `error` |
| `InjectT[T](di)` | 是 | 分配结构体并注入字段 |
| `InjectTContext[T](ctx, di)` | 是 | 分配结构体并进行带 trace 上下文注入 |
| `InjectContext` / `TryInjectContext` | 是 / 否 | 带 trace 上下文传播注入 |
| `Version()` | — | 返回内嵌版本号 |

容器选项：

| 选项 | 默认值 | 说明 |
| --- | --- | --- |
| `WithValuesNull()` | 开启 | 容忍缺失的 map/list 依赖（解析为空集合） |
| `WithRejectEmptyCollections()` | 关闭 | 拒绝缺失的 map/list 依赖，而不是解析为空集合 |
| `WithProviderTimeout(d)` | `15s` | 单次 provider 执行超时（`0` = 关闭） |
| `WithSlowProviderThreshold(d)` | `2s` | 慢 provider 告警阈值（`0` = 关闭） |

### Provide / TryProvide

注册构造函数（Provider）到容器：

```go
// 标准版本 - 失败会 panic
dix.Provide(di, func() *Service { return &Service{} })

// 安全版本 - 返回 error
err := dix.TryProvide(di, func() *Service { return &Service{} })
if err != nil {
    log.Printf("注册失败: %v", err)
}
```

### Inject / TryInject

从容器注入依赖：

```go
// 函数注入
dix.Inject(di, func(svc *Service) {
    svc.DoSomething()
})

// 结构体注入
type App struct {
    Service *Service
    Config  *Config
}
app := &App{}
dix.Inject(di, app)

// 安全版本
err := dix.TryInject(di, func(svc *Service) {
    // ...
})
```

### 泛型辅助

```go
// 创建结构体并注入字段
app := dix.InjectT[App](di)

// 带请求级 trace 上下文注入
err := dix.TryInjectContext(ctx, di, func(svc *Service) {
    svc.DoSomething()
})
```

### 线程安全

`Dix` 容器**不是线程安全的**。请勿在同一容器实例上并发调用 `Provide` / `Inject`（及其 `Try*` 变体）。

推荐用法：

- 在应用启动阶段（单 goroutine）完成全部 provider 注册。
- 启动完成后仅读取已解析的依赖，或继续在单 goroutine 中注入。
- 若需要隔离容器，请为每个 goroutine 使用独立的 `Dix` 实例。
- 进程级单例请使用 `dixglobal`，且仅在启动阶段单线程注册。

### 启动选项

```go
di := dix.New(
    dix.WithProviderTimeout(2*time.Second),              // 默认 15s；0 表示关闭
    dix.WithSlowProviderThreshold(300*time.Millisecond), // 默认 2s；0 表示关闭
)
```

## 🎯 注入模式

### 结构体注入

```go
type In struct {
    Config   *Config
    Database *Database
}

type Out struct {
    UserSvc  *UserService
    OrderSvc *OrderService
}

// 多输入多输出
dix.Provide(di, func(in In) Out {
    return Out{
        UserSvc:  &UserService{DB: in.Database},
        OrderSvc: &OrderService{DB: in.Database},
    }
})
```

### Map 注入（命名空间）

```go
// 提供带 namespace 的依赖
dix.Provide(di, func() map[string]*Database {
    return map[string]*Database{
        "master": &Database{DSN: "master-dsn"},
        "slave":  &Database{DSN: "slave-dsn"},
    }
})

// 注入特定 namespace
dix.Inject(di, func(dbs map[string]*Database) {
    master := dbs["master"]
    slave := dbs["slave"]
})
```

### List 注入

```go
// 多次提供同类型
dix.Provide(di, func() []Handler {
    return []Handler{&AuthHandler{}}
})
dix.Provide(di, func() []Handler {
    return []Handler{&LogHandler{}}
})

// 注入时获取所有
dix.Inject(di, func(handlers []Handler) {
    // handlers 包含 AuthHandler 和 LogHandler
})
```

## 🧩 模块

### dixglobal - 全局容器

提供全局单例容器，适合简单应用：

```go
import "github.com/pubgo/dix/v2/dixglobal"

// 直接使用，无需创建容器
dixglobal.Provide(func() *Config { return &Config{} })
dixglobal.Inject(func(c *Config) { /* ... */ })
```

### dixcontext - Context 集成

将容器绑定到 `context.Context`：

```go
import "github.com/pubgo/dix/v2/dixcontext"

// 存入 context
ctx := dixcontext.Create(context.Background(), di)

// 取出使用
container := dixcontext.Get(ctx)

// 不 panic 的查询
container = dixcontext.GetOrNil(ctx)
```

### dixhttp - 依赖可视化

提供 Web 界面可视化依赖关系图，**专为大型项目设计**：

```go
import (
    "github.com/pubgo/dix/v2/dixhttp"
    "github.com/pubgo/dix/v2/dixinternal"
)

server := dixhttp.NewServer((*dixinternal.Dix)(di))
server.ListenAndServe(":8080")
```

访问 `http://localhost:8080` 查看依赖图。

> **安全提示**：会暴露依赖图、provider 源码位置、运行时错误和 trace 数据。请仅在**本机或内网**使用，勿在未鉴权情况下公网暴露。

**功能亮点**：
- 🔍 **模糊搜索** - 快速定位类型或函数
- 📦 **按包分组** - 可折叠侧边栏浏览
- 🔄 **双向追踪** - 同时展示依赖和被依赖
- 📏 **深度控制** - 限制展示层级（1-5 或全部）
- 🎨 **现代 UI** - Tailwind CSS + Alpine.js

详见 [dixhttp/README_zh.md](./dixhttp/README_zh.md)（API 路由、`di_trace` 事件字典与 UI 排障流程）。

## 🔍 诊断与排障

以下为可选观测能力，用于启动与注入排障。未配置时不会输出文件或额外控制台日志。

| 环境变量 | 默认值 | 用途 |
| --- | --- | --- |
| `DIX_TRACE_DI` | 关闭 | 控制台逐步 DI trace（`di_trace ...`） |
| `DIX_DIAG_FILE` | 关闭 | 追加写入 `trace` / `error` / `llm` JSONL |
| `DIX_TRACE_FILE` | 关闭 | 仅 trace 的 JSONL（未设置时回退到 `DIX_DIAG_FILE`） |

```bash
export DIX_TRACE_DI=true
export DIX_DIAG_FILE=.local/dix-diag.jsonl
```

内存 trace（`dixtrace`）默认开启，可通过 `dixhttp` 的 `/api/trace` 在线查询。

完整 `di_trace` 事件字典、HTTP API 与 UI 排障流程见 [dixhttp/README_zh.md](./dixhttp/README_zh.md)。

## 🛠️ 开发

```bash
# 运行全部测试并生成覆盖率报告
task test

# 代码检查与格式化
task lint

# go vet
task vet

# HTTP 可视化演示
task web-demo
```

GitHub Actions 会在 push/PR 时执行 `go test ./... -race` 与 `golangci-lint`。

## 📚 示例

命名与所演示特性对应:`inject-*` 为注入模式,`provide-*` 为 provider 能力,`context-*` 为 Context 集成。

### 注入模式

| 示例                                             | 说明                                  |
| ------------------------------------------------ | ------------------------------------- |
| [inject-func](./example/inject-func/)            | 函数类型注入                          |
| [inject-struct](./example/inject-struct/)        | 结构体字段注入(嵌套)                |
| [inject-method](./example/inject-method/)        | `DixInject` 方法注入                  |
| [inject-generic](./example/inject-generic/)      | 泛型 `InjectT` / `InjectTContext`     |
| [context-inject](./example/context-inject/)      | 带 Context 的注入与 trace 传播        |
| [inject-map](./example/inject-map/)              | Map/命名空间注入                      |
| [inject-list](./example/inject-list/)            | List 聚合注入                         |
| [inject-map-list](./example/inject-map-list/)    | `map[string][]T` 分组聚合             |
| [provide-multi-output](./example/provide-multi-output/) | 结构体多输出 provider          |

### 运行语义

| 示例                                             | 说明                                       |
| ------------------------------------------------ | ------------------------------------------ |
| [lazy](./example/lazy/)                          | 惰性求值与产物缓存                         |
| [cycle](./example/cycle/)                        | 循环依赖检测                               |
| [singleton](./example/singleton/)                | 容器级单例共享                             |
| [error-handling](./example/error-handling/)      | `TryProvide`/`TryInject` 错误处理          |
| [timeout](./example/timeout/)                    | provider 超时与慢告警                      |
| [empty-collections](./example/empty-collections/) | 缺失集合依赖解析为空集合(可改为报错)     |

### 模块

| 示例                                              | 说明                     |
| ------------------------------------------------- | ------------------------ |
| [global](./example/global/)                       | `dixglobal` 全局容器     |
| [context-container](./example/context-container/) | `dixcontext` 容器随 Context 传递 |
| [custom-logger](./example/custom-logger/)         | `SetLog` 自定义日志      |
| [http](./example/http/)                           | HTTP 可视化(端到端)     |

## 📖 文档

| 文档                                   | 说明                 |
| -------------------------------------- | -------------------- |
| [设计文档](./docs/design_zh.md)        | 架构和详细设计       |
| [审计报告](./docs/audit_zh.md)         | 项目审计、评价和对比 |
| [dixhttp 文档](./dixhttp/README_zh.md) | HTTP 可视化模块文档  |

## 📄 License

MIT
