package dix

import (
	"reflect"

	"github.com/pubgo/xerror"
)

type node struct {
	c          *dix
	fn         reflect.Value
	input      []value
	outputType map[group]key
}

func newNode(c *dix, data reflect.Value) (nd *node, err error) {
	nd = &node{fn: data, c: c, outputType: make(map[group]key)}
	nd.outputType, err = nd.returnedType()
	return
}

func (n *node) returnedType() (map[group]key, error) {
	retType := make(map[group]key)
	v := n.fn
	for i := 0; i < v.Type().NumOut(); i++ {
		out := v.Type().Out(i)
		switch out.Kind() {
		case reflect.Interface:
			retType[_default] = getIndirectType(out)
		case reflect.Ptr:
			retType[_default] = getIndirectType(out)
		case reflect.Map:
			//next := v.MapRange()
			//for next.Next() {
			//	feTye := v.Type().Field(j)
			//	xerror.Assert(feTye.Type.Kind() != reflect.Ptr,
			//		"the struct field[%s] should be Ptr type", feTye.Type.Kind())
			//	retType[n.c.getNS(feTye)] = getIndirectType(feTye.Type)
			//}
		case reflect.Struct:
			for j := 0; j < v.NumField(); j++ {
				feTye := v.Type().Field(j)
				xerror.Assert(feTye.Type.Kind() != reflect.Ptr && feTye.Type.Kind() != reflect.Interface,
					"the struct field[%s] should be Ptr or Interface type", feTye.Type.Kind())
				retType[n.c.getNS(feTye)] = getIndirectType(feTye.Type)
			}
		default:
			if isError(out) {
				continue
			}
			return nil, xerror.Fmt("provide type kind error, (kind %v)", out.Kind())
		}
	}
	return retType, nil
}

func (n *node) handleCall(params []reflect.Value) (gErr error) {
	defer xerror.RespErr(&gErr)

	values, err := defaultInvoker(n.fn, params[:])
	xerror.PanicF(err, "%s", params)

	if len(values) == 0 {
		return nil
	}

	// the returned value should be error
	vErr := values[len(values)-1]
	xerror.Assert(!isError(vErr.Type()), "the last returned value should be error type, got(%v)", vErr.Type())

	if vErr.IsValid() && !vErr.IsNil() {
		xerror.ExitF(vErr.Interface().(error), "func error, func: %s, params: %s", callerWithFunc(n.fn), params)
	}

	var vas []interface{}
	for i := range values[:len(values)-1] {
		vas = append(vas, values[i].Interface())
	}

	if len(vas) == 0 {
		return
	}

	xerror.ExitF(n.c.dix(vas...), "params: <%#v>", vas)
	return
}

func (n *node) isNil(v reflect.Value) bool {
	return n.c.isNil(v)
}

func (n *node) call() (err error) {
	defer xerror.RespErr(&err)

	var params []reflect.Value
	var input []reflect.Value
	for i := 0; i < n.fn.Type().NumIn(); i++ {
		inType := n.fn.Type().In(i)
		switch inType.Kind() {
		case reflect.Interface:
			val := n.c.getAbcValue(getIndirectType(inType), _default)
			if !n.isNil(val) {
				params = append(params, val)
				input = append(input, val)
			}
		case reflect.Ptr:
			val := n.c.getValue(getIndirectType(inType), _default)
			if !n.isNil(val) {
				params = append(params, val)
				input = append(input, val)
			}
		case reflect.Struct:
			mt := reflect.New(inType).Elem()
			for i := 0; i < inType.NumField(); i++ {
				field := inType.Field(i)

				if !n.c.hasNS(field) {
					continue
				}

				kind := field.Type.Kind()
				if kind != reflect.Interface && kind != reflect.Ptr {
					continue
				}

				// 结构体里面所有的属性值全部有值, 且不为nil
				var val reflect.Value
				if kind == reflect.Interface {
					val = n.c.getAbcValue(getIndirectType(field.Type), n.c.getNS(field))
				} else {
					val = n.c.getValue(getIndirectType(field.Type), n.c.getNS(field))
				}

				// 如果value为nil, 就不触发初始化
				if n.isNil(val) {
					return nil
				}

				xerror.TryThrow(func() { mt.Field(i).Set(val) }, "field: ", inType.Name(),".", field.Name)

				input = append(input, val)
			}

			params = append(params, mt)
		default:
			return xerror.Fmt("incorrect input parameter type, got(%s)", inType.Kind())
		}
	}

	if equal(n.input, input) {
		return nil
	}

	xerror.ExitF(n.handleCall(params), "input:%s ,params:%s", n.fn.Type().String(), params)

	n.input = input
	return
}
