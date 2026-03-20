package dixinternal

import (
	"fmt"
	"reflect"
	"strings"
	"time"
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

type providerCallResult struct {
	outputs []reflect.Value
	err     error
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
				"output_type", n.output.typ.String(),
				"hint", "provider 内部发生 panic：建议在 provider 内捕获异常并返回 error；可临时开启 debug 日志查看堆栈")
		}
	}()

	return n.fn.Call(in), nil
}

func (n providerFn) callWithTimeout(in []reflect.Value, timeout time.Duration) (outputs []reflect.Value, err error, timedOut bool) {
	if timeout <= 0 {
		outputs, err = n.call(in)
		return outputs, err, false
	}

	resultCh := make(chan providerCallResult, 1)
	go func() {
		outputs, callErr := n.call(in)
		resultCh <- providerCallResult{outputs: outputs, err: callErr}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case res := <-resultCh:
		return res.outputs, res.err, false
	case <-timer.C:
		return nil, fmt.Errorf("provider execution timeout after %s", timeout), true
	}
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
