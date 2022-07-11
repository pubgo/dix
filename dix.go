package dix

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pubgo/funk"
	"github.com/pubgo/funk/xerr"
	"k8s.io/klog/v2"
)

const (
	defaultKey = "default"
)

type (
	group = string
	key   = reflect.Type
	value = reflect.Value
)

type Dix struct {
	option    Options
	providers map[key][]*node
	objects   map[key]map[group][]value
}

func (x *Dix) Option() Options {
	return x.option
}

func (x *Dix) handleOutput(output []reflect.Value) map[group][]value {
	var out = output[0]
	var rr = make(map[group][]value)
	switch out.Kind() {
	case reflect.Map:
		for _, k := range out.MapKeys() {
			var mapK = strings.TrimSpace(k.String())
			if mapK == "" {
				mapK = defaultKey
			}

			var val = out.MapIndex(k)
			if !val.IsValid() || val.IsNil() {
				continue
			}

			rr[mapK] = append(rr[mapK], val)
		}
	case reflect.Slice:
		for i := 0; i < out.Len(); i++ {
			var val = out.Index(i)
			if !val.IsValid() || val.IsNil() {
				continue
			}

			rr[defaultKey] = append(rr[defaultKey], val)
		}
	default:
		if out.IsValid() && !out.IsNil() {
			rr[defaultKey] = []value{out}
		}
	}
	return rr
}

func (x *Dix) evalProvider(typ key, opt Options) map[group][]value {
	switch typ.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Func:
	default:
		funk.Must(&Err{
			Msg:    "provider type kind error, the supported type kinds are <ptr,interface,func>",
			Detail: fmt.Sprintf("type=%s kind=%s", typ, typ.Kind()),
		})
	}

	funk.AssertErr(len(x.providers[typ]) == 0, &Err{
		Msg:    "provider not found, please check whether the provider imports or type error",
		Detail: fmt.Sprintf("type=%s kind=%s", typ, typ.Kind()),
	})

	if x.objects[typ] == nil {
		x.objects[typ] = make(map[group][]value)
	}

	if val := x.objects[typ]; len(val) != 0 {
		return val
	}

	objects := make(map[group][]value)
	for _, n := range x.providers[typ] {
		var input []reflect.Value
		for _, in := range n.input {
			input = append(input, x.getValue(in.typ, opt, in.isMap, in.isList))
		}

		for k, v := range x.handleOutput(n.call(input)) {
			if n.output.isMap {
				if _, ok := objects[k]; ok {
					klog.Warningf("type value exists, type=%s key=%s\n", typ, k)
				}
			}
			objects[k] = append(objects[k], v...)
		}
	}

	funk.AssertErr(len(objects) == 0, &Err{
		Msg:    "provider values is zero, please check whether the provider imports",
		Detail: fmt.Sprintf("type=%s kind=%s", typ, typ.Kind()),
	})

	x.objects[typ] = objects
	return objects
}

func (x *Dix) getValue(typ reflect.Type, opt Options, isMap bool, isList bool) reflect.Value {
	switch {
	case isMap:
		return makeMap(x.evalProvider(typ, opt))
	case isList:
		valMap := x.evalProvider(typ, opt)
		if valList, ok := valMap[defaultKey]; !ok || len(valList) == 0 {
			panic(&Err{
				Msg:    "slice: provider default value not found",
				Detail: fmt.Sprintf("type=%s, allValues=%v", typ, valMap),
			})
		} else {
			return makeList(valMap[defaultKey])
		}
	case typ.Kind() == reflect.Struct:
		var v = reflect.New(typ)
		x.injectStruct(v.Elem(), opt)
		return v.Elem()
	default:
		valMap := x.evalProvider(typ, opt)
		if valList, ok := valMap[defaultKey]; !ok || len(valList) == 0 {
			panic(&Err{
				Msg:    "provider default value not found",
				Detail: fmt.Sprintf("type=%s, allValues=%v", typ, valMap),
			})
		} else {
			return valList[len(valList)-1]
		}
	}
}

func (x *Dix) injectFunc(vp reflect.Value, opt Options) {
	var inTypes []*inType
	for i := 0; i < vp.Type().NumIn(); i++ {
		switch inTyp := vp.Type().In(i); inTyp.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Func, reflect.Struct:
			inTypes = append(inTypes, &inType{typ: inTyp})
		case reflect.Map:
			inTypes = append(inTypes, &inType{typ: inTyp.Elem(), isMap: true})
		case reflect.Slice:
			inTypes = append(inTypes, &inType{typ: inTyp.Elem(), isList: true})
		default:
			panic(&Err{Msg: "incorrect input type", Detail: fmt.Sprintf("inTyp=%s kind=%s", inTyp, inTyp.Kind())})
		}
	}

	var input []reflect.Value
	for _, in := range inTypes {
		input = append(input, x.getValue(in.typ, opt, in.isMap, in.isList))
	}
	vp.Call(input)
}

