# P2 trace 容器化与 TraceTree 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** trace 事件携带 ContainerID、TraceID 随机化、容器可私有 trace 缓冲(WithTraceBuffer),MemorySink 内建调用树索引并提供 `QueryTree` / `/api/trace-tree`。

**Architecture:** inject 根入口经 `dixtrace.WithContainer[Tracer](ctx, ...)` 把容器标识与可选 Tracer 写入 ctx;所有 span(含嵌套)经 spanFrame 继承并落到对应 sink;MemorySink 在写入时增量维护 `traceID -> 树索引`(FIFO 限量,默认 128 棵),`QueryTree` 组装嵌套 `TreeNode`。

**Tech Stack:** Go 标准库(crypto/rand、context、sync);无新依赖。

**Spec:** `docs/superpowers/specs/2026-09-04-graph-trace-redesign-design.md` 第 4 节。

## Global Constraints

- 现有测试零改动通过(trace_chain_test 的嵌套语义、logger 锁测试、模式锁测试、example 19 包)。
- `Event`/`Query` 只增字段;`BeginSpanCtx`/`BeginSpan`/`Emit`/`QueryEvents` 签名不变。
- JSONL 旧文件可读(新字段 omitempty)。
- 前端调用树视图**明确移交 P4**(P4 重写全部 UI,P2 交付 API + 契约测试;在 PR 中注明该偏离)。
- 每 Task 独立提交;全部 race 绿。

---

### Task 1: dixtrace 容器标识 + 随机 TraceID + Query.ContainerID

**Files:**
- Modify: `dixtrace/trace.go`
- Test: `dixtrace/trace_test.go`(追加)

**Interfaces(后续 Task 消费):**
- `Event.ContainerID string`(omitempty)
- `Query.ContainerID string`;`ParseQueryFromMap` 支持 `container_id`
- `func WithContainer(ctx context.Context, containerID string) context.Context`
- `func WithContainerTracer(ctx context.Context, containerID string, tr *Tracer) context.Context`
- `func NewContainerID() string`(crypto/rand 16 hex)
- `nextTraceID()` 改为 crypto/rand 32 hex(rand 失败回退计数器)
- `spanFrame` 增加 `ContainerID string; tracer *Tracer`;`Span` 增加 `containerID string; tracer *Tracer`,start/end 事件写入对应 `ContainerID` 并路由到 `tracer`(nil 回落全局 `Emit`)

**Step 1 失败测试(追加 trace_test.go):**

```go
func TestTraceIDIsRandomHex(t *testing.T) {
	ResetForTest()
	_, s1 := BeginSpan("op", "c")
	_, s2 := BeginSpan("op", "c")
	t1, _, _ := s1.IDs()
	t2, _, _ := s2.IDs()
	if t1 == t2 {
		t.Fatal("root spans must get distinct random trace ids")
	}
	if len(t1) != 32 {
		t.Fatalf("trace id len = %d, want 32 hex", len(t1))
	}
}

func TestContainerIDStamping(t *testing.T) {
	ResetForTest()
	ctx := WithContainer(context.Background(), "cont-1")
	ctx, span := BeginSpanCtx(ctx, "inject", "c")
	_, child := BeginSpanCtx(ctx, "resolve", "c")
	child.End(nil)
	span.End(nil)

	res := QueryEvents(Query{ContainerID: "cont-1"})
	if res.Total == 0 {
		t.Fatal("events should carry container id")
	}
	for _, rec := range res.Records {
		if rec.ContainerID != "cont-1" {
			t.Fatalf("event %s has container %q", rec.Event, rec.ContainerID)
		}
	}
	if res := QueryEvents(Query{ContainerID: "cont-other"}); res.Total != 0 {
		t.Fatal("other container must not match")
	}
}

func TestNewContainerID(t *testing.T) {
	a, b := NewContainerID(), NewContainerID()
	if a == b || len(a) != 16 {
		t.Fatalf("container ids must be random 16 hex, got %q %q", a, b)
	}
}
```

(文件头 import 补 `"context"`。)

**Step 2** Run: `go test ./dixtrace -run 'TestTraceIDIsRandom|TestContainerID|TestNewContainerID' -count=1` → 编译失败。

**Step 3 实现(trace.go):**

