package dix

import (
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/pubgo/xerror"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	_default = group("default")
	_tagName = "dix"
)

type (
	group = string
	key   = reflect.Type
	abc   = reflect.Type
	value = reflect.Value
)

type Option func(*Options)
type Options struct{}

type dix struct {
	invokes   []*node
	providers map[key][]*node
	objects   map[key]map[group]value

	// providers中保存的是, 类型对应的providers
	// provider的返回值是具体的值
	providers1 map[key]map[group][]*node

	// abcProviders中保存的是, 类型对应的providers
	// provider的返回值是接口的实现
	// 可以有多provider的返回值是接口的实现
	abcProviders map[key]map[group][]*node

	// values中保存的是, 类型对应的各个group的具体的value
	values map[key]map[group]value

	// abcValues中保存的是, 接口类型对应实现的各个group的value的type
	// 通过type去dix.values中获取具体的value
	abcValues map[abc]map[group]key
}

func defaultInvoker(fn reflect.Value, args []reflect.Value) (_ []reflect.Value, gErr error) {
	defer xerror.Resp(func(err xerror.XErr) {
		gErr = err.WrapF("caller: %s args: %v", callerWithFunc(fn), args)
	})

	xerror.Assert(fn.IsNil(), "[fn] is nil")

	return fn.Call(args), nil
}

func (x *dix) getValue(tye key, name group) reflect.Value {
	if x.values[tye] == nil {
		return reflect.ValueOf((*error)(nil))
	}

	return x.values[tye][name]
}

func (x *dix) getAbcValue(tye key, name group) reflect.Value {
	if x.abcValues[tye] != nil {
		return x.values[x.abcValues[tye][name]][name]
	}

	for k := range x.values {
		if reflect.New(k).Type().Implements(tye) {
			return x.values[k][name]
		}
	}

	return reflect.Value{}
}

func (x *dix) getNodes(tye key, name group) []*node {
	if x.providers1[tye] == nil {
		return nil
	}
	return x.providers1[tye][name]
}

// isNil check whether params contain nil value
func (x *dix) isNil(v reflect.Value) bool {
	return v.IsNil()
}

// 检测是否是否个接口的实现
func (x *dix) getAbcType(p reflect.Type) reflect.Type {
	for k := range x.abcProviders {
		if reflect.New(p).Type().Implements(k) {
			return k
		}
	}
	return nil
}

func (x *dix) dixFunc(data reflect.Value) (err error) {
	defer xerror.RespErr(&err)

	fnVal := data
	tye := fnVal.Type()

	xerror.Assert(tye.IsVariadic(), "the func of provider variable parameters are not allowed")
	xerror.Assert(tye.NumIn() == 0, "the number of parameters should not be 0")
	xerror.Assert(tye.NumOut() > 0 && !isError(tye.Out(tye.NumOut()-1)), "the last returned value should be error type")

	for i := 0; i < tye.NumIn(); i++ {
		switch inTye := tye.In(i); inTye.Kind() {
		case reflect.Interface:
			nd, err := newNode(x, data)
			xerror.Panic(err)
			x.setAbcProvider(getIndirectType(inTye), _default, nd)
		case reflect.Ptr:
			nd, err := newNode(x, data)
			xerror.Panic(err)
			x.setProvider(getIndirectType(inTye), _default, nd)
		case reflect.Struct:
			for i := 0; i < inTye.NumField(); i++ {
				feTye := inTye.Field(i)

				if !x.hasNS(feTye) {
					continue
				}

				var kind = feTye.Type.Kind()
				if kind != reflect.Ptr && kind != reflect.Interface {
					continue
				}

				var ns = x.getNS(feTye)

				nd, err := newNode(x, data)
				xerror.Panic(err)

				if kind == reflect.Interface {
					x.setAbcProvider(getIndirectType(feTye.Type), ns, nd)
				} else {
					x.setProvider(getIndirectType(feTye.Type), ns, nd)
				}
			}
		default:
			return xerror.Fmt("incorrect input parameter type, got(%s)", inTye.Kind())
		}
	}
	return nil
}

