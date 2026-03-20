package dixhttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pubgo/dix/v2/dixinternal"
)

type runtimeStatsDep struct{}

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
}
