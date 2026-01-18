# dix
> dix 是一个依赖注入框架

> dix 参考了 [uber-go/dig](https://github.com/uber-go/dig) 的设计, 它能够完成更加复杂的依赖注入管理和namespace依赖隔离

## 功能描述
1. dix 支持依赖循环检测
2. dix 支持 func, struct, map, list 作为注入参数
3. dix 支持 map key 作为 namespace 来进行依赖注入的数据隔离
4. dix 支持 struct 对外提供多组依赖对象
5. dix 支持 struct 依赖嵌套
6. dix Inject 支持 func 和 struct 等多种模式进行数据注入
7. dix 对象提供和注入对于原对象无任何侵入
8. 详情请看 [test example](./example/struct-in/main.go)

## 安装
```bash
go get github.com/pubgo/dix/v2
```

## 快速开始

### 创建容器
```go
import "github.com/pubgo/dix/v2/dixinternal"

// 创建一个新的依赖注入容器
container := dixinternal.New()
```

### 注册依赖
```go
// 注册一个简单的构造函数
container.Provide(func() *MyService {
    return &MyService{}
})

// 注册带依赖的构造函数
container.Provide(func(repo *Repository) *Service {
    return &Service{Repo: repo}
})
```

### 注入依赖
```go
// 函数注入
container.Inject(func(service *Service) {
    // 使用注入的服务
    service.DoSomething()
})

// 结构体注入
target := &MyStruct{}
container.Inject(target)
```

## 核心特性

### 依赖循环检测
dix 能够自动检测并防止依赖循环，避免程序陷入死循环。

### 多种注入模式
- **函数注入**: 通过函数参数自动注入依赖
- **结构体注入**: 自动填充结构体字段
- **集合注入**: 支持切片和映射类型的依赖注入

### Namespace 隔离
通过 map 的 key 作为 namespace 实现依赖注入的数据隔离，不同命名空间的依赖互不影响。

### 无侵入设计
dix 的依赖注入对原有代码无任何侵入性，不需要修改原有结构即可实现依赖注入。

## 使用示例

### 结构体注入示例
```go
type A struct {
    B B
}

type B struct {
    C *C
}

type C struct {
    Value string
}

func main() {
    container := dixinternal.New()
    
    // 提供依赖
    container.Provide(func() *C {
        return &C{Value: "hello"}
    })
    
    // 注入结构体
    a := &A{}
    container.Inject(a)
    
    fmt.Println(a.B.C.Value) // 输出: hello
}
```

### 函数注入示例
```go
type Handler func() string

func main() {
    container := dixinternal.New()
    
    // 提供函数依赖
    container.Provide(func() Handler {
        return func() string {
            return "hello"
        }
    })
    
    // 函数注入
    container.Inject(func(h Handler) {
        fmt.Println(h()) // 输出: hello
    })
}
```

### Map注入示例
```go
func main() {
    container := dixinternal.New()
    
    // 提供map依赖
    container.Provide(func() map[string]error {
        return map[string]error{
            "default": errors.New("default error"),
            "custom":  errors.New("custom error"),
        }
    })
    
    // 注入map
    container.Inject(func(errMap map[string]error) {
        fmt.Println(errMap["default"]) // 输出: default error
        fmt.Println(errMap["custom"])  // 输出: custom error
    })
}
```

更多使用示例请参考 [example 目录](./example/) 中的代码：
- [结构体注入示例](./example/struct-in/main.go)
- [函数注入示例](./example/func/main.go)
- [Map注入示例](./example/map/main.go)

## 构建和测试

### 运行测试
```bash
make test
```

### 代码检查
```bash
make lint
```

### 代码审查
```bash
make vet
```

## API 文档

### New()
创建一个新的依赖注入容器实例。

### Provide()
注册构造函数，用于提供依赖对象。

### Inject()
注入依赖到目标函数或结构体。

## 扩展功能

### 全局容器
dix 提供了全局容器，可以通过 `dixglobal` 包直接使用：

```go
import "github.com/pubgo/dix/v2/dixglobal"

// 注册依赖
dixglobal.Provide(func() *ServiceA { return &ServiceA{} })

// 注入依赖
dixglobal.Inject(func(s *ServiceA) {
    // 使用服务
})
```

### 上下文支持
dix 支持将容器实例存储在 context 中：

```go
import (
    "context"
    "github.com/pubgo/dix/v2/dixcontext"
)

// 将容器放入上下文
ctx := dixcontext.Create(context.Background(), container)

// 从上下文中获取容器
container := dixcontext.Get(ctx)
```

### HTTP 可视化
dix 提供了 HTTP 可视化模块，用于图形化展示依赖关系：

```go
import (
    "log"
    "github.com/pubgo/dix/v2/dixhttp"
)

// 创建 HTTP 服务器
server := dixhttp.NewServer(container)

// 启动服务器
log.Println("服务器启动在 http://localhost:8080")
if err := server.ListenAndServe(":8080"); err != nil {
    log.Fatal(err)
}
```

## License
MIT