# Dix 依赖注入框架设计文档

[English](./design.md)

## 1. 概述

`dix` 是一个基于 Go 语言反射机制实现的轻量级依赖注入（DI）框架。参考了 `dig` 的设计理念，旨在通过自动化的依赖解析和生命周期管理，解耦组件间的依赖关系。

`dix` 的核心是对能够寻址的类型进行依赖注入管理，主要包括 `func`、`ptr`（指针）和 `interface`。

核心特性包括：
*   **构造函数注入**：通过 `Provide` 注册构造函数
*   **结构体/字段注入**：通过 `Inject` 对结构体字段进行填充
*   **高级类型支持**：支持 `Interface`、`Map`、`Slice` 等集合类型的自动聚合注入
*   **循环依赖检测**：内置图算法检测循环依赖
*   **Web 可视化**：HTTP 模块提供交互式依赖图可视化
*   **方法注入**：支持通过 `DixInject` 前缀方法进行 Setter 注入
*   **分组管理**：支持依赖项按命名空间进行分组管理
*   **安全 API**：`TryProvide`/`TryInject` 返回错误而非 panic

## 2. 核心架构

`dix` 的核心是一个名为 `Dix` 的容器结构体，它维护了提供者（Providers）的注册表和已实例化对象（Objects）的缓存。

### 2.1 支持的类型

`dix` 严格限制了可作为依赖项管理的类型范围，核心原则是**仅支持可寻址或引用类型**。这种设计确保了依赖关系的稳定性和对象生命周期的可控性。

*   **Pointer（指针）**:
    *   最常用的类型，通常指向一个结构体实例（如 `*Service`）
    *   指针保证了在整个应用生命周期中，组件是单例的，且状态共享
*   **Interface（接口）**:
    *   支持将具体实现绑定到接口定义（如 `Service` 接口由 `*ServiceImpl` 实现）
    *   这是实现依赖倒置原则（DIP）的关键
*   **Func（函数）**:
    *   函数本身是一等公民，可以作为依赖被注入
    *   常用于注入工厂函数、中间件或回调逻辑

*注意：基础类型（如 `int`、`string`、`bool`）不能直接作为依赖项注入，必须封装在上述类型中（通常是结构体字段）。*

### 2.2 数据结构

```go
type Dix struct {
    // 配置选项
    option Options

    // 提供者注册表
    // key: 输出类型 (reflect.Type)
    // value: 该类型对应的提供者函数列表
    providers map[outputType][]*providerFn

    // 对象缓存池 (单例模式)
    // key: 输出类型 -> 分组(group) -> 值列表
    objects map[outputType]map[group][]value

    // 初始化状态标记，防止 Provider 重复执行
    initializer map[reflect.Value]bool
}
```

### 2.3 核心流程

1.  **注册 (Provide)**: 用户注册构造函数 -> 解析函数签名（输入/输出） -> 存入 `providers` 表
2.  **调用 (Inject)**: 用户请求注入对象 -> 递归查找依赖 -> 执行 Provider 函数 -> 缓存结果 -> 填充目标

## 3. 详细设计

### 3.1 提供者注册

`Provide` 方法负责将构造函数注册到容器中。

#### 3.1.1 输入参数类型
Provider 函数的参数声明了该组件的依赖项：

*   **Func / Ptr / Interface**:
    *   **行为**: 声明为直接依赖
    *   **解析**: 容器会在已注册的 Provider 中查找匹配该类型的实例
*   **Struct**:
    *   **行为**: **递归依赖注入**
    *   **解析**: 框架会递归遍历该结构体的所有**导出字段**，如果字段类型是支持的类型，容器会自动查找并注入对应的实例
*   **Slice (`[]T`)**:
    *   **行为**: 声明为聚合依赖（List）
    *   **解析**: 容器查找所有能提供类型 `T` 的 Provider，并将它们在**默认分组**下的结果汇总成切片
*   **Map (`map[string]T`)**:
    *   **行为**: 声明为聚合依赖（Map）
    *   **解析**: 容器查找所有能提供类型 `T` 的 Provider，利用其返回的 Key（默认为 "default"）汇总成 Map

#### 3.1.2 返回值类型
Provider 函数的返回值定义了它向容器提供的组件。

**约束**：
Provider 函数必须返回 **1 个或 2 个** 值。
*   1 个值：该值即为提供的组件
*   2 个值：第二个值必须是 `error` 类型，如果 `error` 不为 nil，容器将停止初始化并报错

支持的具体类型：

*   **Func / Ptr / Interface**: 注册为该类型的标准 Provider
*   **Slice (`[]T`)**: 注册为类型 `T` 的列表 Provider
*   **Map (`map[string]T`)**: 注册为类型 `T` 的映射 Provider
*   **Struct**: **递归自动分解**，导出字段会被分别注册为对应类型的 Provider

### 3.2 依赖解析与注入

注入过程由 `Inject` 或内部的 `getValue` 驱动，采用**惰性求值**策略。

#### 3.2.1 注入目标类型

`Inject` 函数支持对结构体指针或函数进行注入。

