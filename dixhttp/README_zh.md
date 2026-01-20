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
- 📡 **RESTful API** - 提供 JSON 格式的依赖关系数据

## 快速开始

```go
package main

import (
    "log"
    "github.com/pubgo/dix/v2"
    "github.com/pubgo/dix/v2/dixhttp"
    "github.com/pubgo/dix/v2/dixinternal"
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
    server := dixhttp.NewServer((*dixinternal.Dix)(di))
    log.Println("服务器启动在 http://localhost:8080")
    if err := server.ListenAndServe(":8080"); err != nil {
        log.Fatal(err)
    }
}
```

打开浏览器访问 `http://localhost:8080` 即可查看依赖关系图。

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

| 区域 | 说明 |
|------|------|
| **左侧 - 包列表** | 按包分组的 Provider 列表，支持搜索过滤，可折叠收起 |
| **中间 - 依赖图** | 交互式依赖关系图，支持拖拽、缩放、点击 |
| **右侧 - 详情面板** | 显示选中节点的详细信息，可点击跳转 |

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

| 深度 | 说明 | 适用场景 |
|------|------|---------|
| 1 | 只显示直接依赖/被依赖 | 快速查看直接关系 |
| 2 | 显示两层关系（默认） | 日常使用推荐 |
| 3-5 | 显示更多层级 | 追踪复杂依赖链 |
| 全部 | 展示完整依赖树 | 小型项目或特定分析 |

**示例**：假设依赖链是 `Config → Database → UserService → Handler`

搜索 `UserService`，不同深度显示：
- 深度 1: `Database ← UserService → Handler`
- 深度 2: `Config ← Database ← UserService → Handler`

### 📦 按包分组

左侧面板功能：
- **包列表** - 显示所有包及其 Provider 数量
- **搜索过滤** - 快速定位特定包
- **点击过滤** - 只显示该包的依赖关系
- **折叠收起** - 点击 `«` 按钮收起侧边栏，扩大图形区域

## 交互操作

| 操作 | 效果 |
|------|------|
| **单击节点** | 右侧显示详情 |
| **双击节点** | 以该节点为中心展示依赖图 |
| **拖拽节点** | 移动节点位置 |
| **滚轮缩放** | 放大/缩小图形 |
| **点击详情中的类型** | 跳转查看该类型的依赖 |

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
      "function_name": "main.NewServiceA",
      "input_types": ["*main.Config"]
    }
  ],
  "objects": [...],
  "edges": [...]
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
