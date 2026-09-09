package dixinternal

import (
	"reflect"
	"testing"
)

type identityA struct{}
type identityB struct{}
type identityAggregate struct {
	A *identityA
	B *identityB
}

func TestStructProviderOutputsShareRegistrationID(t *testing.T) {
	di := New()
	di.Provide(func() identityAggregate {
		return identityAggregate{A: &identityA{}, B: &identityB{}}
	})

	ids := make(map[string]uint64)
	for _, provider := range di.providers[reflect.TypeOf(&identityA{})] {
		ids["A"] = provider.registrationID
	}
	for _, provider := range di.providers[reflect.TypeOf(&identityB{})] {
		ids["B"] = provider.registrationID
	}
	if ids["A"] == 0 || ids["A"] != ids["B"] {
		t.Fatalf("struct outputs should share one registration ID: %#v", ids)
	}
}

func TestDistinctClosureRegistrationsHaveDistinctRegistrationIDs(t *testing.T) {
	di := New()
	for _, name := range []string{"first", "second"} {
		value := name
		di.Provide(func() *identityA { return &identityA{} })
		if value == "" {
			t.Fatal("closure registration setup unexpectedly empty")
		}
	}

	providers := di.providers[reflect.TypeOf(&identityA{})]
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}
	if providers[0].registrationID == providers[1].registrationID {
		t.Fatal("distinct registrations must not share a registration ID")
	}
}