func (x *dix) invoke1(param interface{}, options ...Option) (err error) {
	defer xerror.RespErr(&err)

	xerror.Assert(param == nil, "param is nil")

	vp := reflect.ValueOf(param)
	tp := vp.Type()

	xerror.Assert(vp.Kind() != reflect.Func, "param(%#v) should be func type", param)
	xerror.Assert(vp.IsNil(), "param is nil")

	xerror.Assert(tp.NumOut() != 0, "func output num should be zero")
	xerror.Assert(tp.NumIn() == 0, "func output num should be zero")
	vp = vp.Elem()

	var ns = _default

	typ := vp.Type()
	switch typ.Kind() {
	case reflect.Ptr:
		xerror.PanicF(x.dixPtrInvoke(vp, ns), "type: [%s] [%s]", typ.Name(), typ.String())
	case reflect.Struct:
		xerror.Panic(x.dixStructInvoke(vp))
	case reflect.Interface:
		xerror.Panic(x.dixInterfaceInvoke(vp, ns))
	default:
		return xerror.Fmt("invoke type kind(%v) error", typ.Kind())
	}

	return nil
}

func (x *dix) handleOutput(output []reflect.Value) map[group]value {
	var rr = make(map[group]value)
	var typ reflect.Type
	switch output[0].Kind() {
	case reflect.Map:
		for _, k := range output[0].MapKeys() {
			rr[k.String()] = output[0].MapIndex(k)
		}
		typ = output[0].Type().Elem()
	default:
		rr[_default] = output[0]
		typ = output[0].Type()
	}

	for k, v := range rr {
		if !v.IsValid() {
			continue
		}

		if v.IsNil() {
			continue
		}

		x.objects[typ][k] = v
	}
	return x.objects[typ]
}

func (x *dix) evalProvider(typ key) map[group]value {
	switch typ.Kind() {
	case reflect.Interface, reflect.Ptr:
		if x.objects[typ] == nil {
			x.objects[typ] = make(map[group]value)
		}

		if val := x.objects[typ]; val != nil {
			return val
		}

		if len(x.providers[typ]) == 0 {
			panic("typ providers not found")
		}

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
					input = append(input, valMap[_default])
				}
			}

			for k, v := range x.handleOutput(n.fn.Call(input)) {
				rr[k] = v
			}
		}

		for k, v := range rr {
			x.objects[typ][k] = v
		}
		return rr
	case reflect.Map:
		typ = typ.Elem()
	default:
		panic(&Err{Msg: "incorrect input parameter type error", Detail: fmt.Sprintf("inTye=%s", typ)})
	}
}

func (x *dix) inject(param interface{}) {
	xerror.Assert(param == nil, "param is nil")

	vp := reflect.ValueOf(param)
	tp := vp.Type()
	xerror.Assert(vp.Kind() != reflect.Ptr, "param(%#v) should be func type", param)
	xerror.Assert(vp.IsNil(), "param is nil")

	for i := 0; i < tp.NumField(); i++ {
		field := vp.Field(i)
		if !field.CanSet() {
			continue
		}

		switch inTye := tp.In(i); inTye.Kind() {
		case reflect.Interface, reflect.Ptr:
			valMap := x.evalProvider(inTye)
			if len(valMap) == 0 {
				panic("inTye not found")
			}
			field.Set(valMap[_default])
		case reflect.Map:
			inTye = inTye.Elem()
		default:
			panic(&Err{Msg: "incorrect input parameter type error", Detail: fmt.Sprintf("inTye=%s", inTye)})
		}
	}
}

func (x *dix) invoke(param interface{}, namespaces ...string) (err error) {
	defer xerror.RespErr(&err)

	vp := reflect.ValueOf(param)
	xerror.Assert(vp.Kind() != reflect.Ptr, "param(%#v) should be ptr type", param)
	vp = vp.Elem()

	var ns = _default
	if len(namespaces) > 0 && namespaces[0] != "" {
		ns = namespaces[0]
	}

	typ := vp.Type()
	switch typ.Kind() {
	case reflect.Ptr:
		xerror.PanicF(x.dixPtrInvoke(vp, ns), "type: [%s] [%s]", typ.Name(), typ.String())
	case reflect.Struct:
		xerror.Panic(x.dixStructInvoke(vp))
	case reflect.Interface:
		xerror.Panic(x.dixInterfaceInvoke(vp, ns))
	default:
		return xerror.Fmt("invoke type kind(%v) error", typ.Kind())
	}

	return nil
}

