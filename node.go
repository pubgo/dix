package dix

import (
	"reflect"

	"github.com/pubgo/funk/errors"
	"github.com/pubgo/funk/recovery"
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
		err.AddMsg("provider invoke failed")
		err.AddTag("func", callerWithFunc(n.fn))
		err.AddTag("input", in)
	})

	return n.fn.Call(in)
}
