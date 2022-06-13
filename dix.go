package dix

import (
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/pubgo/xerror"
)

var logs = log.New(os.Stderr, "dix: ", log.LstdFlags|log.Lshortfile)

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
	var out = output[0]
	var rr = make(map[group]value)
	switch out.Kind() {
	case reflect.Map:
		for _, k := range out.MapKeys() {
			var mapK = k.String()
			if mapK == "" {
				mapK = Default
			}
			rr[mapK] = out.MapIndex(k)
		}
	default:
		rr[Default] = out
	}

	for k, v := range rr {
		if !v.IsValid() {
			delete(rr, k)
			continue
		}

		if v.IsNil() {
			delete(rr, k)
			continue
		}
	}
	return rr
}

func (x *dix) evalProvider(typ key, opt Options) map[group]value {
	if x.objects[typ] == nil {
		x.objects[typ] = make(map[group]value)
	}

	if val := x.objects[typ]; len(val) != 0 {
		return val
	}

	for _, n := range x.providers[typ] {
		var input []reflect.Value
		for i := range n.input {
			valMap := x.evalProvider(n.input[i].typ, opt)
			if len(valMap) == 0 {
				continue
			}

			if n.input[i].isMap {
				input = append(input, makeMap(valMap))
			} else {
				if _, ok := valMap[Default]; !ok {
					panic(&Err{
						Msg:    "[default] tag value not found",
						Detail: fmt.Sprintf("all values=%v", valMap),
					})
				}
				input = append(input, valMap[Default])
			}
		}

		if len(input) != len(n.input) {
			continue
		}

		for k, v := range x.handleOutput(n.call(input)) {
			if n.output.isList {
				if _, ok := x.objects[typ][Default]; !ok {
					x.objects[typ][Default] = v
				} else {
					x.objects[typ][Default] = reflect.AppendSlice(x.objects[typ][Default], v)
				}
				continue
			}

			if _, ok := x.objects[typ][k]; ok {
				logs.Printf("type value exists, type=%s key=%s\n", typ, k)
			}

			x.objects[typ][k] = v
		}
	}

	xerror.AssertErr(len(x.objects[typ]) == 0, &Err{
		Msg:    "all provider value is zero",
		Detail: fmt.Sprintf("type=%s", typ),
	})

	return x.objects[typ]
}

func (x *dix) injectFunc(vp reflect.Value, opt Options) {
	var inTypes []*inType
	typ := vp.Type()
	for i := 0; i < typ.NumIn(); i++ {
		switch inTye := typ.In(i); inTye.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Func:
			inTypes = append(inTypes, &inType{typ: inTye})
		case reflect.Map:
			inTypes = append(inTypes, &inType{typ: inTye.Elem(), isMap: true})
		case reflect.Slice:
			inTypes = append(inTypes, &inType{typ: inTye.Elem(), isList: true})
		default:
			panic(&Err{Msg: "incorrect input type", Detail: fmt.Sprintf("inTye=%s", inTye)})
		}
	}

	var input []reflect.Value
	for _, in := range inTypes {
		valMap := x.evalProvider(in.typ, opt)
		xerror.AssertErr(len(valMap) == 0, &Err{
			Msg:    "provider value is null",
			Detail: fmt.Sprintf("type=%s", in.typ),
		})

		if in.isMap {
			input = append(input, makeMap(valMap))
		} else {
			if _, _ok := valMap[Default]; !_ok {
				panic(&Err{
					Msg:    "default value not found",
					Detail: fmt.Sprintf("all values=%v", valMap),
				})
			}
			input = append(input, valMap[Default])
		}
	}
	vp.Call(input)
}

func (x *dix) injectStruct(vp reflect.Value, opt Options) {
	tp := vp.Type()
	for i := 0; i < tp.NumField(); i++ {
		field := vp.Field(i)
		if !field.CanSet() {
			continue
		}

		if tp.Field(i).Anonymous {
			continue
		}

		switch field.Kind() {
		case reflect.Struct:
			x.injectStruct(field, opt)
		case reflect.Interface, reflect.Ptr, reflect.Func:
			valMap := x.evalProvider(field.Type(), opt)
			xerror.AssertErr(len(valMap) == 0, &Err{
				Msg:    "provider value not found",
				Detail: fmt.Sprintf("type=%s", field.Type()),
			})

			if _, _ok := valMap[Default]; !_ok {
				panic(&Err{
					Msg:    "default value not found",
					Detail: fmt.Sprintf("all values=%v", valMap),
				})
			}

			field.Set(valMap[Default])
		case reflect.Map:
			valMap := x.evalProvider(field.Type().Elem(), opt)
			xerror.AssertErr(len(valMap) == 0, &Err{
				Msg:    "provider value not found",
				Detail: fmt.Sprintf("type=%s", field.Type()),
			})
			field.Set(makeMap(valMap))
		case reflect.Slice:
			valMap := x.evalProvider(field.Type().Elem(), opt)
			xerror.AssertErr(len(valMap) == 0, &Err{
				Msg:    "provider value not found",
				Detail: fmt.Sprintf("type=%s", field.Type()),
			})

			if _, _ok := valMap[Default]; !_ok {
				panic(&Err{
					Msg:    "default value not found",
					Detail: fmt.Sprintf("all values=%v", valMap),
				})
			}

			field.Set(valMap[Default])
		}
	}
}

