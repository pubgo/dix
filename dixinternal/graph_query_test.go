package dixinternal

import (
	"errors"
	"strings"
	"testing"
	"time"
)

type QAService struct{}
type QARepo struct{}
type QADepA struct{}
type QADepB struct{}
type QADepC struct{}
type QASlow struct{}
type QABroken struct{}

func containsStr(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func viewLabels(v GraphView) []string {
	out := make([]string, 0, len(v.Nodes))
	for _, n := range v.Nodes {
		out = append(out, n.Label)
	}
	return out
}

func TestSearchNodesFilters(t *testing.T) {
	di := New()
	di.Provide(func() *QAService { return &QAService{} })
	di.Provide(func() *QARepo { return &QARepo{} })
	_ = di.TryInject(func(s *QAService, r *QARepo) {})

	hits := di.SearchNodes("qaservice", "", "", "", 50)
	if len(hits) == 0 {
		t.Fatal("q filter should match provider fn and type name")
	}

	hits = di.SearchNodes("", "provider", "", "", 50)
	for _, h := range hits {
		if h.Kind != "provider" {
			t.Fatalf("kind filter failed: %+v", h)
		}
	}

	hits = di.SearchNodes("", "type", "", "instantiated", 50)
	found := 0
	for _, h := range hits {
		if h.Label == "*dixinternal.QAService" || h.Label == "*dixinternal.QARepo" {
			found++
		}
	}
	if found != 2 {
		t.Fatalf("instantiated filter found = %d, want 2: %+v", found, hits)
	}

	if hits := di.SearchNodes("", "", "", "", 1); len(hits) != 1 {
		t.Fatal("limit must be honored")
	}
}

func TestModuleGraphAggregation(t *testing.T) {
	di := New()
	di.Provide(func(r *QARepo) *QAService { return &QAService{} })

	modules := di.ModuleGraph()
	if len(modules) == 0 {
		t.Fatal("module graph should not be empty")
	}
	var dixModule *ModuleInfo
	for i := range modules {
		if modules[i].Name == "github.com/pubgo/dix/v2/dixinternal" {
			m := modules[i]
			dixModule = &m
		}
	}
	if dixModule == nil {
		t.Fatalf("dixinternal module missing: %+v", modules)
	}
	if dixModule.TypeCount == 0 || dixModule.ProviderCount == 0 {
		t.Fatalf("module counts wrong: %+v", dixModule)
	}
	// 同包内声明边不产生跨模块依赖
	if len(dixModule.DependsOn) != 0 {
		t.Fatalf("same-package deps should not appear in DependsOn: %+v", dixModule.DependsOn)
	}
}

func TestEgoGraphDepthAndDirection(t *testing.T) {
	di := New()
	di.Provide(func(*QADepB) *QADepA { return &QADepA{} })
	di.Provide(func(*QADepC) *QADepB { return &QADepB{} })

	view := di.EgoGraph("*dixinternal.QADepA", 1, "deps")
	labels := viewLabels(view)
	if !containsStr(labels, "*dixinternal.QADepA") || !containsStr(labels, "*dixinternal.QADepB") || containsStr(labels, "*dixinternal.QADepC") {
		t.Fatalf("deps view wrong: %v", labels)
	}

	view = di.EgoGraph("*dixinternal.QADepA", 2, "both")
	if !containsStr(viewLabels(view), "*dixinternal.QADepC") {
		t.Fatalf("both depth=2 should include C: %v", viewLabels(view))
	}

	// dependents 方向:没有类型依赖 A,从 A 出发只含自身;从 C 反向可达 B(1 跳)、A(2 跳)
	view = di.EgoGraph("*dixinternal.QADepA", 1, "dependents")
	if containsStr(viewLabels(view), "*dixinternal.QADepB") {
		t.Fatalf("nothing depends on A: %v", viewLabels(view))
	}
	view = di.EgoGraph("*dixinternal.QADepC", 1, "dependents")
	if !containsStr(viewLabels(view), "*dixinternal.QADepB") || containsStr(viewLabels(view), "*dixinternal.QADepA") {
		t.Fatalf("dependents of C at depth 1 should include only B: %v", viewLabels(view))
	}
	view = di.EgoGraph("*dixinternal.QADepC", 2, "dependents")
	if !containsStr(viewLabels(view), "*dixinternal.QADepA") {
		t.Fatalf("dependents of C at depth 2 should include A: %v", viewLabels(view))
	}

	// 未知 center:空视图
	view = di.EgoGraph("*dixinternal.NoSuchType", 2, "both")
	if len(view.Nodes) != 0 || len(view.Edges) != 0 {
		t.Fatalf("unknown center should return empty view: %v", view)
	}
}

func TestResolvedTopNAndProblemProviders(t *testing.T) {
	di := New(WithSlowProviderThreshold(time.Millisecond))
	di.Provide(func() *QASlow {
		time.Sleep(5 * time.Millisecond)
		return &QASlow{}
	})
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
	if !strings.Contains(strings.Join(broken, ","), "QABroken") && !strings.Contains(strings.Join(broken, ","), "func") {
		t.Fatalf("broken should reference the broken provider function: %v", broken)
	}
}
