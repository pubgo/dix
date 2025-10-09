# Dix 依赖注入框架设计文档

## 1. 概述 (Overview)

`dix` 是一个基于 Go 语言反射机制（Reflection）实现的轻量级依赖注入（Dependency Injection, DI）框架。它参考了 `dig` 的设计理念，旨在通过自动化的依赖解析和生命周期管理，解耦组件间的依赖关系。

`dix` 的核心是对能够寻址的类型（Addressable Types）进行依赖注入管理，主要包括 `func`、`ptr`（指针）和 `interface`。

核心特性包括：
*   **构造函数注入**：通过 `Provide` 注册构造函数。
*   **结构体/字段注入**：通过 `Inject` 对结构体字段进行填充。
*   **高级类型支持**：支持 `Interface`、`Map`、`Slice` 等集合类型的自动聚合注入。
*   **循环依赖检测**：内置图算法检测循环依赖。
*   **可视化**：支持生成 DOT 格式的依赖图。

## 2. 核心架构 (Core Architecture)

`dix` 的核心是一个名为 `Dix` 的容器结构体，它维护了提供者（Providers）的注册表和已实例化对象（Objects）的缓存。

### 2.1 支持的类型 (Supported Types)

`dix` 严格限制了可作为依赖项管理的类型范围，核心原则是**仅支持可寻址或引用类型**。这种设计确保了依赖关系的稳定性和对象生命周期的可控性。

*   **Pointer (指针)**:
    *   这是最常用的类型，通常指向一个结构体实例（如 `*Service`）。
    *   指针保证了在整个应用生命周期中，组件是单例的（Singleton），且状态共享。
*   **Interface (接口)**:
    *   支持将具体实现绑定到接口定义（如 `Service` 接口由 `*ServiceImpl` 实现）。
    *   这是实现依赖倒置原则（DIP）的关键，使得组件依赖于抽象而非具体实现。
*   **Func (函数)**:
    *   函数本身是一等公民，可以作为依赖被注入。
    *   常用于注入工厂函数、中间件或回调逻辑。

*注意：基础类型（如 `int`, `string`, `bool`）不能直接作为依赖项注入，必须封装在上述类型中（通常是结构体字段）。*

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

1.  **注册 (Provide)**: 用户注册构造函数 -> 解析函数签名（输入/输出） -> 存入 `providers` 表。
2.  **调用 (Inject)**: 用户请求注入对象 -> 递归查找依赖 -> 执行 Provider 函数 -> 缓存结果 -> 填充目标。

## 3. 详细设计 (Detailed Design)

### 3.1 提供者注册 (Provider Registration)

`Provide` 方法负责将构造函数注册到容器中。

#### 3.1.1 输入参数类型 (Input Types)
Provider 函数的参数声明了该组件的依赖项：

*   **Func / Ptr / Interface**:
    *   **行为**: 声明为直接依赖。
    *   **解析**: 容器会在已注册的 Provider 中查找匹配该类型的唯一实例。
*   **Struct**:
    *   **行为**: **递归依赖注入 (Recursive Dependency Injection)**。
    *   **解析**: 框架会递归遍历该结构体的所有**导出字段**。如果字段类型是支持的类型（Func/Ptr/Interface），容器会自动查找并注入对应的实例。这允许通过定义一个配置结构体来聚合多个依赖项，简化函数签名。
*   **Slice (`[]T`)**:
    *   **行为**: 声明为聚合依赖（List）。
    *   **解析**: 容器查找所有能提供类型 `T` 的 Provider，并将它们在 **默认分组 (default group)** 下的结果汇总成切片。
*   **Map (`map[string]T`)**:
    *   **行为**: 声明为聚合依赖（Map）。
    *   **解析**: 容器查找所有能提供类型 `T` 的 Provider，利用其返回的 Key（默认为 "default"）汇总成 Map。如果同一 Key 有多个值，取最后一个。

#### 3.1.2 返回值类型 (Output Types)
Provider 函数的返回值定义了它向容器提供的组件。

**约束 (Constraints)**:
Provider 函数必须返回 **1 个或 2 个** 值。
*   如果返回 1 个值：该值即为提供的组件。
*   如果返回 2 个值：第二个值必须是 `error` 类型。如果 `error` 不为 nil，容器将停止初始化并报错。

支持的具体类型如下：

*   **Func / Ptr / Interface**:
    *   **行为**: 注册为该类型的标准 Provider。
*   **Slice (`[]T`)**:
    *   **行为**: 注册为类型 `T` 的列表 Provider。
    *   **效果**: 允许一个 Provider 一次性提供多个 `T` 实例，或者多个 Provider 共同向 `[]T` 注入点贡献数据。
*   **Map (`map[string]T`)**:
    *   **行为**: 注册为类型 `T` 的映射 Provider。
    *   **效果**: 允许 Provider 指定组件的 Key。
*   **Struct**:
    *   **行为**: **递归自动分解 (Recursive Auto-Flattening)**。
    *   **效果**: 框架会递归遍历结构体的所有**导出字段**。如果字段类型是支持的类型（Func/Ptr/Interface），该字段的值会被分别注册为对应类型的 Provider。这使得一个构造函数可以同时提供多个不同的服务组件。