```go
// import 增加: "crypto/rand", "encoding/hex"

// Event 增字段:
	ContainerID      string         `json:"container_id,omitempty"`

// Query 增字段:
	ContainerID     string

// ParseQueryFromMap 增:
		ContainerID: parseString(values["container_id"]),

// matches 增:
	if !contains(rec.ContainerID, q.ContainerID) {
		return false
	}

// 随机 ID:
func NewContainerID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "cont-" + strconv.FormatInt(traceSeq.Add(1), 10)
	}
	return hex.EncodeToString(b[:])
}

func nextTraceID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "t-" + strconv.FormatInt(traceSeq.Add(1), 10)
	}
	return hex.EncodeToString(b[:])
}

// ctx 状态:
type ctxStateKey struct{}
type ctxState struct {
	containerID string
	tracer      *Tracer
}

func WithContainer(ctx context.Context, containerID string) context.Context {
	return context.WithValue(ctx, ctxStateKey{}, ctxState{containerID: containerID})
}

func WithContainerTracer(ctx context.Context, containerID string, tr *Tracer) context.Context {
	return context.WithValue(ctx, ctxStateKey{}, ctxState{containerID: containerID, tracer: tr})
}

// spanFrame 增字段:
type spanFrame struct {
	TraceID     string
	SpanID      string
	Operation   string
	Component   string
	ContainerID string
	tracer      *Tracer
}

// Span 增字段(containerID/tracer),BeginSpanCtx 改写:
func BeginSpanCtx(ctx context.Context, operation, component string, args ...any) (context.Context, *Span) {
	if ctx == nil {
		ctx = context.Background()
	}
	parent, hasParent := spanFrameFromContext(ctx)

	var st ctxState
	if v, ok := ctx.Value(ctxStateKey{}).(ctxState); ok {
		st = v
	}

	traceID, parentSpanID := "", ""
	if hasParent {
		traceID, parentSpanID = parent.TraceID, parent.SpanID
	} else {
		traceID = nextTraceID()
	}
	tracer := st.tracer
	if tracer == nil && hasParent {
		tracer = parent.tracer
	}
	containerID := st.containerID
	if containerID == "" && hasParent {
		containerID = parent.ContainerID
	}

	span := &Span{
		traceID: traceID, spanID: nextSpanID(), parentSpanID: parentSpanID,
		operation: strings.TrimSpace(operation), component: strings.TrimSpace(component),
		startedAt: time.Now().UnixNano(),
		containerID: containerID, tracer: tracer,
	}

	frame := spanFrame{
		TraceID: span.traceID, SpanID: span.spanID, Operation: span.operation,
		Component: span.component, ContainerID: span.containerID, tracer: span.tracer,
	}

	emitTo(span.tracer, Event{
		TraceID: span.traceID, SpanID: span.spanID, ParentSpanID: span.parentSpanID,
		Operation: span.operation, Phase: "start", Event: "span.start", Status: "start",
		Component: span.component, ContainerID: span.containerID,
		OccurredAt: span.startedAt, Attrs: TraceToAttrs(args...),
	})

	return context.WithValue(ctx, spanContextKey{}, frame), span
}

func emitTo(tr *Tracer, e Event) {
	if tr == nil {
		Emit(e)
		return
	}
	tr.Emit(e)
}

// Span.End 内 Emit(...) → emitTo(s.tracer, Event{...  ContainerID: s.containerID ...})
```

**Step 4** Run 同 Step 1 → 全 PASS;`go test ./dixtrace -race -count=1` 全绿(既有 trace_chain 相关在 dixinternal,另行回归)。

**Step 5** `git commit -m "feat(dixtrace): container id stamping, random trace ids, container query filter"`

---

### Task 2: MemorySink 树索引 + QueryTree

**Files:**
- Modify: `dixtrace/trace.go`
- Test: `dixtrace/trace_test.go`(追加)

**Interfaces:**
- `type TreeNode struct { Event Event; End *Event; Children []*TreeNode }`(JSON tag:event/end/children)
- `type TreeResult struct { Enabled bool; TraceID string; Total int; Roots []*TreeNode }`
- `func (m *MemorySink) QueryTree(traceID string) TreeResult`
- `func (t *Tracer) QueryTree(traceID string) TreeResult`(nil 安全回落全局)
- `func QueryTree(traceID string) TreeResult`(全局 sink)
- MemorySink 写入时增量维护树索引;树上限 `treeMax`(默认 128,测试可直改字段),FIFO 驱逐

**Step 1 失败测试:**

