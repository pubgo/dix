package dixinternal

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pubgo/funk/errors"
	"github.com/pubgo/funk/recovery"
	"github.com/pubgo/funk/stack"
)

type providerInputType struct {
	typ    reflect.Type
	isMap  bool
	isList bool
}

func (v providerInputType) Validate() error {
	if v.isMap && !isMapListSupportedType(v.typ.Kind()) {
		return errors.Format("input map value type kind not support, kind=%s", v.typ.Kind().String())
	}

	if v.isList && !isMapListSupportedType(v.typ.Kind()) {
		return errors.Format("input list element value type kind not support, kind=%s", v.typ.Kind().String())
	}

	if !isMapListSupportedType(v.typ.Kind()) {
		return errors.Format("input value type kind not support, kind=%s", v.typ.Kind().String())
	}

	return nil
}

type providerOutputType struct {
	typ    reflect.Type
	isMap  bool
	isList bool
}

type providerFn struct {
	fn        reflect.Value
	inputList []*providerInputType
	output    *providerOutputType

	hasError    bool
	initialized bool
}

func (n providerFn) call(in []reflect.Value) []reflect.Value {
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

// reflectTypesToString converts input type list to readable string
func reflectTypesToString(types []*providerInputType) string {
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
