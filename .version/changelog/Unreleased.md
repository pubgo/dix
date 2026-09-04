# [Unreleased]

> 推荐维护方式：
>
> - 建议通过 agent 提示词执行：`/changelog-maintenance draft|release`

## 新增

- 运行时依赖图 Graph（P1）：Provide 增量维护声明边/执行计数，环检测改读增量图；`/api/search`、`/api/modules`、`/api/ego` 服务端检索与分层查询；dixhttp 依赖数据按图版本快照缓存（热请求零反射）
- trace 容器化（P2）：事件携带随机 container_id 实现多容器隔离，TraceID 改为随机 32 hex；`WithTraceBuffer(n)` 私有 trace 缓冲；`Dix.TraceTree` 与 `/api/trace-tree` 嵌套调用树查询，`/api/trace` 支持 container_id 过滤
- 大规模演示容器 example/http：10 个域模块 + 120 个泛型插件/工作器，约 190 provider / 190+ 对象 / 12 模块
- 五视图实验版 UI（`/next`）：概览/依赖图/检索/调用链/诊断，本地静态资源零 CDN

## 修复

- 聚合查询（map/slice 注入）元素为裸 struct 时 fail-fast 报错，不再静默返回错误结果；环路径输出确定性化（起点为环成员类型名字典序最小者）
- dixhttp 依赖数据缓存状态机缺陷：stats/packages 端点预热输入缓存后 `/api/dependencies` 返回字面量 `null`（依赖图视图空白）；缓存刷新时同步构建全量投影，加回归测试
- 依赖图全局视图在数据未就绪时空指针崩溃；检索"图中查看"对 provider/object 类别不生效
- `task web-demo` 单文件编译失败（改为整包 `go run -C example ./http`）；`GetFnName` 对零值 reflect.Value panic

## 变更

- DI 点事件统一走 tracer 事件流（console `di_trace` 与 diag file 成为订阅者，输出契约不变）；直接移除独立 LLM 诊断通道（`DIX_LLM_DIAG_MODE`、stderr `DIX_LLM_DIAG` 行、diag `kind:llm`），`error_type`/`root_cause`/`hint` 结构化字段在全部出口保留
- 默认 Web UI 恢复 v2.0.2 交互版本（完整依赖图、双视图、节点详情、Mermaid/SVG 导出、Trace 诊断），五视图新版 UI 移至 `/next` 实验入口
- example/http 升级为大规模演示容器；任务 `task web-demo` 改为整包构建

## 文档

- 新增 Graph/Trace/调用链路重设计 spec（docs/superpowers/specs/）与 P1–P4a 实施计划（docs/superpowers/plans/）
- design 文档双语同步依赖图与 trace 新架构；README 双语 Option 表、dixhttp README 路由与 UI 信息架构更新
