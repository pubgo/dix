# dix 示例索引(代码即文档)

每个子目录是一个可独立运行的示例:

```bash
cd example/<名称> && go run .
```

各示例注释遵循统一模板:**【功能】【原理】【运行】【预期输出】**。
示例所演示的行为契约由 `dixinternal/pattern_lock_test.go` 用断言锁死——
注释与实现不一致时,以锁测试为准并修正注释。

## 基础注入模式

| 示例 | 功能 | 涉及语义 |
| --- | --- | --- |
| [func](./func/) | 函数类型作为依赖 | 同类型多 provider:全部执行;单值取最后注册者;切片按注册顺序聚合 |
| [struct-in](./struct-in/) | 结构体字段注入 | 导出字段递归解析;未导出/基础类型字段跳过 |
| [struct-out](./struct-out/) | 结构体多输出 | Out 结构体按导出字段拆分注册;各字段共享同一底层实例 |
| [map](./map/) | Map 命名空间注入 | 多 provider 贡献不同 key 合并;同 key 后者覆盖;nil 值跳过;空 key 归 default 组 |
| [list](./list/) | 列表聚合注入 | 按 provider 注册顺序聚合;切片元素顺序保持 |

## 行为语义

| 示例 | 功能 | 涉及语义 |
| --- | --- | --- |
| [lazy](./lazy/) | 惰性求值 | provider 仅在被需要时执行;产物缓存,后续注入不重复执行 |
| [cycle](./cycle/) | 循环依赖检测 | 注入前环检测,报错含化简后的环路径 |
| [inject_method](./inject_method/) | DixInject 方法注入 | 方法按字母序执行;先于字段注入;参数解析规则同函数注入 |
| [test-return-error](./test-return-error/) | TryProvide/TryInject 错误处理 | 根因保留在错误链(errors.Is 可判定);失败 provider 不缓存、会重试 |

## 容错与综合

| 示例 | 功能 | 涉及语义 |
| --- | --- | --- |
| [map-nil](./map-nil/) | 缺失 map 依赖容错 | 默认解析为非 nil 空 map;单值缺失仍报错 |
| [list-nil](./list-nil/) | 缺失 slice 依赖容错 | 默认解析为非 nil 空切片;`WithRejectEmptyCollections` 可改为报错 |
| [handler](./handler/) | 常见装配模式 | 同类型依赖单例共享;单值与命名空间 map 并存 |
| [http](./http/) | dixhttp 可视化 | 端到端综合示例,含运行时诊断与错误场景触发 |

## 配套测试

每个示例目录都有 `main_test.go`,直接断言其演示函数的行为(13/13 覆盖):
`main.go` 把演示逻辑抽取为可返回结果的函数,`main()` 只负责打印——测试与文档互为印证。

- 行为锁:`dixinternal/pattern_lock_test.go`(核心容器语义)+ 各示例 `main_test.go`(示例级契约)
- 全量单测:`task test`(根模块 + 本模块);可视化演示:`task web-demo`
