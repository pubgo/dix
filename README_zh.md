# dix

[![Go Reference](https://pkg.go.dev/badge/github.com/pubgo/dix/v2.svg)](https://pkg.go.dev/github.com/pubgo/dix/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/pubgo/dix)](https://goreportcard.com/report/github.com/pubgo/dix)

> **dix** 是一个轻量、强大的 Go 依赖注入框架

参考了 [uber-go/dig](https://github.com/uber-go/dig) 的设计，支持更复杂的依赖管理和 namespace 隔离。

[English](./README.md)

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

## 📖 核心 API

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

### 启动超时 / 慢 Provider 告警

可在启动阶段限制 provider 执行时间，并对慢调用输出告警：

- 默认 `ProviderTimeout` 为 `15s`
- 可用 `dix.WithProviderTimeout(0)` 显式关闭 provider 超时
- 默认 `SlowProviderThreshold` 为 `2s`
- 可用 `dix.WithSlowProviderThreshold(0)` 显式关闭慢 provider 告警

```go
di := dix.New(
    // 默认 `ProviderTimeout` 为 `15s`
    // 使用 `dix.WithProviderTimeout(0)` 可关闭 provider 超时
    // 默认 `SlowProviderThreshold` 为 `2s`
    // 使用 `dix.WithSlowProviderThreshold(0)` 可关闭慢 provider 告警
    dix.WithProviderTimeout(2*time.Second),               // 覆盖默认值（default: 15s, 0 = 不限制）
    dix.WithSlowProviderThreshold(300*time.Millisecond),  // 覆盖默认值（default: 2s, 0 = 关闭）
)
```

### DI 追踪日志（可选）

可开启“依赖查询 / 注入 / Provider 执行”的全过程日志：

- 环境变量：`DIX_TRACE_DI`
- 默认：关闭
- 开启取值：`1`、`true`、`on`、`yes`、`enable`、`trace`、`debug`

```bash
export DIX_TRACE_DI=true
```

开启后会输出 `di_trace ...` 事件，包含结构化键值（如 provider、输入输出类型、查询类型、父链路、超时等）。

> 注意：若设置 `DIX_LLM_DIAG_MODE=machine`，会按设计抑制人类文本日志，`di_trace` 也会被抑制。

### 诊断文件采集（可选）

你可以把更完整的诊断信息写入可检索的 JSONL 文件：

- 环境变量：`DIX_DIAG_FILE`
- 示例：`export DIX_DIAG_FILE=.local/dix-diag.jsonl`

行为规则：

- 如果 **未配置** `DIX_DIAG_FILE`，dix 保持原有方案（不输出诊断文件）。
- 如果配置了 `DIX_DIAG_FILE`，dix 会追加写入诊断记录（`trace` / `error` / `llm`）。
- 终端可见日志仍由现有开关控制（`DIX_TRACE_DI`、`DIX_LLM_DIAG_MODE`）。

建议：

- 终端给用户看“少而准”。
- 文件给排障/LLM看“全而细”。

### 内存 Trace 查询（`dixtrace`，可选）

从这个版本开始，dix 会把统一 trace 事件写入内存 trace 存储（`dixtrace`），可通过 HTTP API（`/api/trace`）在线查询。

- 默认：开启（内存环形缓冲）
- 可选文件落盘环境变量：`DIX_TRACE_FILE`
- 示例：`export DIX_TRACE_FILE=.local/dix-trace.jsonl`

`/api/trace` 适合在线排障（按 `operation/status/event/component/provider/output_type` 等过滤）。
如需长期持久化，请同时配置 `DIX_TRACE_FILE`。

事件速查：

| 事件                                           | 含义                                                                    |
| ---------------------------------------------- | ----------------------------------------------------------------------- |
| `di_trace inject.start`                        | 开始一次注入请求（`component`、`param_type`）                           |
| `di_trace inject.route`                        | 注入路径已确定（`function` 或 `struct`）                                |
| `di_trace provide.start`                       | 开始一次 provider 注册请求（`component`）                               |
| `di_trace provide.signature`                   | provider 函数签名分析完成（`input_count`、`output_count`）              |
| `di_trace provide.register.output.done`        | provider 输出类型注册成功                                               |
| `di_trace provide.register.failed`             | provider 注册失败（含 `reason` 或 `error`）                             |
| `di_trace resolve.value.search_provider.start` | 开始为某个依赖类型查找 provider                                         |
| `di_trace resolve.value.found`                 | 依赖值查找成功                                                          |
| `di_trace resolve.value.not_found`             | 依赖值查找失败（含 `reason`）                                           |
| `di_trace provider.execute.dispatch`           | 选择并派发 provider 执行（含 `provider`、`output_type`、`input_types`） |
| `di_trace provider.input.resolve.start`        | 开始解析 provider 的某个输入                                            |
| `di_trace provider.input.resolve.found`        | provider 输入解析成功                                                   |
| `di_trace provider.input.resolve.failed`       | provider 输入解析失败                                                   |
| `di_trace provider.call.start`                 | 开始执行 provider（含 `timeout`）                                       |
| `di_trace provider.call.done`                  | provider 执行完成                                                       |
| `di_trace provider.call.failed`                | provider 执行失败（含 `timed_out`、`error`）                            |
| `di_trace provider.call.return_error`          | provider 返回了非 nil `error`                                           |
| `di_trace inject.func.resolve_input.start`     | 开始解析函数注入参数                                                    |
| `di_trace inject.func.resolve_input.failed`    | 函数注入参数解析失败                                                    |
| `di_trace inject.struct.field.resolve.start`   | 开始解析结构体字段注入                                                  |
| `di_trace inject.struct.field.resolve.done`    | 结构体字段注入成功                                                      |
| `di_trace inject.struct.field.resolve.failed`  | 结构体字段注入失败                                                      |

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
```

### dixhttp - 依赖可视化 🆕

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

**功能亮点**：
- 🔍 **模糊搜索** - 快速定位类型或函数
- 📦 **按包分组** - 可折叠侧边栏浏览
- 🔄 **双向追踪** - 同时展示依赖和被依赖
- 📏 **深度控制** - 限制展示层级（1-5 或全部）
- 🎨 **现代 UI** - Tailwind CSS + Alpine.js

详见 [dixhttp/README.md](./dixhttp/README.md)

## 🛠️ 开发

```bash
# 运行测试
task test

# 代码检查
task lint

# 构建
task build
```

## 📚 示例

| 示例                                              | 说明             |
| ------------------------------------------------- | ---------------- |
| [struct-in](./example/struct-in/)                 | 结构体输入注入   |
| [struct-out](./example/struct-out/)               | 结构体多输出     |
| [func](./example/func/)                           | 函数注入         |
| [map](./example/map/)                             | Map/命名空间注入 |
| [map-nil](./example/map-nil/)                     | Map 空值处理     |
| [list](./example/list/)                           | List 注入        |
| [list-nil](./example/list-nil/)                   | List 空值处理    |
| [lazy](./example/lazy/)                           | 延迟注入         |
| [cycle](./example/cycle/)                         | 循环检测示例     |
| [handler](./example/handler/)                     | Handler 模式     |
| [inject_method](./example/inject_method/)         | 方法注入         |
| [test-return-error](./example/test-return-error/) | 错误处理         |
| [http](./example/http/)                           | HTTP 可视化      |

## 📖 文档

| 文档                                   | 说明                 |
| -------------------------------------- | -------------------- |
| [设计文档](./docs/design_zh.md)        | 架构和详细设计       |
| [审计报告](./docs/audit_zh.md)         | 项目审计、评价和对比 |
| [dixhttp 文档](./dixhttp/README_zh.md) | HTTP 可视化模块文档  |

## 📄 License

MIT
