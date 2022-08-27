package dix

import (
	"reflect"

	"github.com/pubgo/funk/recovery"
	"github.com/pubgo/funk/xerr"
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
	defer recovery.Raise(func(err xerr.XErr) xerr.XErr {
		err = err.Wrap("provider invoke failed")
		err = err.WithMeta("func", callerWithFunc(n.fn))
		return err.WithMeta("input", in)
	})

	return n.fn.Call(in)
}
