package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pubgo/dix/v2/dixhttp"

	"github.com/pubgo/dix/example/http/app"
)

func TestDemoContainerShape(t *testing.T) {
	container := buildContainer()
	if err := container.TryInject(func(*app.Application) {}); err != nil {
		t.Fatal(err)
	}
	modules := container.ModuleGraph()
	providers := container.GetProviderDetails()
	objects := container.GetObjects()

	objectCount := 0
	for _, groups := range objects {
		for _, values := range groups {
			objectCount += len(values)
		}
	}
	if len(modules) < 10 {
		t.Fatalf("modules = %d, want at least 10", len(modules))
	}
	// 真实分层后规模来自 domain + plugins,不再依赖人造 ScaleFixture。
	if len(providers) < 90 {
		t.Fatalf("providers = %d, want at least 90", len(providers))
	}
	if objectCount < 80 {
		t.Fatalf("objects = %d, want at least 80", objectCount)
	}
}

func TestPyramidHasTwoBusinessEntries(t *testing.T) {
	container := buildContainer()
	server := dixhttp.NewServer(container)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest("GET", "/api/dependencies", nil))
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	var payload struct {
		Providers []struct {
			ID         string   `json:"id"`
			OutputType string   `json:"output_type"`
			InputTypes []string `json:"input_types"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}

	typeProducers := map[string][]string{}
	for _, p := range payload.Providers {
		out := p.OutputType
		if out == "" {
			continue
		}
		typeProducers[out] = append(typeProducers[out], p.ID)
	}
	consumed := map[string]bool{}
	for _, p := range payload.Providers {
		for _, in := range p.InputTypes {
			for _, producerID := range typeProducers[in] {
				if producerID != p.ID {
					consumed[producerID] = true
				}
			}
		}
	}

	var entries []string
	for _, p := range payload.Providers {
		if consumed[p.ID] {
			continue
		}
		if strings.Contains(p.OutputType, "dixinternal") {
			continue
		}
		entries = append(entries, p.OutputType)
	}
	if len(entries) != 2 {
		t.Fatalf("business pyramid entries = %d (%v), want exactly 2 (Application, TimeoutProbe)", len(entries), entries)
	}
	want := map[string]bool{
		"Application":  false,
		"TimeoutProbe": false,
	}
	for _, e := range entries {
		switch {
		case strings.Contains(e, "Application"):
			want["Application"] = true
		case strings.Contains(e, "TimeoutProbe"):
			want["TimeoutProbe"] = true
		}
	}
	for name, ok := range want {
		if !ok {
			t.Fatalf("missing expected entry kind %s in %v", name, entries)
		}
	}
}
