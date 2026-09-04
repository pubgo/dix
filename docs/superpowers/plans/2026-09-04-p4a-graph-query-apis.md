# P4a 大规模检索与分层 API 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 提供服务端检索与分层查询 API——`/api/search`(过滤检索)、`/api/modules`(模块级聚合+跨模块依赖)、`/api/ego`(以任意类型为中心的 N 跳邻域子图),并升级 `/api/stats`(概览:模块数、resolved TopN、慢/错误 provider)。这是 P4b(五视图前端)与渲染 spike 的数据地基。

**Architecture:** 查询编排在 dixinternal(需同时读 Graph 与 providerStats,纯 Graph 方法不够):新文件 `dixinternal/graph_query.go`,全部走 `graph.mu.RLock` 只读投影;dixhttp 薄 handler 直接 JSON 化。检索为线性扫描(P1 决策:倒排等 P4b 定义查询形状与规模数据后再升级)。

**Tech Stack:** Go 标准库。

**Spec:** `docs/superpowers/specs/2026-09-04-graph-trace-redesign-design.md` 第 6 节(6.1/6.2/6.3/6.4 的服务端部分)。

## Global Constraints

- 既有测试零改动通过;新端点纯增量。
- state 过滤语义:`instantiated`=类型已有对象节点;`error`=provider 最近一次执行出错;`slow`=最近耗时超过容器 SlowProviderThreshold。
- `depth` 缺省 2、上限 10;`direction` ∈ {both,deps,dependents} 缺省 both;`limit` 缺省 50、上限 500。
- 每 Task 独立提交,全量 race 绿。

---

### Task 1: dixinternal 查询层

**Files:**
- Create: `dixinternal/graph_query.go`
- Test: `dixinternal/graph_query_test.go`

**Interfaces(dixhttp 消费):**
- `type SearchHit struct { ID uint32; Kind, Label, Pkg, Group, State, Provider string }`(json tags: id/kind/label/pkg/group/state/provider)
- `(dix *Dix) SearchNodes(q, kind, module, state string, limit int) []SearchHit` — kind ∈ type|provider|object;module 为 pkg 前缀匹配;q 为 label/函数名小写包含
- `type ModuleInfo struct { Name string; TypeCount, ProviderCount, ObjectCount int; DependsOn []string }`(json: name/type_count/provider_count/object_count/depends_on)
- `(dix *Dix) ModuleGraph() []ModuleInfo`(按名称排序;Declared 边跨模块时计入 DependsOn,去重)
- `type GraphView struct { Nodes []SearchHit; Edges []GraphEdge }`;`type GraphEdge struct { From, To string }`(类型 label)
- `(dix *Dix) EgoGraph(center string, depth int, direction string) GraphView` — center 为类型 Label;BFS 声明边;边集为邻域内实际声明的边
- `(dix *Dix) ResolvedTopN(n int) []ResolvedCount`;`type ResolvedCount struct { Type, Count }`(json: type/count,count int64)
- `(dix *Dix) ProblemProviders() (slow []string, errored []string)` — 基于 providerStats.LastError / LastDuration > SlowProviderThreshold,返回函数名去重排序

**Step 1 失败测试(graph_query_test.go):**

测试骨架(三个用例):

```go
func TestSearchNodesFilters(t *testing.T) {
	di := New()
	di.Provide(func() *QAService { return &QAService{} })
	di.Provide(func() *QARepo { return &QARepo{} })
	_ = di.TryInject(func(s *QAService, r *QARepo) {})

	// q 过滤
	hits := di.SearchNodes("qaservice", "", "", "", 50)
	if len(hits) == 0 || hits[0].Label != "*dixinternal.QAService" {
		t.Fatalf("q filter failed: %+v", hits)
	}
	// kind 过滤
	hits = di.SearchNodes("", "provider", "", "", 50)
	for _, h := range hits {
		if h.Kind != "provider" {
			t.Fatalf("kind filter failed: %+v", h)
		}
	}
	// state=instantiated 只返回已有对象的类型
	hits = di.SearchNodes("", "type", "", "instantiated", 50)
	if len(hits) != 2 { // *QAService 与 *QARepo 均已实例化
		t.Fatalf("instantiated filter: %+v", hits)
	}
	// limit 生效
	if hits := di.SearchNodes("", "", "", "", 1); len(hits) != 1 {
		t.Fatal("limit must be honored")
	}
}

func TestModuleGraphAggregation(t *testing.T) {
	di := New()
	di.Provide(func(r *QARepo) *QAService { return &QARepo{}) } // 同包:无跨模块边
	// 见实现:同包内声明边不计入 DependsOn;跨模块用不同 pkg 的类型(测试内构造两个包不可行,
	// 故以 (anonymous) 与 dixinternal 分组验证分组本身成立)
	_ = di
}

func TestEgoGraphDepthAndDirection(t *testing.T) {
	di := New()
	// 链:C -> B -> A(声明依赖)
	di.Provide(func(*QADepB) *QADepA { return &QADepA{} })
	di.Provide(func(*QADepC) *QADepB { return &QADepB{} })

	// center=*QADepA depth=1 deps:只含 A、B
	view := di.EgoGraph("*dixinternal.QADepA", 1, "deps")
	labels := viewLabels(view)
	if !containsStr(labels, "*dixinternal.QADepA") || !containsStr(labels, "*dixinternal.QADepB") || containsStr(labels, "*dixinternal.QADepC") {
		t.Fatalf("deps view wrong: %v", labels)
	}
	// depth=2 both:三层都在
	view = di.EgoGraph("*dixinternal.QADepA", 2, "both")
	if !containsStr(viewLabels(view), "*dixinternal.QADepC") {
		t.Fatalf("both depth=2 should include C: %v", viewLabels(view))
	}
	// dependents 方向从 A 反向:只有 B(以及再往上 C)
	view = di.EgoGraph("*dixinternal.QADepA", 1, "dependents")
	if !containsStr(viewLabels(view), "*dixinternal.QADepB") {
		t.Fatalf("dependents view should include B: %v", viewLabels(view))
	}
}

func TestResolvedTopNAndProblemProviders(t *testing.T) {
	di := New(WithSlowProviderThreshold(time.Millisecond))
	di.Provide(func() *QASlow { time.Sleep(5 * time.Millisecond); return &QASlow{} })
	di.Provide(func() (*QABroken, error) { return nil, errors.New("x") })
	_ = di.TryInject(func(s *QASlow) {})
	_ = di.TryInject(func(b *QABroken) {})

	top := di.ResolvedTopN(5)
	if len(top) == 0 || top[0].Type != "*dixinternal.QASlow" || top[0].Count != 1 {
		t.Fatalf("top resolved = %+v", top)
	}
	slow, broken := di.ProblemProviders()
	if len(slow) == 0 || len(broken) == 0 {
		t.Fatalf("slow=%v broken=%v", slow, broken)
	}
}
```

