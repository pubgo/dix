package dix

import (
	"container/list"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/pubgo/xerror"
)

var logs = log.New(os.Stderr, "dix", log.LstdFlags|log.Llongfile)

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

func (x *dix) isCycle() (string, bool) {
	var types = make(map[reflect.Type]map[reflect.Type]bool)
	for _, nodes := range x.providers {
		for _, n := range nodes {
			if types[n.output.typ] == nil {
				types[n.output.typ] = make(map[reflect.Type]bool)
			}
			for i := range n.input {
				types[n.output.typ][n.input[i].typ] = true
			}
		}
	}

	var check func(root reflect.Type, data map[reflect.Type]bool, nodes *list.List) bool
	check = func(root reflect.Type, nodeTypes map[reflect.Type]bool, nodes *list.List) bool {
		for typ := range nodeTypes {
			nodes.PushBack(typ)
			if root == typ {
				return true
			}

			if check(root, types[typ], nodes) {
				return true
			}
			nodes.Remove(nodes.Back())
		}
		return false
	}

	var nodes = list.New()
	for root := range types {
		nodes.PushBack(root)
		if check(root, types[root], nodes) {
			break
		}
		nodes.Remove(nodes.Back())
	}

	if nodes.Len() == 0 {
		return "", false
	}

	var dep []string
	for nodes.Len() != 0 {
		dep = append(dep, nodes.Front().Value.(reflect.Type).String())
		nodes.Remove(nodes.Front())
	}

	return strings.Join(dep, " -> "), true
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

func (x *dix) evalProvider(typ key) map[group]value {
	xerror.AssertErr(len(x.providers[typ]) == 0, &Err{
		Msg:    "provider dependency not found",
		Detail: fmt.Sprintf("type=%s", typ),
	})

	if x.objects[typ] == nil {
		x.objects[typ] = make(map[group]value)
	}

	if val := x.objects[typ]; len(val) != 0 {
		return val
	}

	var values = make(map[group]value)
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

		for k, v := range x.handleOutput(n.call(input)) {
			if _, ok := values[k]; ok {
				logs.Printf("type value exists, type=%s key=%s\n", typ, k)
			}

			values[k] = v
		}
	}

	xerror.AssertErr(len(values) == 0, &Err{
		Msg:    "all provider value is zero",
		Detail: fmt.Sprintf("type=%s", typ),
	})

	for k, v := range values {
		if _, ok := x.objects[typ][k]; ok {
			logs.Printf("type value exists, type=%s key=%s\n", typ, k)
		}

		x.objects[typ][k] = v
	}
	return values
}

func (x *dix) inject(param interface{}) {
	xerror.Assert(param == nil, "param is null")

	vp := reflect.ValueOf(param)
	xerror.AssertErr(!vp.IsValid() || vp.IsNil(), &Err{
		Msg:    "param should not be invalid or nil",
		Detail: fmt.Sprintf("param=%#v", param),
	})
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
				xerror.AssertFn(err != nil, func() error {
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
			xerror.AssertErr(len(valMap) == 0, &Err{
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
			xerror.AssertErr(len(valMap) == 0, &Err{
				Msg:    "provider value not found",
				Detail: fmt.Sprintf("type=%s", field.Type()),
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
		xerror.Assert(typ.NumIn() == 0, "the func of provider input num should not be zero")
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

	dep, ok := x.isCycle()
	xerror.AssertErr(ok, &Err{
		Msg:    "provider circular dependency",
		Detail: dep,
	})
}

func newDix(opts ...Option) *dix {
	var option = Options{tagName: "inject"}
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
