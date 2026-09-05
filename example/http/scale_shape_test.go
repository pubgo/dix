package main

import "testing"

func TestScaleFixtureShape(t *testing.T) {
	container := buildContainer()
	if err := container.TryInject(func(*Application) {}); err != nil {
		t.Fatal(err)
	}
	if err := container.TryInject(func(ScaleFixture) {}); err != nil {
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
	if len(providers) < 190 {
		t.Fatalf("providers = %d, want at least 190", len(providers))
	}
	if objectCount < 180 {
		t.Fatalf("objects = %d, want at least 180", objectCount)
	}
}
