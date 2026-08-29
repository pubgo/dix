# Dix 项目审计报告

[English](./audit.md)

> **时点快照。** 下述数据描述审计日期的代码树状态，会随后续开发漂移；请将其视为带日期的报告，而非实时看板。

**审计日期**：2026-08-29
**分支**：v2 @ 8746133（v2.0.1 + 发版后修复）
**审计方式**：自动化代码审查

---

## 📊 项目概览

| 指标 | 数值 |
|--------|-------|
| 核心代码行数 | 约 8,600 行（不含示例） |
| 示例模块 | 约 1,300 行，13 个可运行示例（独立 Go 模块） |
| 测试函数 | 127 个（全部通过 ✅） |
| 测试覆盖率（`go test -cover`） | dixinternal 69.7% · dix 48.4% · dixhttp 51.1% · dixtrace 66.0% · dixcontext 94.7% · dixglobal 81.8% |
| CI | GitHub Actions：每次 push/PR 运行 `go test -race`（含 atomic 覆盖率，含 example 模块）与 `golangci-lint` ✅ |
| Lint（`golangci-lint`） | 0 个问题 ✅ |

---

## ✅ 优势

### 1. 架构设计
- 清晰的模块分层：`dix` → `dixinternal` → `dixglobal`/`dixcontext`/`dixhttp`/`dixtrace`
- 关注点分离良好；`dixhttp` 直接接受 `*dix.Dix`（签名不泄漏内部类型）
- 核心 API 简洁：`New()`、`Provide()`、`Inject()`，并提供安全的 `Try*` 变体

### 2. 功能完备性
- ✅ 循环依赖检测（缓存依赖图 + DFS，环路径裁剪）
- ✅ 多种注入模式：函数、结构体、Map、List
- ✅ 命名空间隔离
- ✅ 方法注入（`DixInject` 前缀）
- ✅ 错误处理（Provider 返回 error）
- ✅ 安全 API：`TryProvide`/`TryInject`
- ✅ 注册快速失败：不支持的 Provider 输入/输出类型在 Provide 阶段即被拒绝
- ✅ Provider 超时控制（超时的 Provider 不会被重新执行）
- ✅ 运行时诊断：Provider 统计、最近错误、追踪、JSONL 诊断文件

### 3. 代码质量
- 完善的错误处理与 panic 恢复，携带结构化上下文（阶段、根因、提示）
- 详细的日志输出（slog）
- 类型安全的泛型 API

### 4. 可视化模块
- 现代前端技术栈（Tailwind + Alpine.js + vis-network）
- 功能丰富：模糊搜索、深度控制、双向追踪、分组规则
- RESTful API 设计良好，包含 runtime-stats/errors/diagnostics/trace

### 5. 文档与测试基础设施
- 中英双语的 README 与设计文档，已按当前架构刷新
- CI 编译独立的 `example` 模块（该模块的回归无法再静默合入）

---

## ⚠️ 待改进项

### 1. 测试覆盖率不均衡
```
dixinternal: 69.7%  ✅
dixcontext:  94.7%  ✅
dixglobal:   81.8%  ✅
dixtrace:    66.0%  🟡
dixhttp:     51.1%  🟡
dix:         48.4%  🟡
```
**建议**：优先补齐根包装包与 dixhttp handler 的测试。

### 2. 性能
- 基于反射的解析对启动阶段足够，但长运行场景下热路径注入的性能未量化。
**建议**：为 Inject 解析深度与大规模 Provider 集合补充基准测试。

### 3. API 演进
- `Options.Merge` 已移除（其语义导致调用级 Option 失效）；`dixinternal` 仍可被外部导入——长期建议下沉到真正的 Go `internal/` 目录。

---

## 🔄 与同类项目对比

### vs uber-go/dig

| 特性 | dix | dig |
|---------|-----|-----|
| 基础 DI | ✅ | ✅ |
| 循环依赖检测 | ✅（缓存依赖图） | ✅ |
| Map/List 注入 | ✅ | ✅ |
| 命名空间 | ✅ | ✅（via Group） |
| 方法注入 | ✅ | ❌ |
| 返回错误的注册/调用 | ✅（`TryProvide`/`TryInject`；`Provide`/`Inject` panic） | ✅（`Provide`/`Invoke` 返回 error） |
| Web 可视化 | ✅ | ❌ |
| 泛型 API | ✅ | ❌ |
| 结构体自动分解 | ✅ | ✅（via fx） |

> 注：dig 的 `Provide`/`Invoke` 设计上返回 error；dix 的 `Provide`/`Inject` panic
> 但提供 `TryProvide`/`TryInject` 等价物。此行已修正——早期版本误写 dig 无安全 API。

### vs google/wire

| 特性 | dix | wire |
|---------|-----|------|
| 运行时 DI | ✅ | ❌（编译期） |
| 无代码生成 | ✅ | ❌ |
| 动态注册 | ✅ | ❌ |
| 性能 | 中 | 高 |
| 调试 | 更容易（运行时诊断、可视化） | 更难 |
| 可视化 | ✅ | ❌ |

### 定位

- **dix**：适合需要运行时灵活性、诊断能力与可视化的大型项目
- **dig**：适合较简单的运行时 DI 需求
- **wire**：适合性能敏感、依赖关系静态的应用

---

## 📋 总结

`dix` 是一个设计良好、功能完备的 Go 依赖注入框架。相比上次审计（2026 年 1 月）：CI 已覆盖 race/覆盖率/lint 并包含 example 模块，所有模块测试覆盖率均不再为零，线程安全语义已文档化，Provider 注册快速失败，运行时诊断（stats/errors/trace/诊断文件）已落地。剩余的主要投入方向是根包与 dixhttp 的测试覆盖率，以及性能基准测试。
