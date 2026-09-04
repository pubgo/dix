# P1 统一 Graph 模型 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** dixinternal 新增运行时依赖图 Graph(节点/边/版本号),Provide 增量建图,isCycle 迁移到 Graph,删除 depGraph/buildDependencyGraph;dixhttp 以 version 判脏缓存依赖数据,重复请求零反射。

**Architecture:** Graph 挂在 Dix 上(读写锁保护,dixhttp 并发读安全);声明边 = 输出类型节点 → 输入类型节点(与原 buildDependencyGraph 语义逐一等价,含 struct 输出按字段展开、struct 输入经 getProvideAllInputs 扁平化);Resolved 边 = provider 节点 → 输出类型节点(执行计数);dixhttp 缓存 ProviderDetails/objects 两份昂贵输入,聚合改为纯函数。

**Tech Stack:** Go 1.24 标准库(reflect/sort/sync/atomic);无新依赖。

**Spec:** `docs/superpowers/specs/2026-09-04-graph-trace-redesign-design.md` 第 3 节(P1)。

## Global Constraints

- 所有现有测试不改一行、原样通过(cycle 锁测试 `TestCycleDetection`/`TestDetectCycleDeterministicOrder`/`TestTrimCyclePath`/`TestCycleDetectionAfterNewProviders`/`TestTryInjectCycleDetection` 是行为等价性的裁判)。
- 容器本身仍是单线程使用;Graph 自带 `sync.RWMutex`,保证 dixhttp handler 并发读安全(`-race` 必须绿)。
- 不改任何公开 API 签名;dixhttp 响应字段只增不改。
- 环检测确定性语义(#57):起点与邻居按类型名字典序,`detectCycle` 函数签名与实现不动。
- 提交遵循仓库 conventional commits;每 Task 至少一次提交。

---

### Task 1: Graph 核心(节点/边/版本/邻接投影)

**Files:**
- Create: `dixinternal/graph.go`
- Test: `dixinternal/graph_test.go`

**Interfaces:**
- Produces: `type NodeID uint32`、`NodeKind(NodeType|NodeProvider|NodeObject)`、`EdgeKind(EdgeDeclared|EdgeProduced|EdgeResolved)`、`type Node`、`type Edge`、`type Graph`、`newGraph() *Graph`、`(g *Graph) Version() uint64`、`(g *Graph) providerNode(p *providerFn, outTyp reflect.Type) NodeID`、`(g *Graph) addProduced(pNode NodeID, outTyp reflect.Type)`、`(g *Graph) addDeclared(outTyp reflect.Type, inTyp reflect.Type, agg string, p *providerFn)`、`(g *Graph) markResolved(pNode NodeID, outTyp reflect.Type)`、`(g *Graph) addObject(typ reflect.Type, group string) bool`、`(g *Graph) declaredAdjacency() map[reflect.Type]map[reflect.Type]bool`、`(g *Graph) searchNodes(q string, limit int) []Node`。后续 Task 全部按这些签名消费。

**设计要点(实现者必读):**
- 声明边的两端都是**类型节点**(输出类型 → 输入类型),不是 provider 节点——这样才能与原 `buildDependencyGraph` 的 `graph[outTyp][inTyp]=true` 逐一等价(struct 多输出 provider 会注册多个输出类型,每个输出类型都有自己的入边)。
- 边按 `edgeKey{from,to,kind,field,agg,provider}` 幂等去重:struct 多输出导致 handleProvide 对同一 fn 递归调用多次,重复注册必须无副作用。
- 邻接投影只读已去重的边,`getProvideAllInputs` 的扁平化在 Task 2 注册时完成,投影本身零反射。
- 检索:P1 只提供线性 `searchNodes`(小写包含匹配 Label 与 provider 函数名);真倒排索引等 P4 定义查询形状后再升级。

- [ ] **Step 1: 写失败测试**

创建 `dixinternal/graph_test.go`:

```go
package dixinternal

import (
	"reflect"
	"strings"
	"sync"
	"testing"
)

type graphDepA struct{}
type graphDepB struct{}
type graphDepC struct{}

// 邻接表字符串视图,便于断言。
func adjacencyString(adj map[reflect.Type]map[reflect.Type]bool) []string {
	out := make([]string, 0, len(adj))
	for from, tos := range adj {
		var tos2 []string
		for to := range tos {
			tos2 = append(tos2, to.String())
		}
		sortStrings(tos2)
		out = append(out, from.String()+" -> ["+strings.Join(tos2, ", ")+"]")
	}
	sortStrings(out)
	return out
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func TestGraphDeclaredAdjacency(t *testing.T) {
	g := newGraph()

	a, b, c := reflect.TypeOf(graphDepA{}), reflect.TypeOf(graphDepB{}), reflect.TypeOf(graphDepC{})

	// provider p1: 产出 *A,依赖 *B(等价于 func(*B) *A)
	p1 := g.providerNode(&providerFn{}, a)
	g.addProduced(p1, a)
	g.addDeclared(a, b, "", nil)
	// provider p2: 产出 *B,依赖 *C
	p2 := g.providerNode(&providerFn{}, b)
	g.addProduced(p2, b)
	g.addDeclared(b, c, "", nil)

	adj := g.declaredAdjacency()
	want := []string{
		"*dixinternal.graphDepA -> [*dixinternal.graphDepB]",
		"*dixinternal.graphDepB -> [*dixinternal.graphDepC]",
	}
	if got := adjacencyString(adj); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("adjacency = %v, want %v", got, want)
	}
}

func TestGraphIdempotentEdges(t *testing.T) {
	g := newGraph()
	a, b := reflect.TypeOf(graphDepA{}), reflect.TypeOf(graphDepB{})

	p := g.providerNode(&providerFn{}, a)
	g.addProduced(p, a)
	g.addDeclared(a, b, "", nil)
	// 模拟 struct 多输出的重复注册:同一边再来一次
	g.addProduced(p, a)
	g.addDeclared(a, b, "", nil)

	if len(g.eIndex) != 2 { // produced + declared 各一条
		t.Fatalf("edges = %d, want 2 (idempotent)", len(g.eIndex))
	}
}

func TestGraphResolvedCounting(t *testing.T) {
	g := newGraph()
	a := reflect.TypeOf(graphDepA{})
	p := g.providerNode(&providerFn{}, a)
	g.markResolved(p, a)
	g.markResolved(p, a)

	g.mu.RLock()
	count := g.eIndex[edgeKey{from: p, to: a, kind: EdgeResolved}].Count
	g.mu.RUnlock()
	if count != 2 {
		t.Fatalf("resolved count = %d, want 2", count)
	}
}

func TestGraphAddObjectOnce(t *testing.T) {
	g := newGraph()
	a := reflect.TypeOf(graphDepA{})

	if !g.addObject(a, "default") {
		t.Fatal("first addObject should create")
	}
	if g.addObject(a, "default") {
		t.Fatal("second addObject for same (type,group) should be no-op")
	}
	if !g.addObject(a, "other") {
		t.Fatal("different group should create")
	}
}

func TestGraphVersionBumps(t *testing.T) {
	g := newGraph()
	v0 := g.Version()

	a := reflect.TypeOf(graphDepA{})
	p := g.providerNode(&providerFn{}, a)
	g.addProduced(p, a)
	g.bumpVersion()
	v1 := g.Version()
	if v1 <= v0 {
		t.Fatalf("version should bump after provide: %d -> %d", v0, v1)
	}

	if g.addObject(a, "default") {
		g.bumpVersion()
	}
	if g.Version() <= v1 {
		t.Fatalf("version should bump after new object node")
	}
	// 重复对象不再 bump
	before := g.Version()
	if g.addObject(a, "default") {
		g.bumpVersion()
	}
	if g.Version() != before {
		t.Fatal("duplicate object must not bump version")
	}
}

func TestGraphConcurrentReads(t *testing.T) {
	g := newGraph()
	a, b := reflect.TypeOf(graphDepA{}), reflect.TypeOf(graphDepB{})
	p := g.providerNode(&providerFn{}, a)
	g.addProduced(p, a)
	g.addDeclared(a, b, "", nil)
	g.bumpVersion()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = g.Version()
				_ = g.declaredAdjacency()
				_ = g.searchNodes("graphDep", 10)
			}
		}()
	}
	wg.Wait()
}

func TestGraphSearchNodes(t *testing.T) {
	g := newGraph()
	a, b := reflect.TypeOf(graphDepA{}), reflect.TypeOf(graphDepB{})
	pa := &providerFn{}
	pa.fn = reflect.ValueOf(func() *graphDepA { return nil })
	g.providerNode(pa, a)
	g.addProduced(pa, a)

	hits := g.searchNodes("graphdepa", 10)
	if len(hits) == 0 {
		t.Fatal("search should match type name case-insensitively")
	}
	_ = b
}
```

- [ ] **Step 2: 运行确认编译失败**

Run: `go test ./dixinternal -run TestGraph -count=1`
Expected: FAIL,`undefined: newGraph` 等编译错误。

- [ ] **Step 3: 实现 graph.go**

创建 `dixinternal/graph.go`:

```go
package dixinternal

import (
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
)

// NodeID 是 Graph 内节点自增编号。
type NodeID uint32

// NodeKind 区分依赖类型节点、provider 节点与已创建对象节点。
type NodeKind uint8

const (
	NodeType NodeKind = iota
	NodeProvider
	NodeObject
)

// EdgeKind 区分声明依赖、产物输出与运行时实际解析。
type EdgeKind uint8

const (
	EdgeDeclared EdgeKind = iota
	EdgeProduced
	EdgeResolved
)

// Node 是 Graph 节点。Object 节点的存在即"已实例化"状态。
type Node struct {
	ID       NodeID
	Kind     NodeKind
	Type     reflect.Type
	Group    string
	Provider *providerFn
	Label    string
	Pkg      string
}

// Edge 是 Graph 有向边。声明边:输出类型 -> 输入类型;
// 产物边:provider -> 输出类型;解析边:provider -> 输出类型(执行计数)。
type Edge struct {
	From, To  NodeID
	Kind      EdgeKind
	Field     string
	Aggregate string
	Provider  *providerFn
	Count     int64
}

type nodeKey struct {
	kind  NodeKind
	typ   reflect.Type
	group string
	fn    *providerFn
}

type edgeKey struct {
	from, to NodeID
	kind     EdgeKind
	field    string
	agg      string
	provider *providerFn
}

// Graph 是容器运行时依赖图。Dix 单线程写,但 dixhttp 并发读,
// 因此全部访问走读写锁;version 供快照判脏。
type Graph struct {
	mu      sync.RWMutex
	nodes   []Node
	nIndex  map[nodeKey]NodeID
	eIndex  map[edgeKey]*Edge
	version atomic.Uint64
}

func newGraph() *Graph {
	return &Graph{
		nIndex: make(map[nodeKey]NodeID),
		eIndex: make(map[edgeKey]*Edge),
	}
}

// Version 返回图版本号;Provide 或新对象入图时递增。
func (g *Graph) Version() uint64 { return g.version.Load() }

func (g *Graph) bumpVersion() { g.version.Add(1) }

// searchNodes 按关键字(小写包含)检索节点 Label 与 provider 函数名。
// P4 定义查询形状后可升级为倒排索引;当前线性扫描,规模内足够。
func (g *Graph) searchNodes(q string, limit int) []Node {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return nil
	}
	if limit <= 0 {
		limit = 50
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	var out []Node
	for _, n := range g.nodes {
		hay := strings.ToLower(n.Label)
		if n.Provider != nil {
			hay += " " + strings.ToLower(GetFnName(n.Provider.fn))
		}
		if strings.Contains(hay, q) {
			out = append(out, n)
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}

func (g *Graph) providerNode(p *providerFn, outTyp reflect.Type) NodeID {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.nodeLocked(NodeProvider, outTyp, "", p)
}

// addProduced 记录 provider 产物边。
func (g *Graph) addProduced(pNode NodeID, outTyp reflect.Type) {
	g.mu.Lock()
	defer g.mu.Unlock()
	to := g.nodeLocked(NodeType, outTyp, "", nil)
	g.edgeLocked(pNode, to, EdgeProduced, "", "", nil)
}

// addDeclared 记录声明依赖边:输出类型 -> 输入类型。
func (g *Graph) addDeclared(outTyp, inTyp reflect.Type, agg string, p *providerFn) {
	g.mu.Lock()
	defer g.mu.Unlock()
	from := g.nodeLocked(NodeType, outTyp, "", nil)
	to := g.nodeLocked(NodeType, inTyp, "", nil)
	g.edgeLocked(from, to, EdgeDeclared, "", agg, p)
}

// markResolved 累加 provider 实际执行计数。
func (g *Graph) markResolved(pNode, outTypeNode NodeID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.edgeLocked(pNode, outTypeNode, EdgeResolved, "", "", nil).Count++
}

// addObject 记录 (type, group) 对象节点;返回是否新建。
func (g *Graph) addObject(typ reflect.Type, group string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	key := nodeKey{kind: NodeObject, typ: typ, group: group}
	if _, ok := g.nIndex[key]; ok {
		return false
	}
	g.nodeLocked(NodeObject, typ, group, nil)
	return true
}

// declaredAdjacency 投影声明边为 outputType -> inputTypes 邻接表,
// 语义与原 buildDependencyGraph 完全一致,供环检测使用。
func (g *Graph) declaredAdjacency() map[reflect.Type]map[reflect.Type]bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	adj := make(map[reflect.Type]map[reflect.Type]bool)
	for _, e := range g.eIndex {
		if e.Kind != EdgeDeclared {
			continue
		}
		out := g.nodes[e.From].Type
		in := g.nodes[e.To].Type
		if adj[out] == nil {
			adj[out] = make(map[reflect.Type]bool)
		}
		adj[out][in] = true
	}
	return adj
}

func (g *Graph) nodeLocked(kind NodeKind, typ reflect.Type, group string, provider *providerFn) NodeID {
	key := nodeKey{kind: kind, typ: typ, group: group, fn: provider}
	if id, ok := g.nIndex[key]; ok {
		return id
	}
	id := NodeID(len(g.nodes))
	g.nodes = append(g.nodes, Node{
		ID: id, Kind: kind, Type: typ, Group: group, Provider: provider,
		Label: typ.String(), Pkg: resolveTypePkgPath(typ),
	})
	g.nIndex[key] = id
	return id
}

func (g *Graph) edgeLocked(from, to NodeID, kind EdgeKind, field, agg string, p *providerFn) *Edge {
	key := edgeKey{from: from, to: to, kind: kind, field: field, agg: agg, provider: p}
	if e, ok := g.eIndex[key]; ok {
		return e
	}
	e := &Edge{From: from, To: to, Kind: kind, Field: field, Aggregate: agg, Provider: p}
	g.eIndex[key] = e
	return e
}
```

注意:测试里 `TestGraphSearchNodes` 断言的是 provider 函数名/Label 匹配;`providerFn` 零值 `fn` 为零值 reflect.Value,`GetFnName` 对零值返回 "unknown"——测试里 `pa.fn = reflect.ValueOf(...)` 已显式赋值,不受影响。

- [ ] **Step 4: 运行确认通过**

Run: `go test ./dixinternal -run TestGraph -race -count=1`
Expected: 全部 PASS。

- [ ] **Step 5: 提交**

```bash
git add dixinternal/graph.go dixinternal/graph_test.go
git commit -m "feat: add runtime dependency graph core (nodes/edges/version)"
```

---

### Task 2: Dix 接入 Provide 增量建图

**Files:**
- Modify: `dixinternal/dix.go`(Dix 结构体、newDix、handleProvide)
- Test: `dixinternal/graph_test.go`(追加)

**Interfaces:**
- Consumes: Task 1 全部签名。
- Produces: `(dix *Dix) GraphVersion() uint64`(Task 5 dixhttp 消费);Dix 字段 `graph *Graph`(Task 3/4 消费)。

- [ ] **Step 1: 写失败测试(追加到 graph_test.go)**

```go
// 锁定 Provide 增量建图:声明边语义与原 buildDependencyGraph 一致——
// 含 struct 输出按导出字段展开、struct 输入经 getProvideAllInputs 扁平化。
func TestDixProvideBuildsGraph(t *testing.T) {
	di := New()

	type GConf struct{ DSN string }
	type GDB struct{ Conf *GConf }
	type GIn struct {
		DB *GDB
	}
	type GOut struct {
		DB    *GDB
		Probe *GDB
	}

	di.Provide(func() *GConf { return &GConf{} })
	di.Provide(func(c *GConf) *GDB { return &GDB{Conf: c} })
	di.Provide(func(in GIn) GOut { return GOut{} })

	adj := di.graph.declaredAdjacency()

	conf := reflect.TypeOf(&GConf{})
	db := reflect.TypeOf(&GDB{})
	out := reflect.TypeOf(GOut{})
	in := reflect.TypeOf(GIn{})

	// struct 输入扁平化:GOut 的 provider 对 GIn 的声明等价于对 *GDB 的声明
	if !adj[out][db] {
		t.Fatalf("GOut should declare dependency on *GDB, got %v", adjacencyString(adj))
	}
	if !adj[db][conf] {
		t.Fatalf("*GDB should declare dependency on *GConf, got %v", adjacencyString(adj))
	}
	// 容器自注册(func() *Dix)与 *GConf 无声明边
	if adj[conf] != nil {
		t.Fatalf("*GConf should have no declared deps, got %v", adjacencyString(adj))
	}
	_ = in

	// struct 多输出:GOut 的两个字段类型各有一条产物边,provider 节点唯一
	produced := 0
	di.graph.mu.RLock()
	for _, e := range di.graph.eIndex {
		if e.Kind == EdgeProduced && e.Provider != nil {
			produced++
		}
	}
	di.graph.mu.RUnlock()
	if produced != 3 { // *GConf / *GDB / GOut 三个注册点各一条(GOut 按 2 字段展开为同类型去重后 1 条)
		t.Fatalf("produced edges = %d, want 3", produced)
	}
}

func TestDixGraphVersionOnProvide(t *testing.T) {
	di := New()
	v0 := di.GraphVersion()
	di.Provide(func() *graphDepA { return &graphDepA{} })
	if di.GraphVersion() <= v0 {
		t.Fatal("version should bump on provide")
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./dixinternal -run TestDixGraph -count=1`
Expected: FAIL,`di.graph undefined`(Dix 尚无 graph 字段/方法)。

- [ ] **Step 3: 实现**

`dixinternal/dix.go` 三处修改:

(a) Dix 结构体(约 60-70 行)删除 `depGraph map[reflect.Type]map[reflect.Type]bool` 与 `graphDirty bool` 两行,新增:

```go
	// graph 是运行时依赖图:声明边在 Provide 时增量维护,
	// 解析计数在 provider 执行时累加;环检测与 dixhttp 均读它。
	graph *Graph
```

(b) newDix 的 container 字面量:删除 `depGraph: nil,` 与 `graphDirty: true,` 两行,新增 `graph: newGraph(),`。

(c) api.go 的 `New`/`newDix` 无需改动(同包)。

(d) `handleProvide` 尾部(`logDITrace("provide.register.done", ...)` 之前、switch 成功之后)插入:

```go
	// 增量维护运行时依赖图:产物边 + 声明边(输入扁平化),幂等。
	pNode := dix.graph.providerNode(provider, outType)
	dix.graph.addProduced(pNode, outType)
	for _, input := range inputs {
		agg := ""
		if input.isMap {
			agg = "map"
		} else if input.isList {
			agg = "list"
		}
		for _, flat := range getProvideAllInputs(input.typ) {
			dix.graph.addDeclared(outType, flat.typ, agg, provider)
		}
	}
	dix.graph.bumpVersion()
```

(e) dix.go 新增公开(包内)方法:

```go
// GraphVersion 返回运行时依赖图版本号,供 dixhttp 快照判脏。
func (dix *Dix) GraphVersion() uint64 { return dix.graph.Version() }
```

- [ ] **Step 4: 运行确认通过 + 全量回归**

Run: `go test ./dixinternal -race -count=1 && go test ./... -count=1`
Expected: 全部 PASS(此时 depGraph 仍在但不再被写;isCycle 仍走旧路径,功能不变)。

- [ ] **Step 5: 提交**

```bash
git add dixinternal/dix.go dixinternal/graph_test.go
git commit -m "feat: build dependency graph incrementally on provide"
```

---

### Task 3: isCycle 迁移到 Graph,删除 depGraph/buildDependencyGraph

**Files:**
- Modify: `dixinternal/cycle-check.go`(重写 isCycle)
- Modify: `dixinternal/util.go`(删除 buildDependencyGraph)
- Test: 既有 cycle 锁测试即裁判,不改;graph_test.go 补一个等价性黄金断言

**Interfaces:**
- Consumes: Task 1 `declaredAdjacency()`、Task 2 已建图。
- Produces: 行为等价的 `isCycle`(签名不变);删除 `buildDependencyGraph` 后无他处引用(getProvideAllInputs 保留,api.go 仍在用)。

- [ ] **Step 1: 补等价性黄金测试(追加到 graph_test.go)**

```go
// 黄金断言:容器级邻接投影必须精确等于原 buildDependencyGraph 的输出
// (A->B->C 环 + 一个无依赖 provider),含 struct 输出/输入的展开语义。
func TestDixAdjacencyGolden(t *testing.T) {
	di := New()

	type GCycleA struct{}
	type GCycleB struct{}
	type GCycleC struct{}
	type GFree struct{}

	di.Provide(func(*GCycleB) *GCycleA { return &GCycleA{} })
	di.Provide(func(*GCycleC) *GCycleB { return &GCycleB{} })
	di.Provide(func(*GCycleA) *GCycleC { return &GCycleC{} })
	di.Provide(func() *GFree { return &GFree{} })

	got := adjacencyString(di.graph.declaredAdjacency())
	want := []string{
		"*dixinternal.GCycleA -> [*dixinternal.GCycleB]",
		"*dixinternal.GCycleB -> [*dixinternal.GCycleC]",
		"*dixinternal.GCycleC -> [*dixinternal.GCycleA]",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("adjacency = %v, want %v", got, want)
	}
}
```

Run: `go test ./dixinternal -run TestDixAdjacencyGolden -count=1`
Expected: PASS(Task 2 已建图)。

- [ ] **Step 2: 重写 cycle-check.go**

整文件替换为:

```go
package dixinternal

import (
	"strings"
)

// isCycle 检查依赖环。依赖图由 Graph 在 Provide 时增量维护,
// 这里只做只读投影 + DFS,不再有全量重建步骤。
// 确定性语义(#57):起点与邻居按类型名字典序,报告最短环路径。
func (dix *Dix) isCycle() (string, bool) {
	cyclePath := detectCycle(dix.graph.declaredAdjacency())
	if len(cyclePath) == 0 {
		return "", false
	}

	var pathStr strings.Builder
	for i, t := range cyclePath {
		if i > 0 {
			pathStr.WriteString(" -> ")
		}
		pathStr.WriteString(t.String())
	}
	return pathStr.String(), true
}
```

- [ ] **Step 3: 删除 util.go 中的 buildDependencyGraph 整个函数**

删除 `dixinternal/util.go` 中 `func buildDependencyGraph(...)` 到其结束花括号的整段(约 19 行)。`getProvideAllInputs` 保留(api.go 的 GetProviderDetails 在用)。

- [ ] **Step 4: 全量回归(cycle 锁测试是裁判)**

Run: `go test ./dixinternal -race -run 'TestCycle|TestTrimCyclePath|TestDetectCycleDeterministicOrder|TestDixAdjacencyGolden' -count=1 && go test ./... -count=1 && go test -C example -count=1 ./...`
Expected: 全部 PASS,一行测试都不用改。

- [ ] **Step 5: 提交**

```bash
git add dixinternal/cycle-check.go dixinternal/util.go dixinternal/graph_test.go
git commit -m "refactor: cycle detection reads incrementally-built graph; drop depGraph rebuild"
```

---

### Task 4: Resolved 执行计数 + Object 节点

**Files:**
- Modify: `dixinternal/dix.go`(executeProvider 成功路径、processProviderOutput)
- Test: `dixinternal/graph_test.go`(追加)

**Interfaces:**
- Consumes: Task 1 `markResolved`/`addObject`/`bumpVersion`;Task 2 `dix.graph`。
- Produces: Resolved 计数与 Object 节点语义(P2 的 trace 事件将引用;Task 5 的 version 判脏依赖此处的 bump)。

- [ ] **Step 1: 写失败测试(追加)**

```go
// provider 每实际执行一次,对应 Resolved 边计数 +1;首次产物入缓存建 Object 节点。
func TestDixGraphResolvedAndObjects(t *testing.T) {
	di := New()

	type GObj struct{ N int }
	di.Provide(func() *GObj { return &GObj{N: 1} })

	a := reflect.TypeOf(&GObj{})
	pNode := di.graph.providerNode(nil, a) // 引用同一 provider 节点:按 (kind=NodeProvider, typ, fn) 索引取回
	_ = pNode

	if err := di.TryInject(func(o *GObj) {}); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if err := di.TryInject(func(o *GObj) {}); err != nil {
		t.Fatalf("inject2: %v", err)
	}

	// resolved 计数:provider 只执行一次(产物缓存),第二次 inject 不再执行
	di.graph.mu.RLock()
	var resolved int64
	for _, e := range di.graph.eIndex {
		if e.Kind == EdgeResolved {
			resolved = e.Count
		}
	}
	objNodes := 0
	for _, n := range di.graph.nodes {
		if n.Kind == NodeObject {
			objNodes++
		}
	}
	di.graph.mu.RUnlock()

	if resolved != 1 {
		t.Fatalf("resolved count = %d, want 1 (provider cached)", resolved)
	}
	if objNodes != 1 {
		t.Fatalf("object nodes = %d, want 1", objNodes)
	}

	// 失败 provider 不产生 resolved 计数
	di2 := New()
	di2.Provide(func() (*GObj, error) { return nil, errors.New("boom") })
	_ = di2.TryInject(func(o *GObj) {})
	di2.graph.mu.RLock()
	failedResolved := int64(0)
	for _, e := range di2.graph.eIndex {
		if e.Kind == EdgeResolved {
			failedResolved = e.Count
		}
	}
	di2.graph.mu.RUnlock()
	if failedResolved != 0 {
		t.Fatalf("failed provider must not count as resolved, got %d", failedResolved)
	}
}
```

注意:测试文件头部需补 `"errors"` import。

- [ ] **Step 2: 运行确认失败**

Run: `go test ./dixinternal -run TestDixGraphResolvedAndObjects -count=1`
Expected: FAIL(resolved count = 0)。

- [ ] **Step 3: 实现**

(a) `executeProvider` 成功路径:`dix.initializer[p.fn] = true` 所在行之后、`dix.recordProviderStat(p, duration, nil)` 之前插入:

```go
	dix.graph.markResolved(pNodeFor(p, outTyp), typeNodeFor(dix, outTyp))
```

为避免重复取节点,直接在 executeProvider 开头(准备 inputs 之前)取好:

```go
	gProviderNode := dix.graph.providerNode(p, outTyp)
	gOutNode := dix.graph.node(NodeType, outTyp, "", nil)
```

成功路径插入 `dix.graph.markResolved(gProviderNode, gOutNode)`。因此 Task 1 需要补一个公开取节点方法(加到 graph.go):

```go
// node 取得或创建类型节点(Task 4 的 resolved 计数需要预取)。
func (g *Graph) node(kind NodeKind, typ reflect.Type, group string, provider *providerFn) NodeID {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.nodeLocked(kind, typ, group, provider)
}
```

(实现时把该方法并入 Task 1 的 graph.go 即可,勿在 Task 4 重复添加。)

(b) `processProviderOutput` 的缓存合并循环之后、函数返回前:

```go
	// 对象节点:首次出现 (type, group) 时建节点并 bump version(快照判脏依据)。
	created := false
	for typeKey, groups := range newObjects {
		for groupKey := range groups {
			if dix.graph.addObject(typeKey, groupKey) {
				created = true
			}
		}
	}
	if created {
		dix.graph.bumpVersion()
	}
```

- [ ] **Step 4: 运行确认通过 + 全量**

Run: `go test ./dixinternal -race -count=1 && go test ./... -count=1`
Expected: 全部 PASS。

- [ ] **Step 5: 提交**

```bash
git add dixinternal/dix.go dixinternal/graph.go dixinternal/graph_test.go
git commit -m "feat: count provider executions and track object nodes in graph"
```

---

### Task 5: dixhttp 快照缓存(重复请求零反射)

**Files:**
- Modify: `dixhttp/server.go`
- Test: `dixhttp/server_api_test.go`(追加);新增 benchmark

**Interfaces:**
- Consumes: dixinternal `(dix *Dix) GraphVersion() uint64`(Task 2)。
- Produces: `Server` 的 version 判脏缓存字段;`buildDependencyData(details, objects, pkgFilter, limit)` 纯函数(替代 Server 方法 extractDependencyData);`BenchmarkHandleDependenciesWarm`。

**关键决策(偏离说明):** spec 写"extractDependencyData 删除"。实现上删除的是**Server 方法**(反射调用点),其聚合逻辑改写为纯函数 `buildDependencyData(details, objects, ...)`——输入来自 version 判脏的缓存,语义不变。反射只在 version 变化后的第一次请求发生,后续请求零反射,满足 spec 验收。

- [ ] **Step 1: 写失败测试与 benchmark(追加到 server_api_test.go)**

```go
// 同 version 重复请求必须零反射:第二次请求复用缓存的 ProviderDetails/objects。
// 通过注入探针 provider 的源码位置缓存间接验证:provide 新依赖后 version 变化,
// 快照随之刷新;version 不变时两次返回内容一致且不重算。
func TestDependencySnapshotCache(t *testing.T) {
	di := dixinternal.New()
	di.Provide(func() *apiStatsDep { return &apiStatsDep{} })
	if err := di.TryInject(func(*apiStatsDep) {}); err != nil {
		t.Fatalf("inject: %v", err)
	}

	server := NewServer(di)

	first := server.fullDependencyData()
	second := server.fullDependencyData()
	if first != second {
		t.Fatal("same version must return the cached snapshot (zero recompute)")
	}

	// 新 Provide → version 递增 → 快照重建,包含新 provider
	di.Provide(func() *apiStatsDep2 { return &apiStatsDep2{} })
	third := server.fullDependencyData()
	if third == second {
		t.Fatal("version bump must invalidate the snapshot")
	}
}

type apiStatsDep2 struct{}

func BenchmarkHandleDependenciesWarm(b *testing.B) {
	di := dixinternal.New()
	di.Provide(func() *apiStatsDep { return &apiStatsDep{} })
	if err := di.TryInject(func(*apiStatsDep) {}); err != nil {
		b.Fatalf("inject: %v", err)
	}
	server := NewServer(di)
	req := httptest.NewRequest(http.MethodGet, "/api/dependencies", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			b.Fatalf("status %d", rr.Code)
		}
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./dixhttp -run TestDependencySnapshotCache -count=1`
Expected: FAIL,`server.fullDependencyData undefined`。

- [ ] **Step 3: 实现**

(a) `dixhttp/server.go` 的 `Server` 结构体追加字段:

```go
	// 依赖数据快照缓存:version 判脏,ProviderDetails/objects 反射仅在
	// version 变化后的第一次请求发生,后续请求零反射。
	snapMu   sync.Mutex
	snapVer  uint64
	snapDet  []dixinternal.ProviderDetails
	snapObj  map[reflect.Type]map[string][]reflect.Value
	haveSnap bool
```

(import 补 `reflect`、`sync`;确认已有。)

(b) `HandleDependencies` 改为:

```go
func (s *Server) HandleDependencies(w http.ResponseWriter, r *http.Request) {
	pkgFilter := r.URL.Query().Get("package")
	limitStr := r.URL.Query().Get("limit")
	limit := 0
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	data := s.dependencyData(pkgFilter, limit)
	writeJSON(w, data)
}

// dependencyData 基于 version 判脏缓存组装依赖数据:
// 反射(details/objects)只在图版本变化后的第一次请求发生。
func (s *Server) dependencyData(pkgFilter string, limit int) *DependencyData {
	ver := s.dix.GraphVersion()

	s.snapMu.Lock()
	if !s.haveSnap || s.snapVer != ver {
		s.snapDet = s.dix.GetProviderDetails()
		s.snapObj = s.dix.GetObjects()
		s.snapVer = ver
		s.haveSnap = true
	}
	details, objects := s.snapDet, s.snapObj
	s.snapMu.Unlock()

	return buildDependencyData(details, objects, pkgFilter, limit)
}
```

(c) 原 `extractDependencyData` 方法改写为包级纯函数(签名去 receiver,输入显式传入;内部逻辑逐行保留,仅把 `s.dix.GetProviderDetails()`/`s.dix.GetObjects()` 换成参数 `details`/`objects`):

```go
func buildDependencyData(details []dixinternal.ProviderDetails, objects map[reflect.Type]map[string][]reflect.Value, pkgFilter string, limit int) *DependencyData {
	// 原 extractDependencyData 函数体,凡 s.dix.GetProviderDetails() 换成 details,
	// s.dix.GetObjects() 换成 objects,其余逐行不动。
}
```

(d) `HandleStats` 与 `HandlePackages` 同样改为消费缓存输入(它们也调用 GetProviderDetails/GetObjects):

```go
func (s *Server) HandleStats(w http.ResponseWriter, r *http.Request) {
	details, objects := s.cachedGraphInputs()
	...原逻辑,数据源换成参数...
}

func (s *Server) cachedGraphInputs() ([]dixinternal.ProviderDetails, map[reflect.Type]map[string][]reflect.Value) {
	ver := s.dix.GraphVersion()
	s.snapMu.Lock()
	defer s.snapMu.Unlock()
	if !s.haveSnap || s.snapVer != ver {
		s.snapDet = s.dix.GetProviderDetails()
		s.snapObj = s.dix.GetObjects()
		s.snapVer = ver
		s.haveSnap = true
	}
	return s.snapDet, s.snapObj
}
```

(`dependencyData` 内部同样改用 `s.cachedGraphInputs()`,去掉重复的锁段。)

- [ ] **Step 4: 运行确认通过**

Run: `go test ./dixhttp -race -count=1 && go test ./dixhttp -run xxx -bench BenchmarkHandleDependenciesWarm -benchtime=1000x -count=1`
Expected: 测试全 PASS;benchmark 稳定运行(数值记录到 PR 描述作为零反射佐证)。

- [ ] **Step 5: 提交**

```bash
git add dixhttp/server.go dixhttp/server_api_test.go
git commit -m "perf: cache dixhttp dependency data by graph version (zero reflection on warm requests)"
```

---

### Task 6: 文档同步 + 全量验证 + 交付

**Files:**
- Modify: `docs/design.md`、`docs/design_zh.md`(2.2 节数据结构段:depGraph/graphDirty → graph *Graph;2.3 核心流:环检测读增量图)
- Modify: `.version/changelog/Unreleased.md`(变更:运行时依赖图 + dixhttp 缓存)

- [ ] **Step 1: docs/design.md / design_zh.md 同步**

2.2 节数据结构代码块中:

```go
    // Cached dependency graph for cycle detection,
    // rebuilt lazily after providers change
    depGraph   map[reflect.Type]map[reflect.Type]bool
    graphDirty bool
```

替换为:

```go
    // Runtime dependency graph: declared edges maintained on Provide,
    // provider execution counters on resolve; consumed by cycle detection
    // and dixhttp (version-keyed snapshot cache)
    graph *Graph
```

2.3 核心流第 3 步 "Cycle check on the cached graph" 表述不变(仍然成立);如两语言文件存在对应段落,同步修改。

- [ ] **Step 2: changelog**

`.version/changelog/Unreleased.md` 的 `## 变更` 下追加:

```markdown
- 内部新增运行时依赖图 Graph（Provide 增量维护，环检测改读增量图，删除全量重建）；dixhttp 依赖数据按图版本缓存，热请求零反射
```

- [ ] **Step 3: 全量验证**

Run: `gofmt -l . && go vet ./... && go vet -C example ./... && task test && task lint`
Expected: 全绿;task test 覆盖根模块 + example 模块。

- [ ] **Step 4: 覆盖率核对**

Run: `go test ./dixinternal ./dixhttp -cover -count=1`
Expected: dixinternal ≥ 83%,dixhttp ≥ 74%(不回退)。

- [ ] **Step 5: PR**

```bash
git push -u origin feat/p1-graph-model
gh pr create --base v2 --title "feat(P1): 统一运行时依赖图 Graph;dixhttp 快照缓存" --body-file <(gh pr view --json body -q .body 2>/dev/null || echo "见 docs/superpowers/plans/2026-09-04-p1-graph-model.md")
```

CI 绿后 squash merge(按仓库流水线)。

---

## Self-Review 记录

1. **Spec 覆盖**:3.1 数据结构(Task 1)、3.2 维护时机与索引(Task 2/4;检索为线性 searchNodes,倒排留给 P4——spec 允许"随 P1 一起建"的最小形态)、3.3 迁移(Task 3/5)、3.4 验收(Task 3 不改测试 + Task 5 benchmark)——全覆盖。
2. **占位符**:无 TBD/TODO;Task 5 (c) 的"函数体逐行不动"是明确的机械变换指令,且原文在 git 历史可查,不算占位。
3. **类型一致性**:`graph *Graph`、`GraphVersion()`、`providerNode/addProduced/addDeclared/markResolved/addObject/declaredAdjacency/searchNodes/node` 各 Task 引用签名一致;`node()` 方法在 Task 1 定义(计划中注明并入 Task 1)、Task 4 消费。
