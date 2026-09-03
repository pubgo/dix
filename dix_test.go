package dix

import (
	"context"
	"testing"
	"time"
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

// ---- 公开 API 包装器的行为锁:Inject/Provide/InjectContext/InjectT/Option ----

func TestInjectFunction(t *testing.T) {
	di := New()

	type dep struct{ V int }
	if err := TryProvide(di, func() *dep { return &dep{V: 3} }); err != nil {
		t.Fatalf("TryProvide: %v", err)
	}

	called := false
	Inject(di, func(d *dep) { called = d.V == 3 })
	if !called {
		t.Fatal("Inject should resolve function parameters")
	}
}

func TestInjectStructValue(t *testing.T) {
	di := New()

	type dep struct{ Name string }
	type app struct {
		Dep *dep
	}
	if err := TryProvide(di, func() *dep { return &dep{Name: "v"} }); err != nil {
		t.Fatalf("TryProvide: %v", err)
	}

	got := Inject(di, app{})
	if got.Dep == nil || got.Dep.Name != "v" {
		t.Fatalf("Inject on struct value should fill fields, got %+v", got)
	}
}

func TestProvidePanicsOnInvalid(t *testing.T) {
	assertPanics(t, func() { Provide(New(), nil) })
}

func TestInjectContextFunction(t *testing.T) {
	di := New()

	type dep struct{ V int }
	if err := TryProvide(di, func() *dep { return &dep{V: 4} }); err != nil {
		t.Fatalf("TryProvide: %v", err)
	}

	called := false
	InjectContext(context.Background(), di, func(d *dep) { called = d.V == 4 })
	if !called {
		t.Fatal("InjectContext should resolve function parameters")
	}
}

func TestInjectTFillsStruct(t *testing.T) {
	di := New()

	type cfg struct{ DSN string }
	type app struct {
		Cfg *cfg
	}
	if err := TryProvide(di, func() *cfg { return &cfg{DSN: "x"} }); err != nil {
		t.Fatalf("TryProvide: %v", err)
	}

	got := InjectT[app](di)
	if got.Cfg == nil || got.Cfg.DSN != "x" {
		t.Fatalf("InjectT should construct and fill the struct, got %+v", got)
	}
}

func TestInjectTContextFillsStruct(t *testing.T) {
	di := New()

	type cfg struct{ DSN string }
	type app struct {
		Cfg *cfg
	}
	if err := TryProvide(di, func() *cfg { return &cfg{DSN: "ctx"} }); err != nil {
		t.Fatalf("TryProvide: %v", err)
	}

	got := InjectTContext[app](context.Background(), di)
	if got.Cfg == nil || got.Cfg.DSN != "ctx" {
		t.Fatalf("InjectTContext should construct and fill the struct, got %+v", got)
	}
}

// 容器 Option 必须经由根包转发到位,且默认值保留。
func TestOptionForwarding(t *testing.T) {
	di := New(
		WithValuesNull(),
		WithProviderTimeout(3*time.Second),
		WithSlowProviderThreshold(time.Millisecond),
	)
	opt := di.Option()
	if !opt.AllowValuesNull || opt.ProviderTimeout != 3*time.Second || opt.SlowProviderThreshold != time.Millisecond {
		t.Fatalf("options not forwarded: %+v", opt)
	}

	reject := New(WithRejectEmptyCollections())
	ropt := reject.Option()
	if ropt.AllowValuesNull {
		t.Fatal("WithRejectEmptyCollections must disable AllowValuesNull")
	}
	if ropt.ProviderTimeout != 15*time.Second || ropt.SlowProviderThreshold != 2*time.Second {
		t.Fatalf("unspecified options must keep defaults, got %+v", ropt)
	}
}
