package dix

import (
	"fmt"
	"reflect"

	"github.com/pubgo/funk/errors"
	"github.com/pubgo/funk/recovery"
	"github.com/pubgo/funk/stack"
)

type inType struct {
	typ    reflect.Type
	isMap  bool
	isList bool
}

type outType struct {
	typ    reflect.Type
	isMap  bool
	isList bool
}

type node struct {
	fn     reflect.Value
	input  []*inType
	output *outType
}

func (n node) call(in []reflect.Value) []reflect.Value {
	defer recovery.Raise(func(err errors.XErr) {
		err.AddMsg("failed to handle provider invoke")
		err.AddTag("fn_stack", stack.CallerWithFunc(n.fn).String())
		err.AddTag("input", in)
		err.AddTag("input_data", fmt.Sprintf("%v", in))
	})

	return n.fn.Call(in)
}
