package dix_internal

import (
	"reflect"
	"strings"
)

func (x *Dix) buildDependencyGraph() map[reflect.Type]map[reflect.Type]bool {
	graph := make(map[reflect.Type]map[reflect.Type]bool)
	for typ, nodes := range x.providers {
		for _, n := range nodes {
			if graph[typ] == nil {
				graph[typ] = make(map[reflect.Type]bool)
			}
			for _, input := range n.input {
				for _, provider := range x.getAllProvideInput(input.typ) {
					graph[typ][provider.typ] = true
				}
			}
		}
	}
	return graph
}

// isCycle Check whether type circular dependency
func (x *Dix) isCycle() (string, bool) {
	depGraph := x.buildDependencyGraph()

	cyclePath := detectCycle(depGraph)
	if len(cyclePath) == 0 {
		return "", false
	}

	var pathStr strings.Builder
	for i, t := range cyclePath {
		if i > 0 {
			pathStr.WriteString(" -> ")
		}
		pathStr.WriteString(t.String())
	}
	return pathStr.String(), true
}
