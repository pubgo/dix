package dix

import (
	"testing"
)

func TestVersion(t *testing.T) {
	if Version() == "" {
		t.Fatal("Version() returned empty string")
	}
}

func TestTryProvide(t *testing.T) {
	di := New()

	type svc struct{ Name string }

	if err := TryProvide(di, func() *svc { return &svc{Name: "ok"} }); err != nil {
		t.Fatalf("TryProvide: %v", err)
	}

	if err := TryProvide(di, nil); err == nil {
		t.Fatal("TryProvide(nil) should return error")
	}
}

func TestTryInject(t *testing.T) {
	di := New()

	type dep struct{ V int }

	if err := TryProvide(di, func() *dep { return &dep{V: 1} }); err != nil {
		t.Fatalf("TryProvide: %v", err)
	}

	called := false
	if err := TryInject(di, func(d *dep) { called = d.V == 1 }); err != nil {
		t.Fatalf("TryInject: %v", err)
	}
	if !called {
		t.Fatal("TryInject callback was not invoked correctly")
	}

	if err := TryInject(di, func(*missingType) {}); err == nil {
		t.Fatal("TryInject with missing dependency should return error")
	}
}

type missingType struct{}