func (x *dix) inject(param interface{}, opts ...Option) interface{} {
	xerror.Assert(param == nil, "param is null")

	var opt Options
	for i := range opts {
		opts[i](&opt)
	}

	vp := reflect.ValueOf(param)
	xerror.AssertErr(!vp.IsValid() || vp.IsNil(), &Err{
		Msg:    "param should not be invalid or nil",
		Detail: fmt.Sprintf("param=%#v", param),
	})

	if vp.Kind() == reflect.Func {
		x.injectFunc(vp, opt)
		return nil
	}

	xerror.AssertErr(vp.Kind() != reflect.Ptr, &Err{
		Msg:    "param should be ptr type",
		Detail: fmt.Sprintf("param=%#v", param),
	})

	for vp.Kind() == reflect.Ptr {
		vp = vp.Elem()
	}

	xerror.AssertErr(vp.Kind() != reflect.Struct, &Err{
		Msg:    "param should be struct ptr type",
		Detail: fmt.Sprintf("param=%#v", param),
	})

	x.injectStruct(vp, opt)
	return param
}

func (x *dix) invoke() {
	for _, n := range x.invokes {
		var input []reflect.Value
		for _, in := range n.input {
			valMap := x.evalProvider(in.typ, Options{})
			xerror.AssertErr(len(valMap) == 0, &Err{
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
	xerror.Assert(param == nil, "param is null")

	fnVal := reflect.ValueOf(param)
	xerror.AssertErr(!fnVal.IsValid() || fnVal.IsZero(), &Err{
		Msg:    "param should not be invalid or nil",
		Detail: fmt.Sprintf("param=%#v", param),
	})

	xerror.AssertErr(fnVal.Kind() != reflect.Func, &Err{
		Msg:    "param should be function type",
		Detail: fmt.Sprintf("param=%#v", param),
	})

	typ := fnVal.Type()
	xerror.Assert(typ.IsVariadic(), "the func of provider variable parameters are not allowed")

	var n = &node{fn: fnVal}
	if typ.NumOut() != 0 {
		n.output = new(outType)
		var retTyp = typ.Out(0)
		switch retTyp.Kind() {
		case reflect.Slice:
			n.output.isList = true
			n.output.typ = retTyp.Elem()
		case reflect.Map:
			n.output.isMap = true
			n.output.typ = retTyp.Elem()
		case reflect.Ptr, reflect.Interface, reflect.Func:
			n.output.typ = retTyp
		default:
			panic(&Err{Msg: "ret type error", Detail: fmt.Sprintf("retTyp=%s", retTyp)})
		}
		x.providers[n.output.typ] = append(x.providers[n.output.typ], n)
	} else {
		xerror.Assert(typ.NumIn() == 0, "the func of provider input num should not be zero")
		x.invokes = append(x.invokes, n)
	}

	for i := 0; i < typ.NumIn(); i++ {
		switch inTye := typ.In(i); inTye.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Func:
			n.input = append(n.input, &inType{typ: inTye})
		case reflect.Map:
			n.input = append(n.input, &inType{typ: inTye.Elem(), isMap: true})
		case reflect.Slice:
			n.input = append(n.input, &inType{typ: inTye.Elem(), isList: true})
		default:
			panic(&Err{Msg: "incorrect input type", Detail: fmt.Sprintf("inTye=%s", inTye)})
		}
	}

	dep, ok := x.isCycle()
	xerror.AssertErr(ok, &Err{
		Msg:    "provider circular dependency",
		Detail: dep,
	})
}

func newDix(opts ...Option) *dix {
	var option = Options{}
	defer xerror.RecoverAndRaise(func(err xerror.XErr) xerror.XErr {
		return err.WrapF("options=%#v\n", option)
	})

	for i := range opts {
		opts[i](&option)
	}

	option.Check()

	c := &dix{
		providers: make(map[key][]*node),
		objects:   make(map[key]map[group]value),
		option:    option,
	}

	return c
}
