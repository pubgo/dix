package dix

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"time"

	"github.com/pubgo/dix/dix_opts"
	"github.com/pubgo/xerror"
)

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

type dix struct {
	opts dix_opts.Options

	// providers中保存的是, 类型对应的providers
	// provider的返回值是具体的值
	providers map[key]map[group][]*node

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

func defaultInvoker(fn reflect.Value, args []reflect.Value) []reflect.Value {
	defer xerror.RespRaise(func(err xerror.XErr) error { return xerror.WrapF(err, "caller: %s", callerWithFunc(fn)) })

	xerror.Assert(fn.IsZero(), "[fn] is nil")

	return fn.Call(args)
}

func (x *dix) getValue(tye key, name group) reflect.Value {
	if x.values[tye] == nil {
		return reflect.ValueOf((*error)(nil))
	}

	return x.values[tye][name]
}

func (x *dix) getAbcValue(tye key, name group) reflect.Value {
	if x.abcValues[tye] == nil {
		return reflect.Value{}
	}

	return x.values[x.abcValues[tye][name]][name]
}

func (x *dix) getNodes(tye key, name group) []*node {
	if x.providers[tye] == nil {
		return nil
	}
	return x.providers[tye][name]
}

// isNil check whether params contain nil value
func (x *dix) isNil(v reflect.Value) bool {
	if !x.opts.NilAllowed {
		return v.IsNil()
	}
	return false
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

				if getIndirectType(feTye.Type).Kind() == reflect.Interface {
					nd, err := newNode(x, data)
					xerror.Panic(err)
					x.setAbcProvider(getIndirectType(feTye.Type), x.getNS(feTye), nd)
					continue
				}

				if feTye.Type.Kind() != reflect.Ptr && feTye.Type.Kind() != reflect.Interface {
					continue
				}

				nd, err := newNode(x, data)
				xerror.Panic(err)
				x.setProvider(getIndirectType(feTye.Type), x.getNS(feTye), nd)
			}
		default:
			return xerror.Fmt("incorrect input parameter type, got(%s)", inTye.Kind())
		}
	}
	return nil
}

func (x *dix) init(opts ...dix_opts.Option) error {
	var dixOpt = x.opts
	for _, opt := range opts {
		opt(&dixOpt)
	}

	// TODO check option
	x.opts = dixOpt
	return nil
}

func (x *dix) invoke(params interface{}, namespaces ...string) (err error) {
	defer xerror.RespErr(&err)

	var ns = _default
	if len(namespaces) > 0 {
		ns = namespaces[0]
	}

	vp := reflect.ValueOf(params)

	xerror.Assert(vp.Type().Kind() != reflect.Ptr, "params(%#v) should be ptr type", params)

	typ := vp.Elem().Type()
	switch typ.Kind() {
	case reflect.Ptr:
		xerror.PanicF(x.dixPtrInvoke(vp, ns), "type: [%s] [%s]", typ.Name(), typ.String())
	case reflect.Struct:
		xerror.Panic(x.dixStructInvoke(vp))
	case reflect.Interface:
		xerror.Panic(x.dixInterfaceInvoke(vp, ns))
	default:
		return xerror.Fmt("invoke type kind error, (kind %v)", typ.Kind())
	}

	return nil
}

func (x *dix) dix(params ...interface{}) (err error) {
	defer xerror.RespErr(&err)

	xerror.Assert(len(params) == 0, "[params] should not be zero")

	for _, param := range params {
		vp := reflect.ValueOf(param)
		xerror.Assert(!vp.IsValid() || vp.IsZero(), "[params] should not be invalid or nil")

		var values = make(map[group][]reflect.Type)

		typ := vp.Type()
		switch typ.Kind() {
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
				for _, n := range x.providers[getTy][gup] {
					xerror.Panic(n.call())
				}

				// interface
				for t, mapNodes := range x.abcProviders {
					if !reflect.New(getTy).Type().Implements(t) {
						continue
					}

					for _, n := range mapNodes[gup] {
						xerror.Panic(n.call())
					}
				}
			}
		}
	}

	return nil
}

