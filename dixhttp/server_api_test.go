package dixhttp

// 补充覆盖此前未测的 JSON API handler:stats/packages/trace/group-rules/index。
// 行为契约:这些接口是 dixhttp 可视化前端的数据源。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pubgo/dix/v2/dixinternal"
	"github.com/pubgo/dix/v2/dixtrace"
)

type apiStatsDep struct{}

func TestHandleStatsContract(t *testing.T) {
	di := dixinternal.New()
	di.Provide(func() *apiStatsDep { return &apiStatsDep{} })
	if err := di.TryInject(func(*apiStatsDep) {}); err != nil {
		t.Fatalf("inject: %v", err)
	}

	server := NewServer(di)
	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var stats StatsData
	if err := json.Unmarshal(rr.Body.Bytes(), &stats); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if stats.ProviderCount == 0 || stats.ObjectCount == 0 || stats.EdgeCount != 0 {
		t.Fatalf("unexpected stats: %+v (one provider, no edges)", stats)
	}
}

func TestHandlePackagesContract(t *testing.T) {
	di := dixinternal.New()
	di.Provide(func() *apiStatsDep { return &apiStatsDep{} })
	if err := di.TryInject(func(*apiStatsDep) {}); err != nil {
		t.Fatalf("inject: %v", err)
	}

	server := NewServer(di)
	req := httptest.NewRequest(http.MethodGet, "/api/packages", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var packages []PackageInfo
	if err := json.Unmarshal(rr.Body.Bytes(), &packages); err != nil {
		t.Fatalf("decode packages: %v", err)
	}
	if len(packages) == 0 {
		t.Fatal("expected at least one package group")
	}
	found := false
	for _, pkg := range packages {
		if strings.Contains(pkg.Name, "dixinternal") {
			found = true
		}
	}
	if !found {
		t.Fatalf("packages should contain the dixinternal group, got %+v", packages)
	}
}

func TestHandleTraceQueriesEvents(t *testing.T) {
	di := dixinternal.New()
	di.Provide(func() *apiStatsDep { return &apiStatsDep{} })
	if err := di.TryInject(func(*apiStatsDep) {}); err != nil {
		t.Fatalf("inject: %v", err)
	}

	// 任取一条已产生的内存 trace 事件,用它的 trace_id 验证过滤链路。
	all := dixtrace.QueryEvents(dixtrace.Query{Limit: 1})
	if len(all.Records) == 0 {
		t.Fatal("expected in-memory trace events after inject")
	}
	traceID := all.Records[0].TraceID

	server := NewServer(di)
	req := httptest.NewRequest(http.MethodGet, "/api/trace?trace_id="+traceID, nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var result struct {
		Enabled  bool `json:"enabled"`
		Total    int  `json:"total"`
		Returned int  `json:"returned"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode trace result: %v", err)
	}
	if !result.Enabled || result.Total == 0 || result.Returned == 0 {
		t.Fatalf("trace query should return events for %q, got %+v", traceID, result)
	}
}

func TestHandleGroupRulesRoundTrip(t *testing.T) {
	RegisterGroupRules(
		GroupRule{Name: "core", Prefixes: []string{" github.com/pubgo/dix ", ""}},
		GroupRule{Name: "core", Prefixes: []string{"dup-should-drop"}},
		GroupRule{Name: "  ", Prefixes: []string{"empty-name-should-drop"}},
	)
	t.Cleanup(func() { RegisterGroupRules() })

	server := NewServer(dixinternal.New())
	req := httptest.NewRequest(http.MethodGet, "/api/group-rules", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var rules []GroupRule
	if err := json.Unmarshal(rr.Body.Bytes(), &rules); err != nil {
		t.Fatalf("decode group rules: %v", err)
	}
	// sanitize:去重同名、丢弃空名与前缀、trim 空白。
	if len(rules) != 1 || rules[0].Name != "core" || len(rules[0].Prefixes) != 1 || rules[0].Prefixes[0] != "github.com/pubgo/dix" {
		t.Fatalf("unexpected sanitized rules: %+v", rules)
	}
}

func TestHandleIndexRendersBasePath(t *testing.T) {
	server := NewServerWithOptions(dixinternal.New(), WithBasePath("/dix"))

	req := httptest.NewRequest(http.MethodGet, "/dix/", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("content-type = %q, want text/html", ct)
	}
	if strings.Contains(rr.Body.String(), "__DIX_BASE_PATH__") {
		t.Fatal("base path placeholder must be replaced")
	}

	// 非 base 前缀路径不应命中页面。
	req = httptest.NewRequest(http.MethodGet, "/dix", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusMovedPermanently {
		t.Fatalf("redirect /dix -> /dix/ expected, got %d", rr.Code)
	}
}

type apiStatsDep2 struct{}

// 同 version 重复请求必须复用缓存(零反射);Provide 新依赖后 version 递增,快照刷新。
func TestDependencySnapshotCache(t *testing.T) {
	di := dixinternal.New()
	di.Provide(func() *apiStatsDep { return &apiStatsDep{} })
	if err := di.TryInject(func(*apiStatsDep) {}); err != nil {
		t.Fatalf("inject: %v", err)
	}

	server := NewServer(di)

	// 同 version 的热请求必须复用缓存的反射输入(零重算)
	first, _ := server.cachedGraphInputs()
	ver1 := server.snapVer
	second := server.dependencyData("", 0)
	if second == nil || len(second.Providers) == 0 {
		t.Fatal("warm request must return full dependency data")
	}
	det2, _ := server.cachedGraphInputs()
	if len(first) == 0 || len(det2) != len(first) || &det2[0] != &first[0] {
		t.Fatal("warm request must reuse cached reflection inputs (zero recompute)")
	}
	if server.snapVer != ver1 {
		t.Fatal("version unchanged, cache must not refresh")
	}

	// Provide 新依赖 → version 递增 → 缓存刷新,包含新 provider
	di.Provide(func() *apiStatsDep2 { return &apiStatsDep2{} })
	third := server.dependencyData("", 0)
	if third == nil || len(third.Providers) != len(second.Providers)+1 {
		t.Fatalf("version bump must refresh the snapshot: %d -> %d", len(second.Providers), len(third.Providers))
	}
}

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

// /api/trace-tree 返回一次 trace 的嵌套调用树;缺 trace_id 返回 400。
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

	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/trace-tree", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("missing trace_id should 400, got %d", rr.Code)
	}
}

// /api/search、/api/modules、/api/ego 契约;stats 概览增字段。
func TestHandleSearchModulesEgo(t *testing.T) {
	di := dixinternal.New()
	di.Provide(func() *apiStatsDep { return &apiStatsDep{} })
	if err := di.TryInject(func(*apiStatsDep) {}); err != nil {
		t.Fatalf("inject: %v", err)
	}
	server := NewServer(di)

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/search?q=apistats&kind=type", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("search status %d", rr.Code)
	}
	var hits []dixinternal.SearchHit
	if err := json.Unmarshal(rr.Body.Bytes(), &hits); err != nil || len(hits) == 0 {
		t.Fatalf("search hits = %v err = %v", hits, err)
	}

	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/modules", nil))
	var modules []dixinternal.ModuleInfo
	if err := json.Unmarshal(rr.Body.Bytes(), &modules); err != nil || len(modules) == 0 {
		t.Fatalf("modules = %v err = %v", modules, err)
	}

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

// 回归:stats/packages 先行"预热"输入缓存后,/api/dependencies 仍必须
// 返回完整数据(此前 snapFull 未随缓存刷新,返回了字面量 null)。
func TestHandleDependenciesAfterStatsWarmup(t *testing.T) {
	di := dixinternal.New()
	di.Provide(func() *apiStatsDep { return &apiStatsDep{} })
	if err := di.TryInject(func(*apiStatsDep) {}); err != nil {
		t.Fatalf("inject: %v", err)
	}
	server := NewServer(di)

	// 先命中会填充输入缓存的端点
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/stats", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("stats status %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/dependencies", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("dependencies status %d", rr.Code)
	}
	var data DependencyData
	if err := json.Unmarshal(rr.Body.Bytes(), &data); err != nil {
		t.Fatalf("decode dependencies: %v", err)
	}
	if len(data.Providers) == 0 {
		t.Fatalf("dependencies must include providers after warmup, got %q", rr.Body.String())
	}
}
