package dixinternal

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/pubgo/dix/v2/dixtrace"
)

type (
	graphDepA struct{}
	graphDepB struct{}
	graphDepC struct{}
)

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

	a, b, c := reflect.TypeOf(&graphDepA{}), reflect.TypeOf(&graphDepB{}), reflect.TypeOf(&graphDepC{})

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
	var count int64
	for _, e := range g.eIndex {
		if e.Kind == EdgeResolved {
			count = e.Count
		}
	}
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
	a := reflect.TypeOf(graphDepA{})
	pa := &providerFn{}
	pa.fn = reflect.ValueOf(func() *graphDepA { return nil })
	paNode := g.providerNode(pa, a)
	g.addProduced(paNode, a)

	hits := g.searchNodes("graphdepa", 10)
	if len(hits) == 0 {
		t.Fatal("search should match type name case-insensitively")
	}
}

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

	// 产物边:5 条 = 容器自注册(*Dix)+ *GConf + *GDB + GOut 父类型 + *GDB(两个字段同类型去重后 1 条)。
	// 注:GOut 父类型的产物/声明边是比旧 buildDependencyGraph 更完整的语义——
	// 旧图只看 providers map 键,struct 输出父类型不注册,其字段依赖关系会漏检环。
	produced := 0
	di.graph.mu.RLock()
	for _, e := range di.graph.eIndex {
		if e.Kind == EdgeProduced {
			produced++
		}
	}
	di.graph.mu.RUnlock()
	if produced != 5 {
		t.Fatalf("produced edges = %d, want 5", produced)
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

// provider 每实际执行一次,对应 Resolved 边计数 +1;首次产物入缓存建 Object 节点。
// 失败的 provider 不产生 resolved 计数。
func TestDixGraphResolvedAndObjects(t *testing.T) {
	di := New()

	type GObj struct{ N int }
	di.Provide(func() *GObj { return &GObj{N: 1} })

	if err := di.TryInject(func(o *GObj) {}); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if err := di.TryInject(func(o *GObj) {}); err != nil {
		t.Fatalf("inject2: %v", err)
	}

	// resolved 计数:provider 只执行一次(产物缓存),第二次 inject 不再执行
	di.graph.mu.RLock()
	var resolved int64
	objNodes := 0
	for _, e := range di.graph.eIndex {
		if e.Kind == EdgeResolved {
			resolved = e.Count
		}
	}
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

// version 在对象首次入缓存时递增;重复注入(命中缓存)不再递增。
func TestDixGraphVersionOnObjectCreation(t *testing.T) {
	di := New()
	di.Provide(func() *graphDepA { return &graphDepA{} })

	if err := di.TryInject(func(a *graphDepA) {}); err != nil {
		t.Fatalf("inject: %v", err)
	}
	v1 := di.GraphVersion()

	if err := di.TryInject(func(a *graphDepA) {}); err != nil {
		t.Fatalf("inject2: %v", err)
	}
	if di.GraphVersion() != v1 {
		t.Fatal("cached re-injection must not bump version")
	}
}

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
	res := di.traceTracer.QueryEvents(dixtrace.Query{ContainerID: di.containerID})
	if res.Total == 0 {
		t.Fatal("local tracer should hold container events")
	}
	traceID := res.Records[0].TraceID

	tree := di.TraceTree(traceID)
	if !tree.Enabled || len(tree.Roots) == 0 {
		t.Fatalf("tree roots = %d, want >=1", len(tree.Roots))
	}
	// 根为 cycle_check,inject 挂在其下(与既有 trace 链路语义一致)
	if tree.Roots[0].Event.Operation != "inject.cycle_check" {
		t.Fatalf("root op = %s, want inject.cycle_check", tree.Roots[0].Event.Operation)
	}
	injectFound := false
	var walk func(n *dixtrace.TreeNode)
	walk = func(n *dixtrace.TreeNode) {
		if n.Event.Operation == "inject" {
			injectFound = true
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	for _, r := range tree.Roots {
		walk(r)
	}
	if !injectFound {
		t.Fatal("inject span should appear in the tree")
	}

	// 另一容器的私有缓冲为空(隔离)
	di2 := New(WithTraceBuffer(64))
	if got := di2.traceTracer.QueryEvents(dixtrace.Query{}); got.Total != 0 {
		t.Fatalf("isolated container sink should be empty, got %d", got.Total)
	}
}

// 全局容器(默认配置)事件进入全局 sink 且带 containerID。
func TestDefaultContainerStampsGlobalEvents(t *testing.T) {
	dixtrace.ResetForTest()
	di := New()
	di.Provide(func() *graphDepA { return &graphDepA{} })
	if err := di.TryInject(func(a *graphDepA) {}); err != nil {
		t.Fatalf("inject: %v", err)
	}
	res := dixtrace.QueryEvents(dixtrace.Query{ContainerID: di.containerID})
	if res.Total == 0 {
		t.Fatal("global events should be stamped with container id")
	}
}