func (x *dix) graph() string {
	b := &bytes.Buffer{}
	fPrintln(b, "digraph G {")
	fPrintln(b, "\tsubgraph cluster_0 {")
	fPrintln(b, "\t\tlabel=nodes")
	for k, vs := range x.providers {
		for k1, v1 := range vs {
			for i := range v1 {
				fn := callerWithFunc(v1[i].fn)
				fPrintln(b, fmt.Sprintf("\t\t"+`"%s" -> %s -> "%s"`, k, k1, fn))
				for _, v2 := range v1[i].outputType {
					fPrintln(b, fmt.Sprintf("\t\t"+`"%s" -> %s -> "%s" -> "%s"`, k, k1, fn, v2))
				}
			}
		}
	}
	fPrintln(b, "\t}")

	fPrintln(b, "\tsubgraph cluster_1 {")
	fPrintln(b, "\t\tlabel=values")
	for k, v := range x.values {
		for k1, v1 := range v {
			fPrintln(b, fmt.Sprintf("\t\t"+`"%s" -> %s -> "%s"`, k, k1, v1.String()))
		}
	}
	fPrintln(b, "\t}")

	fPrintln(b, "\tsubgraph cluster_2 {")
	fPrintln(b, "\t\tlabel=abc_nodes")
	for k, vs := range x.abcProviders {
		for k1, v1 := range vs {
			for i := range v1 {
				fn := callerWithFunc(v1[i].fn)
				fPrintln(b, fmt.Sprintf("\t\t"+`"%s" -> %s -> "%s"`, k, k1, fn))
				for _, v2 := range v1[i].outputType {
					fPrintln(b, fmt.Sprintf("\t\t"+`"%s" -> %s -> "%s" -> "%s"`, k, k1, fn, v2))
				}
			}
		}
	}
	fPrintln(b, "\t}")

	fPrintln(b, "\tsubgraph cluster_3 {")
	fPrintln(b, "\t\tlabel=abc_values")
	for k, v := range x.abcValues {
		for k1, v1 := range v {
			fPrintln(b, fmt.Sprintf("\t\t"+`"%s" -> %s -> "%s"`, k, k1, v1.String()))
		}
	}
	fPrintln(b, "\t}")
	fPrintln(b, "}")

	return b.String()
}

func (x *dix) json() map[string]interface{} {
	var nodes []string
	var values []string
	var abcNodes []string
	var abcValues []string
	for k, vs := range x.providers {
		for k1, v1 := range vs {
			for i := range v1 {
				fn := callerWithFunc(v1[i].fn)
				nodes = append(nodes, fmt.Sprintf(`%s -- %s -- %s`, k, k1, fn))
				for _, v2 := range v1[i].outputType {
					nodes = append(nodes, fmt.Sprintf(`%s -- %s -- %s -- %s`, k, k1, fn, v2))
				}
			}
		}
	}

	for k, v := range x.values {
		for k1, v1 := range v {
			values = append(values, fmt.Sprintf(`%s -- %s -- %s`, k, k1, v1.String()))
		}
	}

	for k, vs := range x.abcProviders {
		for k1, v1 := range vs {
			for i := range v1 {
				fn := callerWithFunc(v1[i].fn)
				abcNodes = append(abcNodes, fmt.Sprintf(`%s -- %s -- %s`, k, k1, fn))
				for _, v2 := range v1[i].outputType {
					abcNodes = append(abcNodes, fmt.Sprintf(`%s -- %s -- %s -- %s`, k, k1, fn, v2))
				}
			}
		}
	}

	for k, v := range x.abcValues {
		for k1, v1 := range v {
			abcValues = append(abcValues, fmt.Sprintf(`%s -- %s -- %s`, k, k1, v1.String()))
		}
	}

	return map[string]interface{}{
		"nodes":      nodes,
		"values":     values,
		"abc_Nodes":  abcNodes,
		"abc_Values": abcValues,
	}
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
	if x.providers[k] == nil {
		x.providers[k] = map[group][]*node{name: {nd}}
	} else {
		x.providers[k][name] = append(x.providers[k][name], nd)
	}
}

func (x *dix) setAbcProvider(k key, name group, nd *node) {
	if x.abcProviders[k] == nil {
		x.abcProviders[k] = map[group][]*node{name: {nd}}
	} else {
		x.abcProviders[k][name] = append(x.abcProviders[k][name], nd)
	}
}

func newDix(opts ...dix_opts.Option) *dix {
	c := &dix{
		providers:    make(map[key]map[group][]*node),
		abcProviders: make(map[key]map[group][]*node),
		values:       make(map[key]map[group]value),
		abcValues:    make(map[key]map[group]key),
		opts: dix_opts.Options{
			Rand:       rand.New(rand.NewSource(time.Now().UnixNano())),
			NilAllowed: false,
		},
	}

	xerror.Exit(c.init(opts...))
	return c
}

func (x *dix) Dix(data ...interface{}) error { return x.dix(data...) }
func (x *dix) Invoke(data interface{}, namespaces ...string) error {
	return x.invoke(data, namespaces...)
}
func (x *dix) Init(opts ...dix_opts.Option) error { return x.init(opts...) }
func (x *dix) Graph() string                      { return x.graph() }
func (x *dix) Json() map[string]interface{}       { return x.json() }
