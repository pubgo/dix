package dix_inter

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
		return fmt.Errorf("input map value type kind not support, kind=%s", v.typ.Kind().String())
	}

	if v.isList && !checkType(v.typ.Kind()) {
		return fmt.Errorf("input list element value type kind not support, kind=%s", v.typ.Kind().String())
	}

	if v.typ.Kind() != reflect.Struct {

	}

	if !checkType(v.typ.Kind()) {
		return fmt.Errorf("input value type kind not support, kind=%s", v.typ.Kind().String())
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
		return errors.WrapTag(err,
			errors.T("msg", "failed to handle provider invoke"),
			errors.T("fn_stack", stack.CallerWithFunc(n.fn).String()),
			errors.T("input", fmt.Sprintf("%v", in)),
			errors.T("input_data", reflectValueToString(in)),
		)
	})

	return n.fn.Call(in)
}
