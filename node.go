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

func (v inType) Validate() error {
	if v.isMap && !checkType(v.typ.Kind()) {
		return errors.New("input map value type kind not support, kind=%s", v.typ.Kind().String())
	}

	if v.isList && !checkType(v.typ.Kind()) {
		return errors.New("input list element value type kind not support, kind=%s", v.typ.Kind().String())
	}

	if v.typ.Kind() != reflect.Struct {

	}

	if !checkType(v.typ.Kind()) {
		return errors.New("input value type kind not support, kind=%s", v.typ.Kind().String())
	}

	return nil
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
	defer recovery.Raise(func(err error) error {
		return errors.WrapEventFn(err, func(evt *errors.Event) {
			evt.Str("msg", "failed to handle provider invoke")
			evt.Str("fn_stack", stack.CallerWithFunc(n.fn).String())
			evt.Str("input", fmt.Sprintf("%v", in))
			evt.Str("input_data", fmt.Sprintf("%v", in))
		})
	})

	return n.fn.Call(in)
}