func (x *dix) dixNs(name string, param interface{}) (err error) {
	defer xerror.RespErr(&err)

	vp := reflect.ValueOf(param)
	xerror.Assert(!vp.IsValid() || vp.IsZero(), "[params] [%#v] should not be invalid or nil", param)

	var values = make(map[group][]reflect.Type)

	typ := vp.Type()
	switch typ.Kind() {
	case reflect.Interface:
		xerror.Panic(x.dixInterface(values, vp, name))
	case reflect.Ptr:
		xerror.Panic(x.dixPtr(values, vp, name))
	default:
		return xerror.Fmt("provide type kind error, (kind %v)", typ.Kind())
	}

	for gup, vas := range values {
		for i := range vas {
			getTy := getIndirectType(vas[i])

			for k, gNodes := range x.providers1 {
				// 类型相同
				if k == getTy && gNodes != nil {
					for _, v1 := range gNodes[gup] {
						xerror.ExitF(v1.call(), "fn:%s", callerWithFunc(v1.fn))
					}
				}

				// 实现接口
				if getTy.Kind() == reflect.Interface && !reflect.New(k).Type().Implements(getTy) {
					for _, v1 := range gNodes[gup] {
						xerror.ExitF(v1.call(), "fn:%s", callerWithFunc(v1.fn))
					}
				}
			}

			// interface
			for k, gNodes := range x.abcProviders {
				// 类型相同
				if k == getTy && gNodes != nil {
					for _, v1 := range gNodes[gup] {
						xerror.ExitF(v1.call(), "fn:%s", callerWithFunc(v1.fn))
					}
				}

				// 实现接口
				if reflect.New(getTy).Type().Implements(k) {
					for _, v1 := range gNodes[gup] {
						xerror.ExitF(v1.call(), "fn:%s", callerWithFunc(v1.fn))
					}
				}
			}
		}
	}

	return
}

func (x *dix) register(param interface{}) {
	xerror.Assert(param == nil, "param is null")

	fnVal := reflect.ValueOf(param)
	xerror.Assert(!fnVal.IsValid() || fnVal.IsZero(), "[params] [%#v] should not be invalid or nil", param)
	xerror.Assert(fnVal.Kind() != reflect.Func, "param should be a function")

	typ := fnVal.Type()
	xerror.Assert(typ.IsVariadic(), "the func of provider variable parameters are not allowed")
	xerror.Assert(typ.NumOut() == 0, "the number of parameters should not be 0")
	//xerror.Assert(typ.NumOut() > 0 && !isError(typ.Out(typ.NumOut()-1)), "the last returned value should be error type")

	var n = new(node)
	if typ.NumOut() != 0 {
		n.output = new(outType)
		var retTyp = typ.Out(0)
		switch retTyp.Kind() {
		case reflect.Map:
			n.output.isMap = true
			n.output.typ = retTyp.Elem()
		case reflect.Ptr:
			n.output.typ = retTyp
		default:
			panic(&Err{Msg: "ret type error", Detail: fmt.Sprintf("retTyp=%s", retTyp)})
		}

		x.providers[n.output.typ] = append(x.providers[n.output.typ], n)
	} else {
		x.invokes = append(x.invokes, n)
	}

	for i := 0; i < typ.NumIn(); i++ {
		switch inTye := typ.In(i); inTye.Kind() {
		case reflect.Interface, reflect.Ptr:
			n.input = append(n.input, &inType{typ: inTye})
		case reflect.Map:
			n.input = append(n.input, &inType{typ: inTye, isMap: true})
		default:
			panic(&Err{Msg: "incorrect input parameter type error", Detail: fmt.Sprintf("inTye=%s", inTye)})
		}
	}
}

