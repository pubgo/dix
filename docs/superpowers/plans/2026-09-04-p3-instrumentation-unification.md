# P3 埋点订阅化与 LLM 通道删除 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 删除独立 LLM 诊断通道(env/emitLLMDiagnosticLine/kind:llm,无过渡期);`logDITrace` 双轨埋点全部改为容器事件(`emitDIEvent`)进入 tracer sink 流,console(`di_trace`)与 diag file 成为订阅者,输出契约不变。

**Architecture:** `dixinternal/di_event.go`(新):`emitDIEvent` 把点事件(Event{Operation:"di", Event:name, Attrs})路由到容器 tracer/全局;`consoleDISink`(DIX_TRACE_DI 逐条 gating,`logger.Info("di_trace "+event, ...)`)+ `diagTraceSink`(转 `emitDiagFileTraceEvent`)在 dixinternal init 时挂到全局 tracer、newDix 时挂到容器私有 tracer。错误记录(`recordRecentErrorWithContext`)与 diag error record 保持原样。

**Tech Stack:** Go 标准库。

**Spec:** `docs/superpowers/specs/2026-09-04-graph-trace-redesign-design.md` 第 5 节。

## Global Constraints

- 锁测试口径:`TestDITraceLogsInInjectFlow` / `TestDITraceLogsInProvideFlow` 断言 `di_trace <event>` 子串必须继续通过;`TestDiagFileConfiguredCollectsTraceErrorAndLLM` 改写为 trace+error 两类(llm 删除)。
- `DIX_LLM_DIAG_MODE` 直接删除,无 no-op 过渡;`DIX_TRACE_DI` / `DIX_DIAG_FILE` / `DIX_TRACE_FILE` 语义不变。
- LLM 消费契约:stderr slog 行已带 `dix.error_type=`/`dix.root_cause=`/`dix.hint=` 结构化属性,保留;JSONL 与 /api/errors 同。
- 每个 Task 独立提交,全量 race 绿。

---

### Task A: 删除 LLM 通道

**Files:**
- Modify: `dixinternal/dix.go`(删 `emitLLMDiagnosticLine` 与 `recordRecentErrorWithContext` 中的调用)
- Modify: `dixinternal/logger.go`(删 llmDiagMode* 常量、`currentLLMDiagMode`、`isLLMDiagMachineOnlyMode`、`shouldEmitLLMDiagnosticLine`;`createDefaultLogger` 去掉 machine-only 分支)
- Modify: `dixinternal/diag_file.go`(删 `emitDiagFileLLMRecord`)
- Modify: `example/http/main.go`(删 `isMachineDiagMode`/`configureExampleLogOutput` 及 main 中的调用)
- Modify: `dixhttp/server.go`(HandleDiagnostics 注释 kind 枚举改 `trace|error`)
- Delete/Modify tests: `TestCurrentLLMDiagMode`、`TestLLMDiagOnlyModeSuppressesHumanLogs`、`TestTerminalLLMDiagnosticLine` 删除;`TestDiagFileConfiguredCollectsTraceErrorAndLLM` 改写为 `TestDiagFileConfiguredCollectsTraceAndError`
- Modify: README.md/README_zh.md(诊断表删 `DIX_LLM_DIAG_MODE` 行)、docs/design*.md(LLM records 表述)、dixhttp/README.md(kind 枚举)、changelog

**Steps:**
1. 删除上述实现与测试(测试改写先行:`TestDiagFileConfiguredCollectsTraceAndError` 断言 trace+error 各 ≥1 条且全文无 `"llm"` kind)。
2. `go build ./... && go test ./... -count=1` 全绿。
3. 全仓 `grep -rn "LLM_DIAG\|llmDiag\|emitLLMDiagnosticLine\|kind:llm\|kind=trace|error|llm" --include="*.go" --include="*.md"` 仅允许历史 changelog 版本文件残留。
4. 提交 `refactor: remove dedicated LLM diagnostics channel`。

---

### Task B: logDITrace → emitDIEvent 事件流 + 订阅者

**Files:**
- Create: `dixinternal/di_event.go`
- Modify: `dixinternal/logger.go`(删 `logDITrace`,保留 `shouldTraceDependencyFlow`)、`dixinternal/dix.go`(95 处调用点)、`dixinternal/diag_file.go`(trace record 写入改由订阅者触发后,`emitDiagFileTraceEvent` 保留供订阅者调用)
- Modify: `dixtrace/trace.go`(`func AddDefaultSink(s Sink)`;`Tracer.AddSink`)
- Modify: `dixinternal/newDix`(容器私有 tracer 追加同样的订阅 sink)
- Test: `dixinternal/logger_test.go`(既有三个 di_trace/diag 锁测试应零改动或仅重命名通过)

**Interfaces:**
- `(dix *Dix) emitDIEvent(event string, args ...any)`:`dixtrace.EmitTo(dix.traceTracer, Event{Operation:"di", Event:event, ContainerID:dix.containerID, Attrs:TraceToAttrs(args...), OccurredAt:time.Now().UnixNano()})`
- `consoleDISink.Write(e Event)`:env 未开直接返回;`logger.Info("di_trace "+e.Event, kvArgs(e.Attrs)...)`
- `diagTraceSink.Write(e Event)`:`emitDiagFileTraceEvent(e.Event, kvArgs(e.Attrs)...)`
- `kvArgs(m map[string]any) []any`:key,value 交替展开(key 排序保证 console 属性顺序稳定)
- `dixtrace.AddDefaultSink(s)` / `(t *Tracer) AddSink(s)`

**Steps:**
1. dixtrace 增 `AddDefaultSink`/`AddSink`。
2. 新建 di_event.go(emitDIEvent + 两个 sink + kvArgs + `installDISinks(tr *Tracer)`);dixinternal `init()` 对全局 tracer 安装;newDix 对私有 tracer 安装。
3. `sed -i 's/\blogDITrace(/dix.emitDIEvent(/g' dixinternal/dix.go`;grep 校验无残留;手工核查无 `dix` 不在作用域的位置。
4. 删除 logger.go 中 `logDITrace` 定义;`emitDIEvent` 内部不再区分 env(console sink 自己 gating)。
5. 跑锁测试:`go test ./dixinternal -run 'TestDITraceLogs|TestShouldTrace|TestDiagFileConfigured' -count=1` → 全绿(改写后的 DiagFile 测试)。
6. 全量:`go test ./... -race -count=1`、example、`task lint`。
7. 提交 `refactor: route di_trace point events through the tracer stream`。

---

### Task C: 文档同步 + 交付

- docs/design.md / design_zh.md:埋点统一与 LLM 通道移除说明;README 双语诊断表(如含 DIX_LLM_DIAG_MODE 行删除);dixhttp/README /api/diagnostics kind 枚举。
- changelog Unreleased:`变更` 追加埋点订阅化与 LLM 通道删除(引用 spec 决策)。
- `task test` + `task lint` + 覆盖率核对 → PR → CI → squash merge。

## Self-Review

- Spec §5 覆盖:订阅化(Task B)、LLM 直接删除(Task A)、dix.go 瘦身(随调用点替换发生,如实汇报行数)、格式契约(锁测试口径)。
- 签名一致:emitDIEvent/consoleDISink/diagTraceSink/AddSink 各处一致;`EmitTo` 复用 P2 已有导出。
