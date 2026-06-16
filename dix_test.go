package dix

import (
	"context"
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

func TestTryInjectContext(t *testing.T) {
	di := New()

	type dep struct{ V int }
	if err := TryProvide(di, func() *dep { return &dep{V: 2} }); err != nil {
		t.Fatalf("TryProvide: %v", err)
	}

	called := false
	err := TryInjectContext(context.Background(), di, func(d *dep) {
		called = d != nil && d.V == 2
	})
	if err != nil {
		t.Fatalf("TryInjectContext: %v", err)
	}
	if !called {
		t.Fatal("TryInjectContext callback was not invoked correctly")
	}
}

func TestInjectContextWithStructValue(t *testing.T) {
	di := New()

	type dep struct{ Name string }
	type app struct {
		Dep *dep
	}

	if err := TryProvide(di, func() *dep { return &dep{Name: "ctx"} }); err != nil {
		t.Fatalf("TryProvide: %v", err)
	}

	got := InjectContext(context.Background(), di, app{})
	if got.Dep == nil || got.Dep.Name != "ctx" {
		t.Fatal("InjectContext should inject into struct value")
	}
}

func TestInjectTPanicsForNonStruct(t *testing.T) {
	assertPanics(t, func() {
		_ = InjectT[int](New())
	})
}

func TestInjectTContextPanicsForNonStruct(t *testing.T) {
	assertPanics(t, func() {
		_ = InjectTContext[int](context.Background(), New())
	})
}

func assertPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}