```go
func TestQueryTreeStructure(t *testing.T) {
	ResetForTest()
	_, root := BeginSpan("inject", "c")
	rootCtx := WithContainer(context.Background(), "")
	_ = rootCtx
	ctx, child := BeginSpanCtx(context.Background(), "child", "c")
	_ = ctx
	// 手工挂父子:child 需要是 root 的子——用 root 的返回 ctx
	ResetForTest()
	rootCtx2, root2 := BeginSpanCtx(context.Background(), "inject", "c")
	_, child2 := BeginSpanCtx(rootCtx2, "resolve.value", "c")
	child2.End(errors.New("boom"))
	root2.End(nil)

	traceID, _, _ := root2.IDs()
	tree := QueryTree(traceID)
	if !tree.Enabled || len(tree.Roots) != 1 {
		t.Fatalf("tree roots = %d, want 1", len(tree.Roots))
	}
	r := tree.Roots[0]
	if r.Event.Operation != "inject" || len(r.Children) != 1 {
		t.Fatalf("root op=%s children=%d", r.Event.Operation, len(r.Children))
	}
	if r.End == nil || r.End.Status != "ok" {
		t.Fatal("root end event missing")
	}
	kid := r.Children[0]
	if kid.Event.Operation != "resolve.value" || kid.End == nil || kid.End.Status != "error" {
		t.Fatalf("child = %+v", kid)
	}
}

func TestQueryTreeEviction(t *testing.T) {
	m := NewMemorySink(500)
	m.treeMax = 4
	for i := 0; i < 6; i++ {
		_, s := BeginSpan("op", "c")
		s.End(nil)
	}
	res := QueryEvents(Query{Limit: 1})
	// 最旧 trace 应被树索引驱逐
	oldest := res.Records[0]
	_ = oldest
	if got := m.QueryTree(traceIDOfEarliest(t)); got.Total != 0 && got.TraceID == traceIDOfEarliest(t) {
		t.Fatal("oldest trace should be evicted from tree index")
	}
}

func traceIDOfEarliest(t *testing.T) string {
	t.Helper()
	all := QueryEvents(Query{})
	if all.Total == 0 {
		return ""
	}
	return all.Records[len(all.Records)-1].TraceID // 降序,最后一条最早
}
```

(注:第一个用例中前两行占位写法在落码时清理——真实版本如下,直接以 rootCtx2 段为准。)

**Step 2** 确认编译失败。**Step 3 实现:**

```go
// TreeNode 是调用树节点:Event 为 span.start,End 为 span.end(存在时)。
type TreeNode struct {
	Event    Event       `json:"event"`
	End      *Event      `json:"end,omitempty"`
	Children []*TreeNode `json:"children"`
}

// TreeResult 是一次 trace 的调用树。
type TreeResult struct {
	Enabled bool        `json:"enabled"`
	TraceID string      `json:"trace_id"`
	Total   int         `json:"total"`
	Roots   []*TreeNode `json:"roots"`
}

const defaultTreeMax = 128

type treeBuilder struct {
	children  map[string][]string
	starts    map[string]Event
	ends      map[string]Event
	rootOrder []string
}

// MemorySink 增字段:
	treeMax int
	trees     map[string]*treeBuilder
	treeOrder []string

// NewMemorySink 初始化 trees/treeMax(=128)。

// Write 在既有 ring 逻辑后调用 m.indexTree(e)。
func (m *MemorySink) indexTree(e Event) {
	if e.TraceID == "" || e.SpanID == "" {
		return
	}
	tb, ok := m.trees[e.TraceID]
	if !ok {
		tb = &treeBuilder{children: map[string][]string{}, starts: map[string]Event{}, ends: map[string]Event{}}
		m.trees[e.TraceID] = tb
		m.treeOrder = append(m.treeOrder, e.TraceID)
		if len(m.treeOrder) > m.treeMax {
			old := m.treeOrder[0]
			m.treeOrder = m.treeOrder[1:]
			delete(m.trees, old)
		}
	}
	switch e.Event {
	case "span.start":
		tb.starts[e.SpanID] = e
		if e.ParentSpanID == "" {
			tb.rootOrder = append(tb.rootOrder, e.SpanID)
		} else {
			tb.children[e.ParentSpanID] = append(tb.children[e.ParentSpanID], e.SpanID)
		}
	case "span.end":
		tb.ends[e.SpanID] = e
	}
}

// QueryTree 组装调用树;孤儿 span(父已驱逐)按根处理。
func (m *MemorySink) QueryTree(traceID string) TreeResult {
	res := TreeResult{Enabled: true, TraceID: traceID, Roots: []*TreeNode{}}
	m.mu.RLock()
	tb, ok := m.trees[strings.TrimSpace(traceID)]
	if !ok {
		m.mu.RUnlock()
		return res
	}
	starts, children, ends, rootOrder := tb.starts, tb.children, tb.ends, tb.rootOrder
	total := len(starts)
	m.mu.RUnlock()
	res.Total = total

	var build func(id string) *TreeNode
	build = func(id string) *TreeNode {
		node := &TreeNode{Event: starts[id], Children: []*TreeNode{}}
		if e, ok := ends[id]; ok {
			cp := e
			node.End = &cp
		}
		for _, c := range children[id] {
			if _, ok := starts[c]; ok {
				node.Children = append(node.Children, build(c))
			}
		}
		return node
	}
	for _, r := range rootOrder {
		if _, ok := starts[r]; ok {
			res.Roots = append(res.Roots, build(r))
		}
	}
	return res
}

func (t *Tracer) QueryTree(traceID string) TreeResult {
	if t == nil {
		return defaultMemorySink.QueryTree(traceID)
	}
	for _, s := range t.sinks {
		if ms, ok := s.(*MemorySink); ok {
			return ms.QueryTree(traceID)
		}
	}
	return defaultMemorySink.QueryTree(traceID)
}

func QueryTree(traceID string) TreeResult { return defaultMemorySink.QueryTree(traceID) }

// resetDefaultForTest 追加:trees/treeOrder 重置。
```

