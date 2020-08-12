package dix

import (
	"bytes"
	"fmt"
	"github.com/pubgo/xerror"
	"math/rand"
	"reflect"
)

const (
	_default = ns("_dix_default")
	_tagName = "dix"
)

type (
	ns        = string
	key       = reflect.Type
	value     = reflect.Value
	Option    func(c *dix)
	invokerFn = func(fn reflect.Value, args []reflect.Value) (results []reflect.Value)
)

type dix struct {
	providers       map[key]map[ns][]*node
	values          map[key]map[ns]reflect.Value
	rand            *rand.Rand
	invokerFn       invokerFn
	nilValueAllowed bool
}

func defaultInvoker(fn reflect.Value, args []reflect.Value) []reflect.Value {
	return fn.Call(args)
}

func (x *dix) getValue(tye key, _default ns) reflect.Value {
	if x.values[tye] == nil {
		return reflect.Value{}
	}
	return x.values[tye][_default]
}

func (x *dix) getNodeValue(tye key, _default ns) []*node {
	if x.providers[tye] == nil || x.providers[tye][_default] == nil {
		return nil
	}
	return x.providers[tye][_default]
}

// isNil check whether params contain nil value
func (x *dix) isNil(v reflect.Value) bool {
	if !x.nilValueAllowed {
		return v.IsNil()
	}
	return false
}

func (x *dix) dixPtr(values map[ns][]reflect.Type, data interface{}) error {
	val := reflect.ValueOf(data)
	if x.isNil(val) {
		return xerror.New("data is nil")
	}

	tye := unWrapType(val.Type())
	x.setValue(tye, _default, val)
	values[_default] = append(values[_default], val.Type())
	return nil
}

func (x *dix) dixFunc(data interface{}) error {
	fnVal := reflect.ValueOf(data)
	tye := fnVal.Type()

	if tye.IsVariadic() {
		return xerror.New("provide variable parameters are not allowed")
	}

	if tye.NumIn() == 0 {
		return xerror.New("the number of parameters should not be 0")
	}

	if tye.NumOut() > 0 {
		if !isError(tye.Out(tye.NumOut() - 1)) {
			return xerror.New("the last returned value should be error type")
		}
	}

	nd := newNode(x, data)
	for i := 0; i < tye.NumIn(); i++ {
		switch inTye := tye.In(i); inTye.Kind() {
		case reflect.Map:
			if inTye.Key().Kind() != reflect.Ptr {
				return xerror.New("the map key should be Ptr type")
			}

			x.setProvider(unWrapType(inTye.Key()), _default, nd)
		case reflect.Ptr:
			x.setProvider(unWrapType(inTye), _default, nd)
		case reflect.Struct:
			for i := 0; i < inTye.NumField(); i++ {
				feTye := inTye.Field(i)
				if feTye.Type.Kind() != reflect.Ptr {
					return xerror.New("the struct field should be Ptr type")
				}

				x.setProvider(unWrapType(feTye.Type), x.getTagVal(feTye), nd)
			}
		default:
			return fmt.Errorf("incorrect input parameter type, got(%s)", inTye.Kind())
		}
	}

	return nil
}

func (x *dix) dixMap(values map[ns][]reflect.Type, data interface{}) error {
	val := reflect.ValueOf(data)

	if val.Type().Key().Kind() != reflect.String {
		return xerror.New("the map key should be string type")
	}

	iter := val.MapRange()
	for iter.Next() {
		if iter.Value().Type().Kind() != reflect.Ptr {
			return xerror.New("the map value should be Ptr type")
		}

		k := iter.Key().String()
		if k == "" {
			return xerror.New("map key is null")
		}

		if x.isNil(iter.Value()) {
			return xerror.Fmt("map value is nil, key:%s", k)
		}

		x.setValue(unWrapType(iter.Value().Type()), k, iter.Value())
		values[k] = append(values[k], iter.Value().Type())
	}

	return nil
}

func (x *dix) dixStruct(values map[ns][]reflect.Type, data interface{}) error {
	val := reflect.ValueOf(data)
	tye := val.Type()

	for i := 0; i < tye.NumField(); i++ {
		if tye.Field(i).Type.Kind() != reflect.Ptr {
			return xerror.New("the struct field should be Ptr type")
		}

		if x.isNil(val.Field(i)) {
			return xerror.New("struct field data is nil")
		}

		x.setValue(unWrapType(tye.Field(i).Type), x.getTagVal(tye.Field(i)), val.Field(i))
		values[x.getTagVal(tye.Field(i))] = append(values[x.getTagVal(tye.Field(i))], val.Field(i).Type())
	}

	return nil
}

func (x *dix) dix(data ...interface{}) (err error) {
	defer xerror.RespErr(&err)

	if len(data) == 0 {
		return xerror.New("the num of dix input parameters should > 0")
	}

	var values = make(map[ns][]reflect.Type)
	for i := range data {
		dat := data[i]
		if dat == nil {
			return xerror.New("provide is nil")
		}

		typ := reflect.TypeOf(dat)
		if typ == nil {
			return xerror.New("provide type is nil")
		}

		switch typ.Kind() {
		case reflect.Ptr:
			if err := x.dixPtr(values, dat); err != nil {
				return xerror.Wrap(err)
			}
		case reflect.Func:
			err := x.dixFunc(dat)
			if err != nil {
				return xerror.Wrap(err)
			}
		case reflect.Map:
			if err := x.dixMap(values, dat); err != nil {
				return xerror.Wrap(err)
			}
		case reflect.Struct:
			if err := x.dixStruct(values, dat); err != nil {
				return xerror.Wrap(err)
			}
		default:
			return xerror.Fmt("provide type kind error, (kind %v)", typ.Kind())
		}
	}

	for name, vas := range values {
		for i := range vas {
			for _, n := range x.providers[unWrapType(vas[i])][name] {
				if err := n.call(); err != nil {
					return xerror.Wrap(err)
				}
			}
		}
	}

	return nil
}

func (x *dix) graph() string {
	b := &bytes.Buffer{}
	fPrintln(b, "nodes: {")
	for k, vs := range x.providers {
		for k1, v1 := range vs {
			for i := range v1 {
				fPrintln(b, "\t", k, "->", k1, "->", v1[i].fn.String())
			}
		}
	}
	fPrintln(b, "}")

	fPrintln(b, "values: {")
	for k, v := range x.values {
		for _, v1 := range v {
			fPrintln(b, "\t", k, "->", v, "->", v1.String())
		}
	}
	fPrintln(b, "}")

	return b.String()
}

func (x *dix) setValue(k key, name ns, v value) {
	if x.values[k] == nil {
		x.values[k] = map[ns]value{name: v}
	} else {
		x.values[k][name] = v
	}
}

func (x *dix) getTagVal(field reflect.StructField) string {
	if tag := field.Tag.Get(_tagName); tag != "" {
		return tag
	}
	return _default
}

func (x *dix) setProvider(k key, name ns, nd *node) {
	if x.providers[k] == nil {
		x.providers[k] = map[ns][]*node{name: {nd}}
	} else {
		x.providers[k][name] = append(x.providers[k][name], nd)
	}
}
