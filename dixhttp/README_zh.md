# Dix HTTP 可视化模块

这个模块提供了一个 HTTP 服务器，用于可视化展示 Dix 依赖注入容器中的依赖关系。专为**大型项目**设计，支持模糊搜索、按包分组浏览、双向依赖追踪和层级深度控制。

[English](./README.md)

## 功能特性

- 📊 **交互式可视化** - 使用 vis.js + Tailwind CSS + Alpine.js 构建现代化界面
- 🔍 **全局模糊搜索** - 快速搜索类型名或函数名，直接查看依赖关系
- 📦 **按包分组** - 左侧可折叠面板，按包过滤查看依赖
- 🔄 **双向依赖追踪** - 同时展示依赖（上游）和被依赖（下游）关系
- 📏 **深度控制** - 限制依赖图的展示层级（1-5级或全部）
- 🎨 **多种布局** - 支持层级布局和力导向布局
- 🧩 **分组清单（前缀聚合）** - 通过包路径/前缀聚合节点
- 🔎 **前缀过滤** - 只显示匹配前缀的节点/Provider
- 🧭 **组内子图** - 查看组内及上下游依赖
- 🗂 **诊断文件查询** - 配置 `DIX_DIAG_FILE` 后，页面可查询并展示 `trace/error/llm` JSONL 记录
- 🧵 **Trace 时间线查询** - 通过 `/api/trace` 查询 `dixtrace` 内存统一事件（支持多维过滤）；文件持久化优先使用 `DIX_TRACE_FILE`，未配置时回退复用 `DIX_DIAG_FILE`
- 📡 **RESTful API** - 提供 JSON 格式的依赖关系数据
- 🧩 **Mermaid 预览/导出** - 将当前图生成 Mermaid 流程图（支持分组/过滤）

## 快速开始

```go
package main

import (
    "log"
    "github.com/pubgo/dix/v2"
    "github.com/pubgo/dix/v2/dixhttp"
)

func main() {
    // 创建 Dix 容器
    di := dix.New()
    
    // 注册 providers
    dix.Provide(di, func() *Config {
        return &Config{}
    })
    
    dix.Provide(di, func(c *Config) *Database {
        return &Database{Config: c}
    })
    
    dix.Provide(di, func(db *Database) *UserService {
        return &UserService{DB: db}
    })
    
    // 创建并启动 HTTP 服务器
    server := dixhttp.NewServer(di)
    log.Println("服务器启动在 http://localhost:8080")
    if err := server.ListenAndServe(":8080"); err != nil {
        log.Fatal(err)
    }
}
```

打开浏览器访问 `http://localhost:8080` 即可查看依赖关系图。

### 配置访问前缀

如果需要将页面和 API 挂载到一个前缀路径（例如网关转发），可以使用 `WithBasePath`：

```go
server := dixhttp.NewServerWithOptions(
  di,
  dixhttp.WithBasePath("/dix"),
)
// 访问 http://localhost:8080/dix/
```

### 反向代理鉴权示例（推荐）

`dixhttp` 本身不内置鉴权中间件。生产环境建议放在反向代理后，并开启鉴权与网络访问限制。

示例（Nginx + Basic Auth）：

```nginx
location /dix/ {
    auth_basic "Restricted Dix";
    auth_basic_user_file /etc/nginx/.htpasswd;

    proxy_pass http://127.0.0.1:8080/dix/;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

实用安全清单：

- dixhttp 服务仅监听本机或内网地址。
- 放在鉴权层之后（Basic Auth / SSO / 网关 Token）。
- 尽量按来源 IP/CIDR 做白名单限制。
- 不要将 `/api/errors`、`/api/diagnostics`、`/api/trace` 暴露到公网。

## 界面布局

```
┌─────────────────────────────────────────────────────────────────────┐
│  🔗 Dix 依赖关系可视化                    📦 8 包  ⚡ 42 Providers  │
├──────────┬────────────────────────────────────────────┬─────────────┤
│ 📦 包列表 │  [Providers] [类型依赖]  布局: 层级  深度: 2  🔍 搜索...  │ 📋 节点详情 │
│          │                                            │             │
│ 📁 全部   │                                            │  函数名     │
│  42      │           ┌─────────┐                      │  xxx.func   │
│          │           │ Config  │                      │             │
│ .../app  │           └────┬────┘                      │  输出类型   │
│  12      │                │                           │  *Config    │
│          │           ┌────▼────┐                      │             │
│ .../db   │           │Database │                      │  输入类型   │
│  8       │           └────┬────┘                      │  (点击跳转) │
│          │                │                           │             │
│          │        ┌───────▼───────┐                   │ [查看依赖]  │
│          │        │  UserService  │                   │             │
│          │        └───────────────┘                   │             │
├──────────┴────────────────────────────────────────────┼─────────────┤
│  « 收起                                               │   图例      │
└───────────────────────────────────────────────────────┴─────────────┘
```

### 三栏布局

| 区域                | 说明                                               |
| ------------------- | -------------------------------------------------- |
| **左侧 - 包列表**   | 按包分组的 Provider 列表，支持搜索过滤，可折叠收起 |
| **中间 - 依赖图**   | 交互式依赖关系图，支持拖拽、缩放、点击             |
| **右侧 - 详情面板** | 显示选中节点的详细信息，可点击跳转                 |

## 核心功能

### 🔍 全局模糊搜索

工具栏右侧的搜索框支持：

- **模糊匹配** - 输入关键词匹配类型名或函数名
- **实时提示** - 显示匹配结果下拉列表（最多 20 条）
- **快速跳转** - 点击结果或按 Enter 直接查看依赖图
- **分类标签** - 结果标记为 `Provider` 或 `Type`

### 🔄 双向依赖追踪

搜索或点击类型后，系统会以该类型为中心展示：

```
     ┌──────────┐
     │ 上游依赖  │  ← 绿色节点
     │ (Config) │
     └────┬─────┘
          │
          ▼
     ┌──────────┐
     │  目标类型 │  ← 黄色高亮
     │(Database)│
     └────┬─────┘
          │
          ▼
     ┌──────────┐
     │ 下游被依赖│  ← 红色节点
     │(Service) │
     └──────────┘