func (x *Dix) injectStruct(vp reflect.Value, opt Options) {
	tp := vp.Type()
	for i := 0; i < tp.NumField(); i++ {
		if !vp.Field(i).CanSet() {
			continue
		}

		field := tp.Field(i)
		if field.Anonymous {
			continue
		}

		switch field.Type.Kind() {
		case reflect.Struct:
			x.injectStruct(vp.Field(i), opt)
		case reflect.Interface, reflect.Ptr, reflect.Func:
			vp.Field(i).Set(x.getValue(field.Type, opt, false, false))
		case reflect.Map:
			vp.Field(i).Set(x.getValue(field.Type.Elem(), opt, true, false))
		case reflect.Slice:
			vp.Field(i).Set(x.getValue(field.Type.Elem(), opt, false, true))
		}
	}
}

func (x *Dix) inject(param interface{}, opts ...Option) interface{} {
	funk.Assert(param == nil, "param is null")

	var opt Options
	for i := range opts {
		opts[i](&opt)
	}

	vp := reflect.ValueOf(param)
	funk.AssertErr(!vp.IsValid() || vp.IsNil(), &Err{
		Msg:    "param should not be invalid or nil",
		Detail: fmt.Sprintf("param=%#v", param),
	})

	if vp.Kind() == reflect.Func {
		funk.Assert(vp.Type().NumOut() != 0, "the func of provider output num should be zero")
		x.injectFunc(vp, opt)
		return nil
	}

	funk.AssertErr(vp.Kind() != reflect.Ptr, &Err{
		Msg:    "param should be ptr type",
		Detail: fmt.Sprintf("param=%#v", param),
	})

	for i := 0; i < vp.NumMethod(); i++ {
		var name = vp.Type().Method(i).Name
		if !strings.HasPrefix(name, "DixInject") {
			continue
		}

		x.injectFunc(vp.Method(i), opt)
	}

	for vp.Kind() == reflect.Ptr {
		vp = vp.Elem()
	}

	funk.AssertErr(vp.Kind() != reflect.Struct, &Err{
		Msg:    "param should be struct ptr type",
		Detail: fmt.Sprintf("param=%#v", param),
	})

	x.injectStruct(vp, opt)
	return param
}

func (x *Dix) provider(param interface{}) {
	funk.Assert(param == nil, "param is null")

	fnVal := reflect.ValueOf(param)
	funk.AssertErr(!fnVal.IsValid() || fnVal.IsZero(), &Err{
		Msg:    "param should not be invalid or nil",
		Detail: fmt.Sprintf("param=%#v", param),
	})

	funk.AssertErr(fnVal.Kind() != reflect.Func, &Err{
		Msg:    "param should be function type",
		Detail: fmt.Sprintf("param=%#v", param),
	})

	typ := fnVal.Type()
	funk.Assert(typ.IsVariadic(), "the func of provider variable parameters are not allowed")
	funk.Assert(typ.NumOut() == 0, "the func of provider output num should not be zero")

	var n = &node{fn: fnVal}
	for i := 0; i < typ.NumIn(); i++ {
		switch inTye := typ.In(i); inTye.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Func, reflect.Struct:
			n.input = append(n.input, &inType{typ: inTye})
		case reflect.Map:
			n.input = append(n.input, &inType{typ: inTye.Elem(), isMap: true})
		case reflect.Slice:
			n.input = append(n.input, &inType{typ: inTye.Elem(), isList: true})
		default:
			panic(&Err{Msg: "incorrect input type", Detail: fmt.Sprintf("inTyp=%s kind=%s", inTye, inTye.Kind())})
		}
	}

	switch outTyp := typ.Out(0); outTyp.Kind() {
	case reflect.Slice:
		n.output = &outType{isList: true, typ: outTyp.Elem()}
	case reflect.Map:
		n.output = &outType{isMap: true, typ: outTyp.Elem()}
	case reflect.Ptr, reflect.Interface, reflect.Func:
		n.output = &outType{isList: true, typ: outTyp}
	default:
		panic(&Err{Msg: "incorrect output type", Detail: fmt.Sprintf("ouTyp=%s kind=%s", outTyp, outTyp.Kind())})
	}

	x.providers[n.output.typ] = append(x.providers[n.output.typ], n)

	dep, ok := x.isCycle()
	funk.AssertErr(ok, &Err{
		Msg:    "provider circular dependency",
		Detail: dep,
	})
}

func (x *Dix) dix(opts ...Option) *Dix {
	var sub = newDix(opts...)

	for k, v := range x.providers {
		sub.providers[k] = append(sub.providers[k], v...)
	}

	for k, v := range x.objects {
		if sub.objects[k] == nil {
			sub.objects[k] = make(map[group][]value)
		}
		for k1, v1 := range v {
			sub.objects[k][k1] = append(sub.objects[k][k1], v1...)
		}
	}

	return sub
}

func newDix(opts ...Option) *Dix {
	var option = Options{}
	defer funk.RecoverAndRaise(func(err xerr.XErr) xerr.XErr {
		return err.WrapF("options=%#v\n", option)
	})

	for i := range opts {
		opts[i](&option)
	}

	option.Check()

	c := &Dix{
		option:    option,
		providers: make(map[key][]*node),
		objects:   make(map[key]map[group][]value),
	}

	return c
}
