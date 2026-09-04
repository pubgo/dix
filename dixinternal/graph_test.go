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