```

**颜色说明**：
- 🟡 **黄色** - 目标节点（搜索的对象）
- 🟢 **绿色** - 依赖（上游，目标依赖什么）
- 🔴 **红色** - 被依赖（下游，什么依赖目标）

### 📏 深度控制

深度决定了向上/向下展开多少层依赖：

| 深度 | 说明                  | 适用场景           |
| ---- | --------------------- | ------------------ |
| 1    | 只显示直接依赖/被依赖 | 快速查看直接关系   |
| 2    | 显示两层关系（默认）  | 日常使用推荐       |
| 3-5  | 显示更多层级          | 追踪复杂依赖链     |
| 全部 | 展示完整依赖树        | 小型项目或特定分析 |

**示例**：假设依赖链是 `Config → Database → UserService → Handler`

搜索 `UserService`，不同深度显示：
- 深度 1: `Database ← UserService → Handler`
- 深度 2: `Config ← Database ← UserService → Handler`

### 🧩 分组清单（前缀聚合）

支持通过**包路径/前缀**聚合节点，规则来源：

- **前端分组清单**
- **后端全局注册**（推荐生产使用）

后端注册示例：

```go
import "github.com/pubgo/dix/v2/dixhttp"

dixhttp.RegisterGroupRules(
  dixhttp.GroupRule{
    Name: "service",
    Prefixes: []string{
      "github.com/acme/app/service",
      "github.com/acme/app/internal/service",
    },
  },
  dixhttp.GroupRule{
    Name: "router",
    Prefixes: []string{"github.com/acme/app/router"},
  },
)
```

当本地未配置分组清单时，前端会自动加载 `/api/group-rules`。

### 🔎 前缀过滤

工具栏提供“前缀过滤”，可按包路径/类型名/函数名过滤当前图：

- Providers/类型视图
- 类型依赖视图
- 组内依赖视图

### 🧭 组内子图

点击虚拟组节点 → 详情面板 → “查看组内依赖图”，可以看到：

- 组内节点
- 上下游依赖
- 受工具栏“深度”控制

### 📦 按包分组

左侧面板功能：
- **包列表** - 显示所有包及其 Provider 数量
- **搜索过滤** - 快速定位特定包
- **点击过滤** - 只显示该包的依赖关系
- **折叠收起** - 点击 `«` 按钮收起侧边栏，扩大图形区域

## 交互操作

| 操作                 | 效果                     |
| -------------------- | ------------------------ |
| **单击节点**         | 右侧显示详情             |
| **双击节点**         | 以该节点为中心展示依赖图 |
| **拖拽节点**         | 移动节点位置             |
| **滚轮缩放**         | 放大/缩小图形            |
| **点击详情中的类型** | 跳转查看该类型的依赖     |

## Mermaid 支持

工具栏新增 **Mermaid** 按钮，会基于**当前视图**生成 Mermaid `flowchart`，并弹出预览窗口，支持一键复制源码。

**典型用法**：
1. 调整视图/分组/过滤条件。
2. 点击 **Mermaid**。
3. 复制生成的 Mermaid 文本或直接预览。

## API 端点

### GET `/`
返回 HTML 可视化页面

### GET `/api/stats`
返回统计概要

```json
{
  "provider_count": 42,
  "object_count": 15,
  "package_count": 8,
  "edge_count": 67
}
```

### GET `/api/packages`
返回包列表

```json
[
  {
    "name": "github.com/example/app/service",
    "provider_count": 12,
    "types": ["*service.UserService", "*service.OrderService"]
  }
]
```

### GET `/api/dependencies?package=xxx&limit=100`
返回依赖关系数据，支持按包过滤

```json
{
  "providers": [
    {
      "id": "provider_*main.ServiceA_0",
      "output_type": "*main.ServiceA",
      "output_pkg": "github.com/example/app/service",
      "function_name": "main.NewServiceA",
      "function_pkg": "github.com/example/app",
      "input_types": ["*main.Config"],
      "input_pkgs": ["github.com/example/app/config"]
    }
  ],
  "objects": [...],
  "edges": [...]
}
```

### GET `/api/errors?limit=50`
返回最近的 `Inject` / `TryInject` 错误（按时间倒序），用于在启动阶段注入失败后继续排查。

```json
[
  {
    "operation": "provider_execute",
    "component": "main.main.func12",
    "stage": "resolve_input",
    "provider_function": "main.main.func12",
    "output_type": "*main.UserService",
    "input_type": "*main.Database",
    "root_cause": "value not found: type=*main.Database ...",
    "message": "failed to get input value for provider: value not found: type=*main.Database ...",
    "occurred_at_unix_nano": 1700000000000000000
  }
]
```

### GET `/api/diagnostics?kind=trace&q=provider&event=provider.call.start&limit=200`
读取并过滤 `DIX_DIAG_FILE` 中的 JSONL 记录。

若未配置 `DIX_DIAG_FILE`，返回 `enabled=false` 且记录为空。

```json
{
  "enabled": true,
  "path": "/tmp/dix-diag.jsonl",
  "exists": true,
  "total": 42,
  "returned": 42,
  "next_before_id": 0,
  "records": [
    {
      "record_id": 128,
      "source": "dix",
      "pid": 12345,
      "process": "my-app",
      "hostname": "dev-mac",
      "trace_di": true,
      "llm_diag_mode": "dual",
      "kind": "trace",
      "event": "provider.call.start",
      "occurred_at_unix_nano": 1700000000000000000,
      "fields": {
        "provider": "github.com/acme/app.main.NewDB"
      }
    }
  ]
}
```

### GET `/api/trace?operation=provider&status=error&limit=200`
返回来自 `dixtrace` 的内存统一 trace 事件。

文件落盘行为：

- 优先使用 `DIX_TRACE_FILE`（若已配置）。
- 当未配置 `DIX_TRACE_FILE` 且已配置 `DIX_DIAG_FILE` 时，trace 文件落盘会以追加模式复用 `DIX_DIAG_FILE`（单文件排查配置）。

支持过滤参数：

- `trace_id`, `operation`, `status`, `event`, `component`, `provider`, `output_type`, `q`
- `limit`, `before_id`, `since_unix_nano`, `until_unix_nano`

```json
{
  "enabled": true,
  "total": 2,
  "returned": 2,
  "records": [
    {
      "id": 102,
      "operation": "provider",
      "phase": "call.failed",
      "event": "provider.call.failed",
      "status": "error",
      "provider_function": "github.com/acme/app.main.NewDB",
      "output_type": "*db.Client",
      "error": "dial tcp timeout",
      "timed_out": true,
      "occurred_at_unix_nano": 1700000000000000000
    }
  ]
}
```

### GET `/api/package/{packageName}`
返回指定包内的 Provider 详情

### GET `/api/type/{typeName}?depth=2`
返回指定类型的依赖链

```json
{
  "root_type": "*service.UserService",
  "depth": 2,
  "nodes": [
    {"id": "*service.UserService", "type": "*service.UserService", "package": "...", "level": 0}
  ],
  "edges": [
    {"from": "*db.Database", "to": "*service.UserService", "type": "dependency"}
  ]
}

### GET `/api/group-rules`
返回后端注册的分组清单（前端默认配置）

```json
[
  {"name": "service", "prefixes": ["github.com/acme/app/service"]}
]
```
```

## 技术栈

- **后端**: Go 标准库 `net/http`
- **前端框架**: 
  - [Tailwind CSS](https://tailwindcss.com/) - 样式
  - [Alpine.js](https://alpinejs.dev/) - 响应式交互
  - [vis-network](https://visjs.github.io/vis-network/) - 图形渲染
- **模板**: Go embed 嵌入单文件 HTML

## 适用场景

✅ **推荐使用**：
- 大型项目（100+ providers）
- 模块化架构，需要按包查看
- 追踪特定类型的依赖链
- 调试依赖循环问题
- 新人了解项目结构

⚠️ **注意事项**：
- 生产环境建议限制访问（仅内网或开发环境）
- 超大型项目（1000+ providers）建议使用深度限制

## 示例

查看 `example/http/main.go` 获取完整示例。