func (x *dix) dix(param ...interface{}) (err error) {
	defer xerror.RespErr(&err)

	vp := reflect.ValueOf(param)
	xerror.Assert(!vp.IsValid() || vp.IsZero(), "[params] [%#v] should not be invalid or nil", param)
	xerror.Assert(vp.Kind() != reflect.Func, "param should be a function")

	for _, param := range params {
		vp := reflect.ValueOf(param)
		xerror.Assert(!vp.IsValid() || vp.IsZero(), "[params] [%#v] should not be invalid or nil", param)

		var values = make(map[group][]reflect.Type)

		typ := vp.Type()
		switch typ.Kind() {
		case reflect.Interface:
			xerror.Panic(x.dixInterface(values, vp))
		case reflect.Ptr:
			xerror.Panic(x.dixPtr(values, vp))
		case reflect.Func:
			xerror.Panic(x.dixFunc(vp))
		case reflect.Map:
			xerror.Panic(x.dixMap(values, param))
		case reflect.Struct:
			xerror.Panic(x.dixStruct(values, param))
		default:
			return xerror.Fmt("provide type kind error, (kind %v)", typ.Kind())
		}

		for gup, vas := range values {
			for i := range vas {
				getTy := getIndirectType(vas[i])

				for k, gNodes := range x.providers1 {
					// 类型相同
					if k == getTy && gNodes != nil {
						for _, v1 := range gNodes[gup] {
							xerror.ExitF(v1.call(), "fn:%s", callerWithFunc(v1.fn))
						}
					}

					// 实现接口
					if getTy.Kind() == reflect.Interface && !reflect.New(k).Type().Implements(getTy) {
						for _, v1 := range gNodes[gup] {
							xerror.ExitF(v1.call(), "fn:%s", callerWithFunc(v1.fn))
						}
					}
				}

				// interface
				for k, gNodes := range x.abcProviders {
					// 类型相同
					if k == getTy && gNodes != nil {
						for _, v1 := range gNodes[gup] {
							xerror.ExitF(v1.call(), "fn:%s", callerWithFunc(v1.fn))
						}
					}

					// 实现接口
					if reflect.New(getTy).Type().Implements(k) {
						for _, v1 := range gNodes[gup] {
							xerror.ExitF(v1.call(), "fn:%s", callerWithFunc(v1.fn))
						}
					}
				}
			}
		}
	}

	return nil
}

// 非接口类型map中保存值
func (x *dix) setValue(k key, name group, v value) {
	if x.values[k] == nil {
		x.values[k] = map[group]value{name: v}
	} else {
		x.values[k][name] = v
	}
}

// 在接口类型map中保存值
func (x *dix) setAbcValue(k key, name group, v key) {
	if x.abcValues[k] == nil {
		x.abcValues[k] = map[group]key{name: v}
	} else {
		x.abcValues[k][name] = v
	}
}

func (x *dix) hasNS(field reflect.StructField) bool {
	_, ok := field.Tag.Lookup(_tagName)
	return ok
}

func (x *dix) getWithVal(field reflect.StructField, val1 interface{}) string {
	// 如果结构体属性存在tag, 那么就获取tag
	// 不存在tag或者tag为空, 那么tag默认为default
	val, ok := field.Tag.Lookup(_tagName)
	val = strings.TrimSpace(val)
	if ok && val != "" {
		return os.Expand(val, func(s string) string {
			if !strings.HasPrefix(s, ".") {
				return os.Getenv(strings.ToUpper(s))
			}

			return templates(fmt.Sprintf("${%s}", s), val1)
		})
	}

	return _default
}

// 获取数据的分组或者namespace
func (x *dix) getNS(field reflect.StructField) string {
	// 如果结构体属性存在tag, 那么就获取tag
	// 不存在tag或者tag为空, 那么tag默认为default
	val, ok := field.Tag.Lookup(_tagName)
	val = strings.TrimSpace(val)
	if ok && val != "" {
		return val
	}

	return _default
}

func (x *dix) setProvider(k key, name group, nd *node) {
	if x.providers1[k] == nil {
		x.providers1[k] = map[group][]*node{name: {nd}}
	} else {
		x.providers1[k][name] = append(x.providers1[k][name], nd)
	}
}

func (x *dix) setAbcProvider(k key, name group, nd *node) {
	if x.abcProviders[k] == nil {
		x.abcProviders[k] = map[group][]*node{name: {nd}}
	} else {
		x.abcProviders[k][name] = append(x.abcProviders[k][name], nd)
	}
}

func newDix(opts ...Option) *dix {
	c := &dix{
		providers1:   make(map[key]map[group][]*node),
		abcProviders: make(map[key]map[group][]*node),
		values:       make(map[key]map[group]value),
		abcValues:    make(map[key]map[group]key),
	}
	return c
}

func (x *dix) Dix(data ...interface{}) error                  { return x.dix(data...) }
func (x *dix) Provider(data ...interface{}) error             { return x.dix(data...) }
func (x *dix) ProviderNs(name string, data interface{}) error { return x.dixNs(name, data) }
func (x *dix) Invoke(data interface{}, namespaces ...string) error {
	return x.invoke(data, namespaces...)
}
func (x *dix) Inject(data interface{}, namespaces ...string) error {
	return x.invoke(data, namespaces...)
}
func (x *dix) Graph() string                { return x.graph() }
func (x *dix) Json() map[string]interface{} { return x.json() }