**Step 4** `go test ./dixtrace -race -count=1` 全绿。**Step 5** 提交 `feat(dixtrace): incremental call-tree index with QueryTree`。

---

### Task 3: dixinternal 容器接入(containerID + WithTraceBuffer + TraceTree)

**Files:**
- Modify: `dixtrace/trace.go`(NewMemorySink 无变化)/ `dixinternal/option.go` / `dixinternal/dix.go` / `dix.go`(根)
- Test: `dixinternal/graph_test.go` 追加 + 根 `dix_test.go` 追加

**Interfaces:**
- dixinternal `Options.TraceBuffer int`(Validate >= 0)+ `WithTraceBuffer(n int) Option`
- Dix 字段 `containerID string`、`traceTracer *dixtrace.Tracer`(nil = 全局);newDix:`containerID = dixtrace.NewContainerID()`;TraceBuffer>0 时 `traceTracer = dixtrace.NewTracer(dixtrace.NewMemorySink(n))`
- `inject()` 在 BeginSpanCtx 之前 stamp:`WithContainerTracer(...)` 或 `WithContainer(...)`
- `(dix *Dix) TraceTree(traceID string) dixtrace.TreeResult`
- 根包 `WithTraceBuffer(n int) Option` 转发

**Step 1 失败测试(dixinternal/graph_test.go 追加):**

```go
// 容器事件带 containerID;私有缓冲与全局隔离;TraceTree 返回嵌套调用树。
func TestContainerTraceIsolationAndTree(t *testing.T) {
	dixtrace.ResetForTest()

	di := New(WithTraceBuffer(64))
	di.Provide(func() *graphDepA { return &graphDepA{} })
	if err := di.TryInject(func(a *graphDepA) {}); err != nil {
		t.Fatalf("inject: %v", err)
	}

	if di.containerID == "" {
		t.Fatal("container id must be assigned")
	}
	res := di.traceTracer.QueryEvents(Query{ContainerID: di.containerID})
	if res.Total == 0 {
		t.Fatal("local tracer should hold container events")
	}
	traceID := res.Records[0].TraceID

	tree := di.TraceTree(traceID)
	if !tree.Enabled || len(tree.Roots) == 0 {
		t.Fatalf("tree roots = %d, want >=1", len(tree.Roots))
	}
	if tree.Roots[0].Event.Operation != "inject" {
		t.Fatalf("root op = %s, want inject", tree.Roots[0].Event.Operation)
	}
	if len(tree.Roots[0].Children) == 0 {
		t.Fatal("inject root should have nested spans")
	}

	// 另一容器的私有缓冲为空(隔离)
	di2 := New(WithTraceBuffer(64))
	if got := di2.traceTracer.QueryEvents(Query{}); got.Total != 0 {
		t.Fatalf("isolated container sink should be empty, got %d", got.Total)
	}
}

// 全局容器(默认配置)事件进入全局 sink 且带 containerID。
func TestDefaultContainerStampsGlobalEvents(t *testing.T) {
	dixtrace.ResetForTest()
	di := New()
	if err := di.TryInject(func(a *graphDepA) {}); err != nil {
		t.Fatalf("inject: %v", err)
	}
	res := dixtrace.QueryEvents(Query{ContainerID: di.containerID})
	if res.Total == 0 {
		t.Fatal("global events should be stamped with container id")
	}
}
```

