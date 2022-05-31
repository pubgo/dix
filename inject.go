package dix

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/pubgo/dix/internal/assert"
)

const (
	Default = "default"
)

type (
	group = string
	key   = reflect.Type
	value = reflect.Value
)

type dix struct {
	option    Options
	invokes   []*node
	providers map[key][]*node
	objects   map[key]map[group]value
}

func (x *dix) handleOutput(output []reflect.Value) map[group]value {
	var typ reflect.Type
	var out = output[0]
	var rr = make(map[group]value)
	switch out.Kind() {
	case reflect.Map:
		for _, k := range out.MapKeys() {
			rr[k.String()] = out.MapIndex(k)
		}
		typ = out.Type().Elem()
	default:
		rr[Default] = out
		typ = out.Type()
	}

	for k, v := range rr {
		if !v.IsValid() {
			continue
		}

		if v.IsNil() {
			continue
		}

		if x.objects[typ] == nil {
			x.objects[typ] = make(map[group]value)
		}

		if _, ok := x.objects[typ][k]; ok {
			continue
		}

		x.objects[typ][k] = v
	}
	return x.objects[typ]
}

func (x *dix) evalProvider(typ key) map[group]value {
	if x.objects[typ] == nil {
		x.objects[typ] = make(map[group]value)
	}

	if val := x.objects[typ]; len(val) != 0 {
		return val
	}

	assert.Assert(len(x.providers[typ]) == 0, &Err{
		Msg:    "provider not found",
		Detail: fmt.Sprintf("type=%s", typ),
	})

	var rr = make(map[group]value)
	for _, n := range x.providers[typ] {
		var input []reflect.Value
		for i := range n.input {
			valMap := x.evalProvider(n.input[i].typ)
			if len(valMap) == 0 {
				continue
			}

			if n.input[i].isMap {
				input = append(input, makeMap(valMap))
			} else {
				input = append(input, valMap[Default])
			}
		}

		for k, v := range x.handleOutput(n.fn.Call(input)) {
			if _, ok := rr[k]; ok {
				continue
			}

			rr[k] = v
		}
	}
	return rr
}

func (x *dix) inject(param interface{}) {
	assert.Assertf(param == nil, "param is null")

	vp := reflect.ValueOf(param)
	assert.Assert(!vp.IsValid() || vp.IsNil(), &Err{
		Msg:    "param should not be invalid or nil",
		Detail: fmt.Sprintf("param=%#v", param),
	})
	assert.Assert(vp.Kind() != reflect.Ptr, &Err{
		Msg:    "param should be ptr type",
		Detail: fmt.Sprintf("param=%#v", param),
	})

	for vp.Kind() == reflect.Ptr {
		vp = vp.Elem()
	}

	assert.Assert(vp.Kind() != reflect.Struct, &Err{
		Msg:    "param should be struct ptr type",
		Detail: fmt.Sprintf("param=%#v", param),
	})

	tp := vp.Type()
	for i := 0; i < tp.NumField(); i++ {
		field := vp.Field(i)
		if !field.CanSet() {
			continue
		}

		inTye := tp.Field(i)
		tagVal, ok := inTye.Tag.Lookup(x.option.tagName)
		if !ok {
			continue
		}

		if tagVal == "" {
			tagVal = Default
		} else {
			tagVal = os.Expand(tagVal, func(s string) string {
				if !strings.HasPrefix(s, ".") {
					return os.Getenv(strings.ToUpper(s))
				}

				var out, err = templates(fmt.Sprintf("${%s}", s), param)
				assert.AssertFn(err != nil, func() error {
					return &Err{
						Err:    err,
						Msg:    "expr eval failed",
						Detail: fmt.Sprintf("param=%#v", param),
					}
				})
				return fmt.Sprintf("%v", out)
			})
		}

		switch field.Kind() {
		case reflect.Interface, reflect.Ptr:
			valMap := x.evalProvider(field.Type())
			assert.Assert(len(valMap) == 0, &Err{
				Msg:    "provider value not found",
				Detail: fmt.Sprintf("type=%s", field.Type()),
			})

			if _, ok := valMap[tagVal]; !ok {
				panic(&Err{
					Msg:    "default value not found",
					Detail: fmt.Sprintf("all values=%v", valMap),
				})
			}

			field.Set(valMap[tagVal])
		case reflect.Map:
			valMap := x.evalProvider(field.Type().Elem())
			assert.Assert(len(valMap) == 0, &Err{
				Msg:    "provider value not found",
				Detail: fmt.Sprintf("type=%s", inTye),
			})
			field.Set(makeMap(valMap))
		}
	}
}

func (x *dix) invoke() {
	for _, n := range x.invokes {
		var input []reflect.Value
		for _, in := range n.input {
			valMap := x.evalProvider(in.typ)
			assert.Assert(len(valMap) == 0, &Err{
				Msg:    "provider value is null",
				Detail: fmt.Sprintf("type=%s", in.typ),
			})

			if in.isMap {
				input = append(input, makeMap(valMap))
			} else {
				input = append(input, valMap[Default])
			}
		}
		n.fn.Call(input)
	}
}

func (x *dix) register(param interface{}) {
	assert.Assertf(param == nil, "param is null")

	fnVal := reflect.ValueOf(param)
	assert.Assert(!fnVal.IsValid() || fnVal.IsZero(), &Err{
		Msg:    "param should not be invalid or nil",
		Detail: fmt.Sprintf("param=%#v", param),
	})
	assert.Assert(fnVal.Kind() != reflect.Func, &Err{
		Msg:    "param should be function type",
		Detail: fmt.Sprintf("param=%#v", param),
	})

	typ := fnVal.Type()
	assert.Assertf(typ.IsVariadic(), "the func of provider variable parameters are not allowed")

	var n = &node{fn: fnVal}
	if typ.NumOut() != 0 {
		n.output = new(outType)
		var retTyp = typ.Out(0)
		switch retTyp.Kind() {
		case reflect.Map:
			n.output.isMap = true
			n.output.typ = retTyp.Elem()
		case reflect.Ptr, reflect.Interface:
			n.output.typ = retTyp
		default:
			panic(&Err{Msg: "ret type error", Detail: fmt.Sprintf("retTyp=%s", retTyp)})
		}
		x.providers[n.output.typ] = append(x.providers[n.output.typ], n)
	} else {
		assert.Assertf(typ.NumIn() == 0, "the func of provider input num should not be zero")
		x.invokes = append(x.invokes, n)
	}

	for i := 0; i < typ.NumIn(); i++ {
		switch inTye := typ.In(i); inTye.Kind() {
		case reflect.Interface, reflect.Ptr:
			n.input = append(n.input, &inType{typ: inTye})
		case reflect.Map:
			n.input = append(n.input, &inType{typ: inTye.Elem(), isMap: true})
		default:
			panic(&Err{Msg: "incorrect input type", Detail: fmt.Sprintf("inTye=%s", inTye)})
		}
	}
}

func newDix(opts ...Option) *dix {
	var option = Options{
		tagName: "inject",
	}

	for i := range opts {
		opts[i](&option)
	}

	c := &dix{
		providers: make(map[key][]*node),
		objects:   make(map[key]map[group]value),
		option:    option,
	}

	return c
}

//func (x *dix) Graph() string                { return x.graph() }
//func (x *dix) Json() map[string]interface{} { return x.json() }
