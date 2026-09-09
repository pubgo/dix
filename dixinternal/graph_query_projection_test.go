package dixinternal

import (
	"reflect"
	"testing"
)

type packageTarget struct{}

func packageProvider() *packageTarget { return &packageTarget{} }

func TestEgoInstantiatedUsesObjectNodes(t *testing.T) {
	di := New()
	di.graph.node(NodeType, reflect.TypeOf(&packageTarget{}), "", nil)
	view := di.EgoGraph("*dixinternal.packageTarget", 1, "both")
	for _, node := range view.Nodes {
		if node.Label == "*dixinternal.packageTarget" && node.State == "instantiated" {
			t.Fatal("missing provider must not be reported instantiated")
		}
	}

	di.Provide(func() *packageTarget { return &packageTarget{} })
	di.graph.addObject(reflect.TypeOf(&packageTarget{}), "")
	view = di.EgoGraph("*dixinternal.packageTarget", 1, "both")
	instantiated := false
	for _, node := range view.Nodes {
		if node.Label == "*dixinternal.packageTarget" && node.State == "instantiated" {
			instantiated = true
		}
	}
	if !instantiated {
		t.Fatal("object-bearing ego node should be instantiated")
	}
}

func TestResolvedTopNAggregatesByType(t *testing.T) {
	di := New()
	targetType := reflect.TypeOf(&packageTarget{})

	for i := 0; i < 2; i++ {
		provider := &providerFn{fn: reflect.ValueOf(packageProvider), registrationID: uint64(i + 1)}
		node := di.graph.providerNode(provider, targetType)
		di.graph.markResolved(node, targetType)
	}

	rows := di.ResolvedTopN(10)
	found := 0
	for _, row := range rows {
		if row.Type == "*dixinternal.packageTarget" {
			found++
			if row.Count != 2 {
				t.Fatalf("count = %d, want 2", row.Count)
			}
		}
	}
	if found != 1 {
		t.Fatalf("type rows = %d, want 1", found)
	}
}
