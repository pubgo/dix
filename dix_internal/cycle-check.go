package dix_internal

import (
	"reflect"
	"slices"
	"strings"

	"github.com/pubgo/funk/generic"
)

// isCycle Check whether type circular dependency
func (x *Dix) isCycle() (string, bool) {
	types := make(map[reflect.Type]map[reflect.Type]bool)
	for _, nodes := range x.providers {
		for _, n := range nodes {
			if types[n.output.typ] == nil {
				types[n.output.typ] = make(map[reflect.Type]bool)
			}

			for i := range n.input {
				for _, v := range x.getAllProvideInput(n.input[i].typ) {
					types[n.output.typ][v.typ] = true
				}
			}
		}
	}

	var checkHasCycle func(root reflect.Type, data map[reflect.Type]bool, nodes *[]reflect.Type) bool
	checkHasCycle = func(outT reflect.Type, inTypes map[reflect.Type]bool, nodePaths *[]reflect.Type) bool {
		for inT := range inTypes {
			if slices.ContainsFunc(*nodePaths, func(r reflect.Type) bool { return outT == inT }) {
				return true
			}

			*nodePaths = append(*nodePaths, inT)
			if checkHasCycle(outT, types[inT], nodePaths) {
				return true
			}
			*nodePaths = (*nodePaths)[:len(*nodePaths)-1]
		}
		return false
	}

	var nodes []reflect.Type
	for outT := range types {
		nodes = append(nodes, outT)
		if checkHasCycle(outT, types[outT], &nodes) {
			break
		}
		nodes = nodes[:len(nodes)-1]
	}

	if len(nodes) == 0 {
		return "", false
	}

	return strings.Join(generic.Map(nodes, func(i int) string { return nodes[i].String() }), " -> "), true
}
