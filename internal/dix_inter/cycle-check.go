package dix_inter

import (
	"container/list"
	"reflect"
	"strings"
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
				types[n.output.typ][n.input[i].typ] = true
			}
		}
	}

	var check func(root reflect.Type, data map[reflect.Type]bool, nodes *list.List) bool
	check = func(root reflect.Type, nodeTypes map[reflect.Type]bool, nodes *list.List) bool {
		for typ := range nodeTypes {
			nodes.PushBack(typ)
			if root == typ {
				return true
			}

			if check(root, types[typ], nodes) {
				return true
			}
			nodes.Remove(nodes.Back())
		}
		return false
	}

	nodes := list.New()
	for root := range types {
		nodes.PushBack(root)
		if check(root, types[root], nodes) {
			break
		}
		nodes.Remove(nodes.Back())
	}

	if nodes.Len() == 0 {
		return "", false
	}

	var dep []string
	for nodes.Len() != 0 {
		dep = append(dep, nodes.Front().Value.(reflect.Type).String())
		nodes.Remove(nodes.Front())
	}

	return strings.Join(dep, " -> "), true
}
