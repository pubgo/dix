package dixhttp

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	dix "github.com/pubgo/dix/v2"
)

type projectionOutputA struct{}
type projectionOutputB struct{}
type projectionAggregate struct {
	A *projectionOutputA
	B *projectionOutputB
}

func TestDependenciesPreserveRegistrationAndOutputIdentity(t *testing.T) {
	container := dix.New()
	dix.Provide(container, func() projectionAggregate {
		return projectionAggregate{A: &projectionOutputA{}, B: &projectionOutputB{}}
	})

	server := NewServer(container)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest("GET", "/api/dependencies", nil))
	if recorder.Code != 200 {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response DependencyData
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	registrationIDs := map[uint64]bool{}
	providerIDs := map[string]bool{}
	for _, provider := range response.Providers {
		if provider.OutputType != "*dixhttp.projectionOutputA" && provider.OutputType != "*dixhttp.projectionOutputB" {
			continue
		}
		if provider.RegistrationID == 0 || len(provider.ProviderIDs) == 0 {
			t.Fatalf("provider identities must not be empty: %+v", provider)
		}
		registrationIDs[provider.RegistrationID] = true
		for _, providerID := range provider.ProviderIDs {
			providerIDs[providerID] = true
		}
	}
	if len(registrationIDs) != 1 {
		t.Fatalf("one struct registration should produce one registration_id, got %v", registrationIDs)
	}
	if len(providerIDs) != 2 {
		t.Fatalf("two outputs should produce two provider IDs, got %v", providerIDs)
	}
}

func TestDistinctSameLineRegistrationsDoNotCartesianProductEdges(t *testing.T) {
	container := dix.New()
	for i := 0; i < 2; i++ {
		dix.Provide(container, func() *projectionOutputA { return &projectionOutputA{} })
	}

	server := NewServer(container)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest("GET", "/api/dependencies", nil))

	var response DependencyData
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	targetCount := 0
	for _, provider := range response.Providers {
		if provider.OutputType == "*dixhttp.projectionOutputA" {
			targetCount++
		}
	}
	if targetCount != 2 {
		t.Fatalf("expected distinct provider nodes, got %d", targetCount)
	}
	for _, provider := range response.Providers {
		if provider.OutputType != "*dixhttp.projectionOutputA" {
			continue
		}
		if len(provider.InputTypes)*len(provider.OutputTypes) > 1 {
			t.Fatalf("unrelated input/output pairs created a Cartesian product: %+v", provider)
		}
	}
}