(测试类型 QAService/QARepo/QADepA/B/C/QASlow/QABroken 在测试文件内声明;`TestModuleGraphAggregation` 落码时以同包分组断言 TypeCount/ProviderCount 正确、DependsOn 为空为准——跨模块场景由 dixhttp demo(多包)覆盖。)

**Step 2** 确认编译失败 → **Step 3 实现 graph_query.go** → **Step 4** `go test ./dixinternal -run 'TestSearchNodes|TestModuleGraph|TestEgoGraph|TestResolvedTopN' -race -count=1` 全绿 → **Step 5** 提交 `feat: server-side graph query APIs (search/modules/ego/topN)`。

---

### Task 2: dixhttp 端点 + stats 升级

**Files:**
- Modify: `dixhttp/server.go`(路由 + 3 个新 handler + StatsData 增字段)
- Test: `dixhttp/server_api_test.go` 追加

**Interfaces:**
- `GET /api/search?q=&kind=&module=&state=&limit=` → `[]SearchHit`
- `GET /api/modules` → `[]ModuleInfo`
- `GET /api/ego?center=&depth=&direction=` → `GraphView`(缺 center → 400)
- `StatsData` 增字段:`modules int`、`top_resolved []{type,count}`、`slow_providers []string`、`error_providers []string`

**Step 1 失败测试(server_api_test.go 追加):**

```go
func TestHandleSearchModulesEgo(t *testing.T) {
	di := dixinternal.New()
	di.Provide(func() *apiStatsDep { return &apiStatsDep{} })
	if err := di.TryInject(func(*apiStatsDep) {}); err != nil {
		t.Fatalf("inject: %v", err)
	}
	server := NewServer(di)

	// search
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/search?q=apistats&kind=type", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("search status %d", rr.Code)
	}
	var hits []dixinternal.SearchHit
	if err := json.Unmarshal(rr.Body.Bytes(), &hits); err != nil || len(hits) == 0 {
		t.Fatalf("search hits = %v err = %v", hits, err)
	}

	// modules
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/modules", nil))
	var modules []dixinternal.ModuleInfo
	if err := json.Unmarshal(rr.Body.Bytes(), &modules); err != nil || len(modules) == 0 {
		t.Fatalf("modules = %v err = %v", modules, err)
	}

	// ego:缺 center → 400;有效 center → 视图非空
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/ego", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("ego without center should 400, got %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/ego?center=*dixinternal.apiStatsDep&depth=1&direction=deps", nil))
	var view dixinternal.GraphView
	if err := json.Unmarshal(rr.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode ego: %v", err)
	}

	// stats 增字段
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/stats", nil))
	var stats StatsData
	if err := json.Unmarshal(rr.Body.Bytes(), &stats); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if stats.Modules == 0 || stats.TopResolved == nil {
		t.Fatalf("stats upgrade missing: %+v", stats)
	}
}
```

**Step 2** 确认 404 失败。**Step 3 实现 handler/路由/StatsData 增字段**。**Step 4** `go test ./dixhttp -race -count=1` 全绿。**Step 5** 提交。

---

### Task 3: 文档 + changelog + 全量验证 + 交付

- dixhttp/README.md 增三个路由文档;design 双语一句概览;changelog Unreleased 新增一条
- `gofmt -l .`、vet、`task test`、`task lint`、覆盖率核对
- PR → CI → squash merge

## Self-Review

- Spec 6.1/6.2/6.3/6.4 的**服务端部分**全覆盖;前端五视图与渲染 spike 属 P4b(P4 开工决策点,spec 允许)。
- 签名一致:SearchHit/ModuleInfo/GraphView/GraphEdge 在 Task 1 定义、Task 2 消费。
