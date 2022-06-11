package dix

import (
	"reflect"

	"github.com/pubgo/xerror"
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
	defer xerror.RecoverAndRaise(func(err xerror.XErr) xerror.XErr {
		err = err.WrapF("provider call failed")
		err = err.WrapF("provider is %s", callerWithFunc(n.fn))
		return err.WrapF("provider input is %v\n", in)
	})

	return n.fn.Call(in)
}
