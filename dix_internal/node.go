package dix_internal

import (
	"fmt"
	"reflect"
	"strings"

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
		return errors.Format("input map value type kind not support, kind=%s", v.typ.Kind().String())
	}

	if v.isList && !checkType(v.typ.Kind()) {
		return errors.Format("input list element value type kind not support, kind=%s", v.typ.Kind().String())
	}

	if !checkType(v.typ.Kind()) {
		return errors.Format("input value type kind not support, kind=%s", v.typ.Kind().String())
	}

	return nil
}

type outType struct {
	typ    reflect.Type
	isMap  bool
	isList bool
}

type node struct {
	fn        reflect.Value
	inputList []*inType
	output    *outType
}

func (n node) call(in []reflect.Value) []reflect.Value {
	defer recovery.Raise(func(err error) error {
		return errors.WrapTag(err,
			errors.T("msg", "failed to handle provider invoke"),
			errors.T("fn_stack", stack.CallerWithFunc(n.fn).String()),
			errors.T("fn_type", n.fn.Type().String()),
			errors.T("input", fmt.Sprintf("%v", in)),
			errors.T("input_data", reflectValueToString(in)),
			errors.T("input_types", reflectTypesToString(n.inputList)),
			errors.T("output_type", n.output.typ.String()),
		)
	})

	return n.fn.Call(in)
}

// reflectTypesToString 将输入类型列表转换为可读字符串
func reflectTypesToString(types []*inType) string {
	var result strings.Builder
	for i, t := range types {
		if i > 0 {
			result.WriteString(", ")
		}
		result.WriteString(t.typ.String())
		if t.isMap {
			result.WriteString("(map)")
		}
		if t.isList {
			result.WriteString("(list)")
		}
	}
	return result.String()
}
