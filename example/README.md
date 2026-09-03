# dix 示例索引(代码即文档)

每个子目录是一个可独立运行的示例:

```bash
cd example/<名称> && go run .
```

示例命名与所演示特性对应:`inject-*` 为注入模式,`provide-*` 为 provider 能力,`context-*` 为 Context 集成。
注释遵循统一模板:**【功能】【原理】【运行】【预期输出】**,且"预期输出"逐个实测核对。

各示例演示的行为契约由两层测试锁定——注释与实现不一致时,以测试为准并修正注释:

- `dixinternal/pattern_lock_test.go` / `api_lock_test.go`:核心容器语义
- 各示例目录的 `main_test.go`:示例抽取出的演示函数本身(19/19 全覆盖)

## 注入模式(inject-* / provide-*)

| 示例 | 功能 | 涉及语义 |
| --- | --- | --- |
| [inject-func](./inject-func/) | 函数类型作为依赖 | 同类型多 provider:全部执行;单值取最后注册者;切片按注册顺序聚合 |
| [inject-struct](./inject-struct/) | 结构体字段注入 | 导出字段递归解析;未导出/基础类型字段跳过 |
| [inject-method](./inject-method/) | DixInject 方法注入 | 方法按字母序执行;先于字段注入;参数解析规则同函数注入 |
| [inject-generic](./inject-generic/) | InjectT/InjectTContext 泛型注入 | 分配并填充结构体一步到位;非 struct 类型 fail-fast |
| [context-inject](./context-inject/) | InjectContext/TryInjectContext | 注入携带的 ctx 决定 trace 链路归属,事件可按 trace_id 查询 |
| [inject-map](./inject-map/) | Map 命名空间注入 | 多 provider 贡献不同 key 合并;同 key 后者覆盖;nil 值跳过;空 key 归 default 组 |
| [inject-list](./inject-list/) | 列表聚合注入 | 按 provider 注册顺序聚合;切片元素顺序保持 |
| [inject-map-list](./inject-map-list/) | map[string][]T 分组聚合 | 同 key 切片按注册顺序拼接;聚合元素必须为指针/接口/函数 |
| [provide-multi-output](./provide-multi-output/) | 结构体多输出 | Out 结构体按导出字段拆分注册;各字段共享同一底层实例 |

## 运行语义

| 示例 | 功能 | 涉及语义 |
| --- | --- | --- |
| [lazy](./lazy/) | 惰性求值 | provider 仅在被需要时执行;产物缓存,后续注入不重复执行 |
| [cycle](./cycle/) | 循环依赖检测 | 注入前环检测,报错含化简后的环路径(起点可能轮换) |
| [singleton](./singleton/) | 单例共享 | 同类型依赖容器级单例;单值与命名空间 map 并存 |
| [error-handling](./error-handling/) | TryProvide/TryInject 错误处理 | 根因保留在错误链(errors.Is 可判定);失败 provider 不缓存、会重试 |
| [timeout](./timeout/) | provider 超时与慢告警 | 超时使注入失败且 provider 不再重试;慢 provider 输出告警 |
| [empty-collections](./empty-collections/) | 缺失集合依赖容错 | 默认解析为非 nil 空集合;`WithRejectEmptyCollections` 反转;单值缺失恒报错 |

## 模块

| 示例 | 功能 | 涉及语义 |
| --- | --- | --- |
| [global](./global/) | dixglobal 全局容器 | 免创建直接注册/注入;仅限启动期单协程使用 |
| [context-container](./context-container/) | dixcontext 容器随 Context 传递 | Create/Get/GetOrNil;调用链任意深度取容器注入 |
| [custom-logger](./custom-logger/) | SetLog 自定义日志 | dix 诊断日志接入应用日志体系 |
| [http](./http/) | dixhttp 可视化 | 端到端综合示例,含运行时诊断与错误场景触发 |

## 配套测试

- 行为锁:`dixinternal/pattern_lock_test.go`(核心语义)+ 各示例 `main_test.go`(示例级契约)
- 全量单测:`task test`(根模块 + 本模块);可视化演示:`task web-demo`
