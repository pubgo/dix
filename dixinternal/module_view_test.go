package dixinternal

import (
	"reflect"
	"testing"
)

type moduleViewTarget struct{}

func newModuleViewTarget(*testing.T) *moduleViewTarget {
	return &moduleViewTarget{}
}

func TestModuleViewIncludesProducedAndDeclaredTopology(t *testing.T) {
	di := New()
	di.Provide(newModuleViewTarget)
	module := resolveTypePkgPath(reflect.TypeOf(&moduleViewTarget{}))

	view := di.ModuleView(module, 100, 100)
	if view.Name != module {
		t.Fatalf("name = %q", view.Name)
	}
	if view.TypeCount < 2 {
		t.Fatalf("summary = %+v", view)
	}
	providerFound, targetFound, externalFound := false, false, false
	for _, node := range view.Nodes {
		switch node.Label {
		case "*dixinternal.moduleViewTarget":
			if node.Kind == "provider" {
				providerFound = true
			} else {
				targetFound = true
			}
		case "*testing.T":
			externalFound = true
			if !node.External {
				t.Fatal("external dependency node was not marked")
			}
		}
	}
	if !providerFound || !targetFound || !externalFound {
		t.Fatalf("expected provider, target, and external nodes: %+v", view.Nodes)
	}
	if len(view.Edges) < 1 {
		t.Fatalf("expected declared topology, got %+v", view.Edges)
	}
	if len(view.DependsOn) == 0 || view.DependsOn[0].Name != "testing" {
		t.Fatalf("depends on = %+v", view.DependsOn)
	}
}

func TestModuleViewBoundsNodesAndEdges(t *testing.T) {
	di := New()
	targetType := reflect.TypeOf(&moduleViewTarget{})
	for i := 0; i < 12; i++ {
		provider := &providerFn{fn: reflect.ValueOf(newModuleViewTarget), registrationID: uint64(i + 1)}
		node := di.graph.providerNode(provider, targetType)
		di.graph.addProduced(node, targetType)
	}

	view := di.ModuleView("github.com/pubgo/dix/v2/dixinternal", 2, 1)
	if len(view.Nodes) != 2 {
		t.Fatalf("nodes = %d, want 2", len(view.Nodes))
	}
	if len(view.Edges) > 1 {
		t.Fatalf("edges = %d, want <= 1", len(view.Edges))
	}
	if !view.Truncated {
		t.Fatal("expected truncated marker")
	}
}