**1. 结构体指针**
当传入 `&MyStruct{}` 时，框架会扫描其字段进行注入：
*   **Func / Ptr / Interface**: 查找并注入对应类型的实例
*   **Struct**: **递归注入**，框架会继续对嵌套结构体进行注入
*   **Slice / Map**: 执行聚合注入逻辑
*   **方法注入**: 自动扫描并执行以 `DixInject` 为前缀的方法

**2. 函数**
当传入一个函数时（如 `dix.Inject(func(a *A, b *B){ ... })`）：
*   参数列表被视为依赖项
*   解析逻辑与 Provider 的输入参数完全一致
*   常用于执行初始化逻辑或启动钩子

#### 3.2.2 内部存储与解析策略

当存在同一类型的多个 Provider 时，`dix` 使用分组（Group/Key）机制。底层存储结构是 `map[group][]value`。

解析策略：

1.  **单值依赖 (`T`)**: 查找**默认分组**，取**最后一个值**
2.  **列表依赖 (`[]T`)**: 查找**默认分组**，取**所有值**
3.  **映射依赖 (`map[string]T`)**: 查找**所有分组**，每个分组取**最后一个值**
4.  **完全映射依赖 (`map[string][]T`)**: 查找**所有分组**，取每个分组的**所有值**

### 3.3 循环依赖检测

为防止无限递归，`dix` 在执行注入前会构建依赖图。

*   **算法**: 深度优先搜索 (DFS)
*   **实现**: `dixinternal/cycle-check.go` 中的 `detectCycle` 函数
*   **逻辑**: 构建邻接表，遍历图寻找回边，如发现循环立即报错并打印循环路径

### 3.4 错误处理

使用日志系统进行错误记录。
*   **上下文丰富**: 错误信息包含堆栈跟踪、Provider 函数名、参数类型等详细信息
*   **Panic 捕获**: 执行用户代码时使用 `defer recover` 机制捕获 Panic
*   **安全 API**: `TryProvide` 和 `TryInject` 方法返回错误而非 panic

### 3.5 可视化

`dix` 通过 `dixhttp` 模块提供基于 HTTP 的可视化：

*   **Web 界面**: 使用 Tailwind CSS + Alpine.js + vis-network 构建的现代 UI
*   **功能**:
    *   模糊搜索类型和函数
    *   按包分组，可折叠侧边栏
    *   双向依赖追踪（上游和下游）
    *   深度控制（1-5 级或全部）
    *   交互式图形，支持拖拽、缩放、点击
*   **RESTful API**: JSON 端点用于程序化访问
    *   `/api/stats` - 统计摘要
    *   `/api/packages` - 包列表
    *   `/api/dependencies` - 完整依赖数据
    *   `/api/type/{name}` - 特定类型的依赖链

### 3.6 扩展模块

#### 3.6.1 全局容器
`dixglobal` 包提供全局容器实例，方便在应用程序的不同部分共享。

#### 3.6.2 Context 支持
`dixcontext` 包提供将容器实例存储在 context 中的功能，便于在请求链路中传递。

#### 3.6.3 HTTP 可视化
`dixhttp` 包提供 HTTP 服务器，用于可视化展示依赖图。详见 [dixhttp/README_zh.md](../dixhttp/README_zh.md)。

## 4. 模块划分

| 文件 | 职责 |
| :--- | :--- |
| `dix.go` | 公共 API 包装，支持泛型 (`Inject[T]`、`InjectT[T]`、`Provide`) |
| `dixinternal/api.go` | 核心公共 API (`New`、`Provide`、`TryProvide`、`Inject`、`TryInject`) |
| `dixinternal/dix.go` | 核心逻辑：`newDix` 初始化、`inject` 递归流程、Provider 注册 |
| `dixinternal/provider.go` | `providerFn` 结构定义，封装反射调用细节 |
| `dixinternal/util.go` | 工具函数：反射 Map/Slice 创建、输出类型处理、图构建 |
| `dixinternal/cycle-check.go` | 循环依赖检测逻辑 |
| `dixinternal/logger.go` | 日志系统集成 |
| `dixinternal/option.go` | 配置选项模式（如 `WithValuesNull`） |
| `dixglobal/` | 全局容器模块 |
| `dixcontext/` | Context 集成模块 |
| `dixhttp/` | HTTP 可视化模块 |

## 5. 线程安全性

**重要**：`Dix` 容器设计上**不是线程安全的**。用户不应在同一个容器实例上并发调用 `Provide`/`Inject`。这是为了性能而做出的设计决策，因为依赖注入通常发生在应用初始化阶段（单线程）。

对于并发场景，考虑：
1. 在启动并发操作前完成所有 `Provide` 调用
2. 每个 goroutine 使用独立的容器实例
3. 如需要，使用外部同步机制包装容器访问

## 6. 总结

`dix` 是一个功能完备的 Go 依赖注入容器。它通过反射牺牲了一定的运行时性能（但在初始化阶段通常可接受），换取了极大的开发灵活性。

设计亮点：
*   充分利用 Go 语言特性（多返回值处理 error、结构体字段标签等）
*   优雅支持集合类型注入
*   丰富的扩展生态（全局容器、context 集成、Web 可视化）
*   安全 API 变体支持优雅的错误处理

这使得 `dix` 成为一个完整的 Go 应用依赖注入解决方案。
