# dix 项目指引

## 适用范围

- 本文件是仓库级 always-on 指引，适用于整个 `dix` 工作区。
- 如存在更细分的 `*.instructions.md`，以细分规则作为补充，而非互相冲突。

## 技术栈与目标

- 语言：Go（见 `go.mod`，当前 `go 1.24`）。
- 项目类型：依赖注入框架（DI），核心能力包括 Provider 注册、依赖解析、循环检测、trace 诊断与可视化（`dixhttp`）。
- 修改实现时优先保持公开 API 与行为兼容，避免无关重构。

## 首选开发命令

- 测试：`task test`
- 静态检查：`task vet`
- Lint：`task lint`
- Web 示例：`task web-demo`

如需直接使用 Go 命令：

- `go test ./...`
- `go vet ./...`

## 架构边界（修改前先定位）

- `dix.go`：对外入口 API（创建容器、Provide/Inject 相关能力）。
- `dixinternal/`：核心实现（provider、inject、cycle-check、diag、logger、option）。
- `dixtrace/`：内存 trace 存储与查询。
- `dixhttp/`：HTTP 可视化与 trace API。
- `dixcontext/`：与 `context.Context` 集成。
- `dixglobal/`：全局容器便捷封装。

## 项目特有约定

- 错误优先通过 `TryProvide` / `TryInject` 路径返回，避免在新增逻辑中引入不必要 panic。
- 涉及 trace/诊断行为时，注意与环境变量约定保持一致（如 `DIX_TRACE_DI`、`DIX_DIAG_FILE`、`DIX_TRACE_FILE`）。
- 优先补充或复用现有测试文件（尤其 `dixinternal/*_test.go`、`dixtrace/*_test.go`、`dixhttp/server_runtime_test.go`）。

## 实施原则（对 AI 代理）

- 仅做与任务直接相关的最小改动，避免顺手大改。
- 保持现有命名风格与目录组织，不随意移动包结构。
- 新增行为时优先补测试，再改实现。
- 变更对外行为时，同步更新 `README.md`/`README_zh.md` 与对应模块 README（如 `dixhttp/README*.md`）。

## 常见坑点

- `task test` 当前默认覆盖 `dixinternal/...`，若改动到其他包需补充执行 `go test ./...` 进行全量验证。
- 项目包含 `go.work`，处理示例工程时注意模块工作区上下文。

## 参考文件

- `README.md`
- `README_zh.md`
- `Taskfile.yml`
- `dixinternal/dix.go`
- `dixinternal/provider.go`
- `dixinternal/diag_file.go`
- `dixtrace/trace.go`
- `dixhttp/server.go`
