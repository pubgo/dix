package dix_internal

import (
	"reflect"
	"strings"
)

func (x *Dix) buildDependencyGraph() map[reflect.Type]map[reflect.Type]bool {
	graph := make(map[reflect.Type]map[reflect.Type]bool)
	for outTyp, nodes := range x.providers {
		for _, providerNode := range nodes {
			if graph[outTyp] == nil {
				graph[outTyp] = make(map[reflect.Type]bool)
			}
			for _, input := range providerNode.inputList {
				for _, provider := range x.getProvideAllInputs(input.typ) {
					graph[outTyp][provider.typ] = true
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
