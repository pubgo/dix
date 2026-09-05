package dixinternal

import (
	"reflect"
	"testing"
)

type runtimeTarget struct{ Name string }

func TestRuntimeStatsIncludeConcreteProviderIdentity(t *testing.T) {
	di := New()
	di.Provide(func() *runtimeTarget { return &runtimeTarget{Name: "ready"} })
	_ = di.TryInject(func(*runtimeTarget) {})

	stats := di.GetProviderRuntimeStats()
	for _, stat := range stats {
		if stat.OutputType != "*dixinternal.runtimeTarget" {
			continue
		}
		if stat.RegistrationID == 0 {
			t.Fatal("expected non-zero registration ID")
		}
		if stat.ProviderID == "" {
			t.Fatal("expected provider ID")
		}
		if stat.CallCount != 1 {
			t.Fatalf("call count = %d, want 1", stat.CallCount)
		}
		return
	}
	t.Fatalf("target stat not found in %+v", stats)
}

func TestRuntimeStatsDoNotDeduplicateDistinctClosures(t *testing.T) {
	di := New()
	for i := 0; i < 2; i++ {
		di.Provide(func() *runtimeTarget { return &runtimeTarget{} })
	}
	stats := di.GetProviderRuntimeStats()
	count := 0
	for _, stat := range stats {
		if stat.OutputType == reflect.TypeOf(&runtimeTarget{}).String() {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("provider stats = %d, want 2", count)
	}
}
