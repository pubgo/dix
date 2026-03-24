package dixhttp

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/pubgo/dix/v2/dixinternal"
)

type runtimeStatsDep struct{}
type basePathTypeDep interface {
	Name() string
}

type basePathTypeDepImpl struct{}

func (basePathTypeDepImpl) Name() string {
	return "dep"
}

func TestHandleRuntimeStats(t *testing.T) {
	di := dixinternal.New()
	di.Provide(func() *runtimeStatsDep { return &runtimeStatsDep{} })

	if err := di.TryInject(func(*runtimeStatsDep) {}); err != nil {
		t.Fatalf("failed to initialize dependency: %v", err)
	}

	server := NewServer(di)
	req := httptest.NewRequest(http.MethodGet, "/api/runtime-stats?limit=1", nil)
	rr := httptest.NewRecorder()

	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var stats []dixinternal.ProviderRuntimeStats
	if err := json.Unmarshal(rr.Body.Bytes(), &stats); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(stats) == 0 {
		t.Fatal("expected at least one runtime stat record")
	}

	if len(stats) > 1 {
		t.Fatalf("expected response limited to 1 item, got %d", len(stats))
	}
}

func TestHandleRuntimeStatsIncludeUninitializedProviders(t *testing.T) {
	type depA struct{}
	type depB struct{}

	di := dixinternal.New()
	di.Provide(func() *depA { return &depA{} })
	di.Provide(func() *depB { return &depB{} })

	if err := di.TryInject(func(*depA) {}); err != nil {
		t.Fatalf("failed to initialize depA: %v", err)
	}

	server := NewServer(di)
	req := httptest.NewRequest(http.MethodGet, "/api/runtime-stats", nil)
	rr := httptest.NewRecorder()

	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var stats []dixinternal.ProviderRuntimeStats
	if err := json.Unmarshal(rr.Body.Bytes(), &stats); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(stats) < 2 {
		t.Fatalf("expected at least 2 providers in runtime stats, got %d", len(stats))
	}
}

func TestHandleErrors(t *testing.T) {
	type missingDep struct{}

	di := dixinternal.New()
	err := di.TryInject(func(*missingDep) {})
	if err == nil {
		t.Fatal("expected TryInject to fail for missing dependency")
	}

	server := NewServer(di)
	req := httptest.NewRequest(http.MethodGet, "/api/errors?limit=1", nil)
	rr := httptest.NewRecorder()

	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var events []dixinternal.RecentError
	if err := json.Unmarshal(rr.Body.Bytes(), &events); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected one error event with limit=1, got %d", len(events))
	}

	if events[0].Operation == "" || events[0].Message == "" {
		t.Fatalf("expected operation and message in error event, got %+v", events[0])
	}

	if events[0].ErrorType == "" {
		t.Fatalf("expected error_type in error event, got %+v", events[0])
	}

	if events[0].Hint == "" {
		t.Fatalf("expected hint in error event, got %+v", events[0])
	}
}

func TestHandleDiagnosticsWhenEnvNotConfigured(t *testing.T) {
	t.Setenv("DIX_DIAG_FILE", "")

	server := NewServer(dixinternal.New())
	req := httptest.NewRequest(http.MethodGet, "/api/diagnostics", nil)
	rr := httptest.NewRecorder()

	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	enabled, _ := resp["enabled"].(bool)
	if enabled {
		t.Fatalf("expected enabled=false when DIX_DIAG_FILE is empty, got %+v", resp)
	}
}

func TestHandleDiagnosticsReadsFileRecords(t *testing.T) {
	di := dixinternal.New()

	diagPath := filepath.Join(t.TempDir(), "diag.jsonl")
	content := "" +
		`{"record_id":1,"kind":"trace","event":"inject.start","occurred_at_unix_nano":100}` + "\n" +
		`{"record_id":2,"kind":"error","event":"","occurred_at_unix_nano":200,"payload":{"message":"boom"}}` + "\n"

	if err := os.WriteFile(diagPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write diagnostic file: %v", err)
	}

	t.Setenv("DIX_DIAG_FILE", diagPath)

	server := NewServer(di)
	req := httptest.NewRequest(http.MethodGet, "/api/diagnostics?kind=trace&limit=10", nil)
	rr := httptest.NewRecorder()

	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp struct {
		Enabled bool                         `json:"enabled"`
		Path    string                       `json:"path"`
		Total   int                          `json:"total"`
		Records []dixinternal.DiagFileRecord `json:"records"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Enabled {
		t.Fatalf("expected enabled=true, got %+v", resp)
	}
	if resp.Path != diagPath {
		t.Fatalf("expected path %s, got %s", diagPath, resp.Path)
	}
	if resp.Total != 1 {
		t.Fatalf("expected total filtered records 1, got %d", resp.Total)
	}
	if len(resp.Records) != 1 || resp.Records[0].Kind != "trace" {
		t.Fatalf("expected single trace record, got %+v", resp.Records)
	}
}

func TestHandleDetailsRoutesWithBasePath(t *testing.T) {
	di := dixinternal.New()
	di.Provide(func() basePathTypeDep { return basePathTypeDepImpl{} })

	if err := di.TryInject(func(basePathTypeDep) {}); err != nil {
		t.Fatalf("failed to initialize basePathTypeDep: %v", err)
	}

	providerDetails := di.GetProviderDetails()
	if len(providerDetails) == 0 {
		t.Fatal("expected provider details")
	}

	targetType := ""
	targetPkg := ""
	for _, d := range providerDetails {
		if d.FunctionName != "" && d.OutputType != "" {
			targetType = d.OutputType
			targetPkg = extractPackage(d.OutputType)
			if targetPkg != "" {
				break
			}
		}
	}

	if targetType == "" || targetPkg == "" {
		t.Fatalf("failed to locate provider detail with valid type/package: %+v", providerDetails)
	}

	server := NewServerWithOptions(di, WithBasePath("/dix"))

	t.Run("package details", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/dix/api/package/"+targetPkg, nil)
		rr := httptest.NewRecorder()

		server.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		var resp PackageDetailsData
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode package response: %v", err)
		}

		if resp.Package != targetPkg {
			t.Fatalf("expected package %q, got %q", targetPkg, resp.Package)
		}
	})

	t.Run("type details", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/dix/api/type/"+targetType+"?depth=1", nil)
		rr := httptest.NewRecorder()

		server.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		var resp TypeDetailsData
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode type response: %v", err)
		}

		if resp.RootType != targetType {
			t.Fatalf("expected root type %q, got %q", targetType, resp.RootType)
		}
	})
}

func TestWriteJSONEncodeFailureReturns500(t *testing.T) {
	rr := httptest.NewRecorder()

	writeJSON(rr, map[string]float64{"nan": math.NaN()})

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}
