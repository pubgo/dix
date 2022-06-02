package dix

import (
	"reflect"

	"github.com/pubgo/dix/internal/assert"
)

type inType struct {
	typ   reflect.Type
	isMap bool
}

type outType struct {
	typ   reflect.Type
	isMap bool
}

type node struct {
	fn     reflect.Value
	input  []*inType
	output *outType
}

func (n node) call(in []reflect.Value) []reflect.Value {
	defer assert.Recovery(func(err error) {
		logs.Println("provider call failed")
		logs.Println("provider is", callerWithFunc(n.fn))
		logs.Printf("provider input is %v\n", in)
	})

	return n.fn.Call(in)
}
