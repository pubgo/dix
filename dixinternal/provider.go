package dixinternal

import (
	"fmt"
	"reflect"
	"strings"
)

type providerInputType struct {
	typ      reflect.Type
	isMap    bool
	isList   bool
	isStruct bool
}

func (v providerInputType) Validate() error {
	if v.isMap && !isMapListSupportedType(v.typ) {
		return fmt.Errorf("input map value type kind not support, kind=%s", v.typ.Kind().String())
	}

	if v.isList && !isMapListSupportedType(v.typ) {
		return fmt.Errorf("input list element value type kind not support, kind=%s", v.typ.Kind().String())
	}

	if !isMapListSupportedType(v.typ) {
		return fmt.Errorf("input value type kind not support, kind=%s", v.typ.Kind().String())
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
	hasError  bool
}

func (n providerFn) call(in []reflect.Value) (outputs []reflect.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			maybePrintStack()
			if rErr, ok := r.(error); ok {
				err = rErr
			} else {
				err = fmt.Errorf("panic: %v", r)
			}

			logger.Error("failed to invoke provider",
				"error", err,
				"fn_name", GetFnName(n.fn),
				"fn_type", n.fn.Type().String(),
				"input_data", reflectValueToString(in),
				"input_types", reflectTypesToString(n.inputList),
				"output_type", n.output.typ.String())
		}
	}()

	return n.fn.Call(in), nil
}

// reflectTypesToString converts input type list to readable string
func reflectTypesToString(types []*providerInputType) string {
	var builder strings.Builder
	for i, t := range types {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(t.typ.String())
		if t.isMap {
			builder.WriteString("(map)")
		}
		if t.isList {
			builder.WriteString("(list)")
		}
	}
	return builder.String()
}
