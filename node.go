package dix

import (
	"github.com/pubgo/xerror"
	"reflect"
	"sort"
)

type node struct {
	c          *dix
	fn         reflect.Value
	input      []value
	outputType map[ns]key
}

func newNode(c *dix, data interface{}) (nd *node, err error) {
	nd = &node{fn: reflect.ValueOf(data), c: c, outputType: make(map[ns]key)}
	nd.outputType, err = nd.returnedType()
	return
}

func (n *node) returnedType() (map[ns]key, error) {
	opts := make(map[ns]key)
	v := n.fn
	for i := 0; i < v.Type().NumOut(); i++ {
		out := v.Type().Out(i)
		switch out.Kind() {
		case reflect.Ptr:
			opts[_default] = unWrapType(out)
		case reflect.Struct:
			for j := 0; j < v.NumField(); j++ {
				feTye := v.Type().Field(j)
				if feTye.Type.Kind() != reflect.Ptr {
					return nil, xerror.New("the struct field should be Ptr type")
				}
				opts[n.c.getNS(feTye)] = unWrapType(feTye.Type)
			}
		default:
			if isError(out) {
				continue
			}
			return nil, xerror.Fmt("provide type kind error, (kind %v)", out.Kind())
		}
	}
	return opts, nil
}

func (n *node) handleCall(params []reflect.Value) (err error) {
	defer xerror.RespErr(&err)
	values := n.c.opts.InvokerFn(n.fn, params[:])

	if len(values) == 0 {
		return nil
	}

	// the returned value should be error
	if len(values) == 1 {
		if !isError(values[len(values)-1].Type()) {
			return xerror.New("the last returned value should be error type")
		}

		err, _ := values[0].Interface().(error)
		return xerror.Wrap(err)
	}

	var vas []interface{}
	for i := range values[:len(values)-1] {
		vas = append(vas, values[i].Interface())
	}

	return xerror.Wrap(n.c.dix(vas...))
}

func (n *node) isNil(v reflect.Value) bool {
	return n.c.isNil(v)
}

type sortValue struct {
	Key   string
	Value reflect.Value
}

func (n *node) call() (err error) {
	defer xerror.RespErr(&err)

	var values []reflect.Value
	var params []reflect.Value
	var input []reflect.Value
	for i := 0; i < n.fn.Type().NumIn(); i++ {
		inType := n.fn.Type().In(i)
		switch inType.Kind() {
		case reflect.Interface:
			val := n.c.getAbcValue(unWrapType(inType), _default)
			if !n.isNil(val) {
				params = append(params, val)
				input = append(input, val)
			}
		case reflect.Ptr:
			val := n.c.getValue(unWrapType(inType), _default)
			if !n.isNil(val) {
				params = append(params, val)
				input = append(input, val)
			}
		case reflect.Struct:
			mt := reflect.New(inType).Elem()
			var sv []sortValue
			for i := 0; i < inType.NumField(); i++ {
				field := inType.Field(i)

				if _, ok := field.Tag.Lookup(_tagName); n.c.opts.Strict && !ok {
					continue
				}

				var val reflect.Value
				if unWrapType(field.Type).Kind() == reflect.Interface {
					val = n.c.getAbcValue(unWrapType(field.Type), n.c.getNS(field))
					if n.isNil(val) {
						return nil
					}

					values = append(values, val)
					mt.Field(i).Set(val)
				} else {
					val = n.c.getValue(unWrapType(field.Type), n.c.getNS(field))
					if n.isNil(val) {
						return nil
					}

					values = append(values, val)
					mt.Field(i).Set(val)
				}

				sv = append(sv, sortValue{
					Key:   n.c.getNS(field),
					Value: val,
				})
			}

			sort.Slice(sv, func(i, j int) bool {
				return sv[i].Key > sv[j].Key
			})
			for i := range sv {
				input = append(input, sv[i].Value)
			}

			params = append(params, mt)
		default:
			return xerror.Fmt("incorrect input parameter type, got(%s)", inType.Kind())
		}
	}

	if reflect.DeepEqual(n.input, input) {
		return nil
	}

	if err := n.handleCall(params); err != nil {
		return xerror.Wrap(err)
	}
	n.input = input
	return nil
}
