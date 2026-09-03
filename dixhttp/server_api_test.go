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