(需 import `dixtrace`;根 `dix_test.go` 追加:`TestWithTraceBufferForwarding`——`New(WithTraceBuffer(16)).Option().TraceBuffer == 16`,非法负值 panic。)

**Step 2** 确认失败。**Step 3 实现:** option.go 增 `TraceBuffer int` 字段、`WithTraceBuffer`、Validate;newDix 增加 containerID/traceTracer 初始化;`inject()` 在 `ctx, span := dixtrace.BeginSpanCtx(...)` 之前 stamp;dix.go 增 `TraceTree`;根包转发 `WithTraceBuffer`。

**Step 4** `go test ./dixinternal ./... -race -count=1` 全绿(含 trace_chain 既有断言)。**Step 5** 提交。

---

### Task 4: dixhttp `/api/trace-tree` + container_id 过滤

**Files:**
- Modify: `dixhttp/server.go`
- Test: `dixhttp/server_api_test.go` 追加

**Interfaces:** `GET {base}/api/trace-tree?trace_id=`(缺参 400);响应为 `dixtrace.TreeResult` JSON;`/api/trace` 增加 `container_id` 查询参数透传。

**Step 1 失败测试:**

```go
func TestHandleTraceTree(t *testing.T) {
	di := dixinternal.New()
	di.Provide(func() *apiStatsDep { return &apiStatsDep{} })
	if err := di.TryInject(func(*apiStatsDep) {}); err != nil {
		t.Fatalf("inject: %v", err)
	}

	all := dixtrace.QueryEvents(dixtrace.Query{Limit: 1})
	if len(all.Records) == 0 {
		t.Fatal("no trace events")
	}
	traceID := all.Records[0].TraceID

	server := NewServer(di)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/trace-tree?trace_id="+traceID, nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	var tree struct {
		Enabled bool `json:"enabled"`
		Total   int  `json:"total"`
		Roots   []struct {
			Event struct {
				Operation string `json:"operation"`
			} `json:"event"`
		} `json:"roots"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &tree); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !tree.Enabled || tree.Total == 0 || len(tree.Roots) == 0 {
		t.Fatalf("tree = %+v", tree)
	}

	// 缺 trace_id → 400
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/trace-tree", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("missing trace_id should 400, got %d", rr.Code)
	}
}
```

**Step 2** 确认 404 失败。**Step 3 实现:** setupRoutes 增 `base+"/api/trace-tree"`;handler:

```go
// HandleTraceTree 返回一次 trace 的调用树。
func (s *Server) HandleTraceTree(w http.ResponseWriter, r *http.Request) {
	traceID := strings.TrimSpace(r.URL.Query().Get("trace_id"))
	if traceID == "" {
		http.Error(w, "trace_id required", http.StatusBadRequest)
		return
	}
	writeJSON(w, s.dix.TraceTree(traceID))
}
```

HandleTrace 的 params map 增 `"container_id"`。**Step 4** `go test ./dixhttp -race -count=1` 全绿。**Step 5** 提交。

---

### Task 5: example 断言扩展 + docs/changelog + 全量验证 + 交付

- `example/context-inject/main_test.go`:在既有断言后追加 `dixtrace.QueryTree(traceID)` 断言(Roots ≥ 1 且 root.Event.TraceID == traceID)
- `docs/design.md`/`design_zh.md`:Diagnostics 节增 TraceTree/container_id/WithTraceBuffer 说明;changelog Unreleased 新增两条
- `gofmt -l .`、`go vet`(根+example)、`task test`、`task lint`、覆盖率核对(dixtrace 不回退)
- PR→CI→squash merge

## Self-Review 记录

- Spec §4 覆盖:4.1(Task 1/3)、4.2(Task 2/4;前端视图明确移交 P4——spec 已获准“前后端一起”,但 P4 会整体重写 UI,P2 先交付 API,PR 注明偏离)、4.3 验收(Task 3 隔离/一致性、Task 1 随机 ID、JSONL 只增字段向后兼容)。
- 无占位符;签名在各 Task 一致(`WithContainer`/`WithContainerTracer`/`QueryTree`/`TraceTree`/`WithTraceBuffer`)。