### 3.2 依赖解析与注入 (Resolution & Injection)

注入过程由 `Inject` 或内部的 `getValue` 驱动，采用**惰性求值（Lazy Evaluation）**策略。

#### 3.2.1 注入目标类型 (Injection Targets)

`Inject` 函数支持对结构体指针或函数进行注入。

**1. 结构体指针 (Struct Pointer)**
当传入 `&MyStruct{}` 时，框架会扫描其字段进行注入：
*   **Func / Ptr / Interface**: 查找并注入对应类型的实例。
*   **Struct**: **递归注入 (Recursive Injection)**。框架会深入结构体内部，继续对该字段的成员进行注入。这允许定义嵌套的配置或组件对象。
*   **Slice / Map**: 执行聚合注入逻辑。对于 Slice，仅收集 **默认分组 (default group)** 下的实例。详细策略见 3.2.2。
*   **方法注入**: 自动扫描对象中以 `DixInject` 为前缀的方法并执行注入（Setter 注入的一种变体）。

**2. 函数 (Function)**
当传入一个函数时（如 `dix.Inject(func(a *A, b *B){ ... })`）：
*   参数列表被视为依赖项。
*   解析逻辑与 Provider 的输入参数完全一致（见 3.1.1）。
*   常用于执行初始化逻辑或启动钩子。

#### 3.2.2 内部存储与解析策略 (Internal Storage & Resolution Strategy)

当存在同一类型的多个 Provider 时，`dix` 使用分组（Group/Key）机制来管理这些实例。底层的存储结构实际上是 `map[group][]value`，其中 `group` 是标签（默认为 "default"），用于区分不同的实例集合。

针对不同的依赖声明方式，容器采用以下解析策略：

1.  **单值依赖 (`T`)**:
    *   仅查找 **默认分组 ("default")**。
    *   取该分组下的 **最后一个值**。
2.  **列表依赖 (`[]T`)**:
    *   仅查找 **默认分组 ("default")**。
    *   取该分组下的 **所有值**。
3.  **映射依赖 (`map[string]T`)**:
    *   查找 **所有分组**。
    *   对于每个分组，取其 **最后一个值**。
4.  **完全映射依赖 (`map[string][]T`)**:
    *   查找 **所有分组**。
    *   取每个分组下的 **所有值**。这是获取该类型所有实例的最完整方式。

### 3.3 循环依赖检测 (Cycle Detection)

为了防止无限递归，`dix` 在执行注入前或过程中会构建依赖图。

*   **算法**: 基于深度优先搜索 (DFS)。
*   **实现**: `dixinternal/cycle-check.go` 中的 `detectCycle` 函数。
*   **逻辑**: 构建 `map[reflect.Type]map[reflect.Type]bool` 的邻接表，遍历图寻找回边。如果发现循环，立即报错并打印循环路径。

### 3.4 错误处理 (Error Handling)

使用了 `github.com/pubgo/funk/v2` 库进行错误包装。
*   **上下文丰富**: 错误信息中包含了堆栈跟踪（Stack Trace）、Provider 函数名、参数类型等详细信息，便于调试。
*   **Panic 捕获**: 在执行用户代码（Provider 函数）时，使用 `defer recovery` 机制捕获 Panic，防止整个应用崩溃，并将其转化为 Error 返回。

### 3.5 可视化 (Observability)

`dix` 提供了 `Graph()` 方法，利用 `DotRenderer` 生成 Graphviz DOT 格式的文本。
*   **Provider Graph**: 展示 Provider 函数之间的调用关系。
*   **Object Graph**: 展示实际实例化对象之间的引用关系。

## 4. 模块划分 (Module Breakdown)

| 文件 | 职责 |
| :--- | :--- |
| `dix.go` | 核心逻辑实现，包括 `newDix` 初始化、`inject` 递归注入流程、`handleProvide` 注册逻辑。 |
| `api.go` | 对外暴露的 API 接口 (`New`, `Provide`, `Inject`, `Graph`)。 |
| `provider.go` | 定义 `providerFn` 结构，封装反射调用的细节，处理函数调用的输入输出转换。 |
| `util.go` | 工具函数集合，包括反射创建 Map/Slice (`makeMap`, `makeList`)，输出类型处理 (`handleOutput`)，以及图构建逻辑。 |
| `cycle-check.go` | 专门负责循环依赖检测的逻辑。 |
| `renderer.go` | 负责将内部依赖关系渲染为 DOT 字符串。 |
| `option.go` | 配置选项模式实现（如 `WithValuesNull`）。 |

## 5. 总结

`dix` 是一个功能完备的 Go 依赖注入容器。它通过反射牺牲了一定的运行时性能（但在初始化阶段通常可接受），换取了极大的开发灵活性。其设计亮点在于对 Go 语言特性的充分利用（如多返回值处理 error，结构体字段标签等）以及对集合类型注入的优雅支持。
