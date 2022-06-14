package dix

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/pubgo/xerror"
)

var logs = log.New(os.Stderr, "dix: ", log.LstdFlags|log.Lshortfile)

const (
	defaultKey = "default"
)

type (
	group = string
	key   = reflect.Type
	value = reflect.Value
)

type dix struct {
	option    Options
	providers map[key][]*node
	objects   map[key]map[group][]value
}

func (x *dix) handleOutput(output []reflect.Value) map[group][]value {
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

func (x *dix) evalProvider(typ key, opt Options) map[group][]value {
	xerror.AssertErr(len(x.providers[typ]) == 0, &Err{
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
					logs.Printf("type value exists, type=%s key=%s\n", typ, k)
				}
			}
			objects[k] = append(objects[k], v...)
		}
	}

	xerror.AssertErr(len(objects) == 0, &Err{
		Msg:    "provider values is zero, please check whether the provider imports",
		Detail: fmt.Sprintf("type=%s kind=%s", typ, typ.Kind()),
	})

	x.objects[typ] = objects
	return objects
}

func (x *dix) getValue(typ reflect.Type, opt Options, isMap bool, isList bool) reflect.Value {
	valMap := x.evalProvider(typ, opt)
	switch {
	case isMap:
		return makeMap(valMap)
	case isList:
		if valList, ok := valMap[defaultKey]; !ok || len(valList) == 0 {
			panic(&Err{
				Msg:    "slice: provider default value not found",
				Detail: fmt.Sprintf("type=%s, allValues=%v", typ, valMap),
			})
		} else {
			return makeList(valMap[defaultKey])
		}
	default:
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

func (x *dix) injectFunc(vp reflect.Value, opt Options) {
	var inTypes []*inType
	for i := 0; i < vp.Type().NumIn(); i++ {
		switch inTyp := vp.Type().In(i); inTyp.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Func:
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

func (x *dix) injectStruct(vp reflect.Value, opt Options) {
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
		xerror.Assert(vp.Type().NumOut() != 0, "the func of provider output num should be zero")
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

func (x *dix) provider(param interface{}) {
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
	xerror.Assert(typ.NumOut() == 0, "the func of provider output num should not be zero")

	var n = &node{fn: fnVal}
	for i := 0; i < typ.NumIn(); i++ {
		switch inTye := typ.In(i); inTye.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Func:
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
	case reflect.Map:
		n.output = &outType{isMap: true, typ: outTyp.Elem()}
	case reflect.Ptr, reflect.Interface, reflect.Func:
		n.output = &outType{isList: true, typ: outTyp}
	case reflect.Slice:
		n.output = &outType{isList: true, typ: outTyp.Elem()}
	default:
		panic(&Err{Msg: "incorrect output type", Detail: fmt.Sprintf("ouTyp=%s kind=%s", outTyp, outTyp.Kind())})
	}

	x.providers[n.output.typ] = append(x.providers[n.output.typ], n)

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
		option:    option,
		providers: make(map[key][]*node),
		objects:   make(map[key]map[group][]value),
	}

	return c
}
