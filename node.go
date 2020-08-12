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

func newNode(c *dix, data interface{}) *node {
	return &node{fn: reflect.ValueOf(data), c: c}
}

func (n *node) handleCall(params []reflect.Value) error {
	values := n.c.invokerFn(n.fn, params[:])

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

	return n.c.dix(vas...)
}

func (n *node) isNil(v reflect.Value) bool {
	return n.c.isNil(v)
}

type sortValue struct {
	Key   string
	Value reflect.Value
}

func (n *node) call() error {
	var values []reflect.Value
	var params []reflect.Value
	var input []reflect.Value
	for i := 0; i < n.fn.Type().NumIn(); i++ {
		inType := n.fn.Type().In(i)
		switch inType.Kind() {
		case reflect.Ptr:
			val := n.c.getValue(unWrapType(inType), _default)
			if n.isNil(val) {
				return nil
			}

			mt := reflect.New(unWrapType(inType))
			mt.Elem().Set(val.Elem())
			params = append(params, mt)
			input = append(input, mt)
		case reflect.Map:
			tye := unWrapType(inType.Key())
			mt := reflect.MakeMap(inType)

			var sv []sortValue
			for k, v := range n.c.values[tye] {
				if n.isNil(v) {
					return nil
				}
				values = append(values, v)
				mt.SetMapIndex(v, reflect.ValueOf(k))
				sv = append(sv, sortValue{
					Key:   k,
					Value: v,
				})
			}

			sort.Slice(sv, func(i, j int) bool {
				return sv[i].Key > sv[j].Key
			})
			for i := range sv {
				input = append(input, sv[i].Value)
			}
			params = append(params, mt)
		case reflect.Struct:
			mt := reflect.New(inType)
			var sv []sortValue
			for i := 0; i < inType.NumField(); i++ {
				val := n.c.getValue(unWrapType(inType.Field(i).Type), n.c.getTagVal(inType.Field(i)))
				if n.isNil(val) {
					return nil
				}
				values = append(values, val)
				mt.Field(i).Set(val)

				sv = append(sv, sortValue{
					Key:   n.c.getTagVal(inType.Field(i)),
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
