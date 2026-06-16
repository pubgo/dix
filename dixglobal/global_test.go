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
	type app struct {
		Dep *testGlobalDep
	}

	got := InjectT[app]()
	if got.Dep == nil {
		t.Fatal("InjectT should populate struct fields")
	}
}
