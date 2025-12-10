# Dix HTTP 可视化模块

这个模块提供了一个 HTTP 服务器，用于可视化展示 Dix 依赖注入容器中的 Providers 和 Objects 之间的依赖关系。

## 功能特性

- 📊 **交互式可视化**: 使用 vis.js 在浏览器中展示依赖关系图
- 🔄 **多种视图**: 支持 Providers 视图、Objects 视图和组合视图
- 📡 **RESTful API**: 提供 JSON 格式的依赖关系数据
- 🎨 **美观的界面**: 现代化的 UI 设计，易于使用

## 使用方法

### 基本使用

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
    dix.Provide(di, func() *ServiceA {
        return &ServiceA{}
    })
    
    dix.Provide(di, func(a *ServiceA) *ServiceB {
        return &ServiceB{ServiceA: a}
    })
    
    // 创建 HTTP 服务器
    server := dixhttp.NewServer((*dixinternal.Dix)(di))
    
    // 启动服务器
    log.Println("服务器启动在 http://localhost:8080")
    if err := server.ListenAndServe(":8080"); err != nil {
        log.Fatal(err)
    }
}
```

### API 端点

#### GET `/`
返回 HTML 可视化页面

#### GET `/api/dependencies`
返回 JSON 格式的依赖关系数据

响应格式：
```json
{
  "providers": [
    {
      "id": "provider_*main.ServiceA_0",
      "output_type": "*main.ServiceA",
      "function_name": "main.initServiceA",
      "input_types": []
    }
  ],
  "objects": [
    {
      "id": "object_*main.ServiceA_default_0",
      "type": "*main.ServiceA",
      "group": "default",
      "is_initialized": true
    }
  ],
  "edges": [
    {
      "from": "*main.ServiceA",
      "to": "*main.ServiceB",
      "type": "provider"
    }
  ]
}
```

#### GET `/api/graph?type=providers`
返回 DOT 格式的依赖图（用于 Graphviz）

支持的 type 参数：
- `providers`: Provider 函数视图
- `provider_types`: Provider 类型视图
- `objects`: Objects 视图

## 可视化界面

打开浏览器访问 `http://localhost:8080`，你将看到：

1. **顶部控制栏**: 切换不同的视图模式
2. **主视图区**: 交互式依赖关系图
   - 绿色节点: Provider 函数
   - 蓝色节点: 类型
   - 橙色节点: Object 实例
3. **侧边栏**: 统计信息和图例

### 交互功能

- **拖拽**: 可以拖拽节点重新布局
- **缩放**: 使用鼠标滚轮缩放
- **点击**: 点击节点查看详细信息
- **悬停**: 鼠标悬停显示节点信息

## 示例

查看 `example/http/main.go` 获取完整示例。
