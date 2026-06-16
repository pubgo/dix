package dixglobal

import "testing"

type testGlobalDep struct {
	Value string
}

func TestProvideAndInject(t *testing.T) {
	Provide(func() *testGlobalDep {
		return &testGlobalDep{Value: "ok"}
	})

	called := false
	Inject(func(dep *testGlobalDep) {
		called = dep != nil && dep.Value == "ok"
	})

	if !called {
		t.Fatal("Inject should resolve provided dependency")
	}
}

func TestInjectT(t *testing.T) {
	type injectTDep struct {
		Value string
	}

	type app struct {
		Dep *injectTDep
	}

	Provide(func() *injectTDep {
		return &injectTDep{Value: "ok"}
	})

	got := InjectT[app]()
	if got.Dep == nil || got.Dep.Value != "ok" {
		t.Fatal("InjectT should populate struct fields")
	}
}
