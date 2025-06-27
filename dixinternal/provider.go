package dixinternal

import (
	"fmt"
	"github.com/pubgo/funk/log"
	"github.com/pubgo/funk/v2/result"
	"reflect"
	"strings"

	"github.com/pubgo/funk/errors"
	"github.com/pubgo/funk/stack"
)

type providerInputType struct {
	typ      reflect.Type
	isMap    bool
	isList   bool
	isStruct bool
}

func (v providerInputType) Validate() error {
	if v.isMap && !isMapListSupportedType(v.typ) {
		return errors.Format("input map value type kind not support, kind=%s", v.typ.Kind().String())
	}

	if v.isList && !isMapListSupportedType(v.typ) {
		return errors.Format("input list element value type kind not support, kind=%s", v.typ.Kind().String())
	}

	if !isMapListSupportedType(v.typ) {
		return errors.Format("input value type kind not support, kind=%s", v.typ.Kind().String())
	}

	return nil
}

type providerOutputType struct {
	typ    reflect.Type
	isMap  bool
	isList bool
	// isStruct bool
}

type providerFn struct {
	fn        reflect.Value
	inputList []*providerInputType
	output    *providerOutputType

	hasError bool
}

func (n providerFn) call(in []reflect.Value) (r result.Result[[]reflect.Value]) {
	return result.WrapFn(func() ([]reflect.Value, error) { return n.fn.Call(in), nil }).
		InspectErr(func(err error) {
			log.Err(err).
				Any("fn_stack", stack.CallerWithFunc(n.fn)).
				Any("fn_type", n.fn.Type().String()).
				Any("input", fmt.Sprintf("%v", in)).
				Any("input_data", reflectValueToString(in)).
				Any("input_types", reflectTypesToString(n.inputList)).
				Any("output_type", n.output.typ.String()).
				Msgf("failed to invoke provider")
		})
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
