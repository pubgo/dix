package dix

import (
	"reflect"

	"github.com/pubgo/funk"
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
	defer funk.RecoverAndRaise(func(err funk.XErr) funk.XErr {
		err = err.WrapF("provider call failed")
		err = err.WrapF("provider is %s", callerWithFunc(n.fn))
		return err.WrapF("provider input is %v\n", in)
	})

	return n.fn.Call(in)
}
