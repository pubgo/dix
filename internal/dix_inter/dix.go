package dix_inter

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/alecthomas/repr"
	"github.com/pubgo/funk/assert"
	"github.com/pubgo/funk/errors"
	"github.com/pubgo/funk/log"
	"github.com/pubgo/funk/recovery"
)

func newDix(opts ...Option) *Dix {
	var option = Options{}
	defer recovery.Raise(func(err error) error {
		return errors.WrapKV(err, "options", option)
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

type Dix struct {
	option    Options
	providers map[key][]*node
	objects   map[key]map[group][]value
}

func (x *Dix) Option() Options {
	return x.option
}

func (x *Dix) handleOutput(output []reflect.Value) map[key]map[group][]value {
	var out = output[0]
	var rr = make(map[key]map[group][]value)
	switch out.Kind() {
	case reflect.Map:
		if rr[nil] == nil {
			rr[nil] = make(map[group][]value)
		}

		for _, k := range out.MapKeys() {
			var mapK = strings.TrimSpace(k.String())
			if mapK == "" {
				mapK = defaultKey
			}

			var val = out.MapIndex(k)
			if !val.IsValid() || val.IsNil() {
				continue
			}

			rr[nil][mapK] = append(rr[nil][mapK], val)
		}
	case reflect.Slice:
		if rr[nil] == nil {
			rr[nil] = make(map[group][]value)
		}

		for i := 0; i < out.Len(); i++ {
			var val = out.Index(i)
			if !val.IsValid() || val.IsNil() {
				continue
			}

			rr[nil][defaultKey] = append(rr[nil][defaultKey], val)
		}
	case reflect.Struct:
		for i := 0; i < out.NumField(); i++ {
			f := out.Field(i)
			if rr[f.Type()] == nil {
				rr[f.Type()] = make(map[group][]value)
			}

			rr[f.Type()][defaultKey] = append(rr[f.Type()][defaultKey], f)
		}
	default:
		if rr[nil] == nil {
			rr[nil] = make(map[group][]value)
		}

		if out.IsValid() && !out.IsNil() {
			rr[nil][defaultKey] = []value{out}
		}
	}
	return rr
}

func (x *Dix) evalProvider(typ key, opt Options) map[group][]value {
	switch typ.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Func:
	default:
		assert.Must(errors.SimpleErr(func(err *errors.Err) {
			err.Msg = "provider type kind error, the supported type kinds are <ptr,interface,func>"
			err.Detail = fmt.Sprintf("type=%s kind=%s", typ, typ.Kind())
		}))
	}

	if len(x.providers[typ]) == 0 {
		log.Info().
			Str("type", typ.String()).
			Str("kind", typ.Kind().String()).
			Msg("provider not found, please check whether the provider imports or type error")
		return make(map[group][]value)
	}

	if x.objects[typ] == nil {
		x.objects[typ] = make(map[group][]value)
	}

	if val := x.objects[typ]; len(val) != 0 {
		return val
	}

	objects := make(map[key]map[group][]value)
	for _, n := range x.providers[typ] {
		var input []reflect.Value
		for _, in := range n.input {
			var val = x.getValue(in.typ, opt, in.isMap, in.isList)
			input = append(input, val)
		}

		for k, oo := range x.handleOutput(n.call(input)) {
			if n.output.isMap {
				if _, ok := objects[k]; ok {
					log.Info().
						Str("type", typ.String()).
						Str("key", k.String()).
						Msg("type value exists")
				}
			}

			if k == nil {
				k = typ
			}

			if objects[k] == nil {
				objects[k] = make(map[group][]value)
			}

			for g, o := range oo {
				objects[k][g] = append(objects[k][g], o...)
			}
		}
	}

	for a, b := range objects {
		if x.objects[a] == nil {
			x.objects[a] = make(map[group][]value)
		}

		for c, d := range b {
			x.objects[a][c] = append(x.objects[a][c], d...)
		}
	}

	return x.objects[typ]
}

func (x *Dix) getValue(typ reflect.Type, opt Options, isMap bool, isList bool) reflect.Value {
	switch {
	case isMap:
		valMap := x.evalProvider(typ, opt)
		if !opt.AllowValuesNull && len(valMap) == 0 {
			panic(&errors.Err{
				Msg:    "provider default value not found",
				Detail: fmt.Sprintf("type=%s kind=%s allValues=%v", typ, typ.Kind(), valMap),
			})
		}

		return makeMap(typ, valMap)
	case isList:
		valMap := x.evalProvider(typ, opt)
		if !opt.AllowValuesNull && len(valMap[defaultKey]) == 0 {
			panic(&errors.Err{
				Msg:    "provider default value not found",
				Detail: fmt.Sprintf("type=%s kind=%s allValues=%v", typ, typ.Kind(), valMap),
			})
		}

		return makeList(typ, valMap[defaultKey])
	case typ.Kind() == reflect.Struct:
		var v = reflect.New(typ)
		x.injectStruct(v.Elem(), opt)
		return v.Elem()
	default:
		valMap := x.evalProvider(typ, opt)
		if valList, ok := valMap[defaultKey]; !ok || len(valList) == 0 {
			panic(&errors.Err{
				Msg:    "provider default value not found",
				Detail: fmt.Sprintf("type=%s kind=%s allValues=%v", typ, typ.Kind(), valMap),
			})
		} else {
			var val = valList[len(valList)-1]
			if val.IsZero() {
				panic(&errors.Err{
					Msg:    "provider default value is nil",
					Detail: fmt.Sprintf("type=%s kind=%s value=%v", typ, typ.Kind(), val.Interface()),
				})
			}
			return val
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
			panic(&errors.Err{
				Msg:    "incorrect input type",
				Detail: fmt.Sprintf("inTyp=%s kind=%s", inTyp, inTyp.Kind()),
			})
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
	defer recovery.Raise(func(err error) error {
		return errors.WrapKV(err, "param", repr.String(param))
	})

	assert.If(param == nil, "param is null")

	var opt Options
	for i := range opts {
		opts[i](&opt)
	}
	opt = x.option.Merge(opt)

	vp := reflect.ValueOf(param)
	assert.Err(!vp.IsValid() || vp.IsNil(), &errors.Err{
		Msg: "param should not be invalid or nil",
	})

	if vp.Kind() == reflect.Func {
		assert.If(vp.Type().NumOut() != 0, "the func of provider output num should be zero")
		assert.If(vp.Type().NumIn() == 0, "the func of provider input num should not be zero")
		x.injectFunc(vp, opt)
		return nil
	}

	assert.Err(vp.Kind() != reflect.Ptr, &errors.Err{
		Msg: "param should be ptr type",
	})

	for i := 0; i < vp.NumMethod(); i++ {
		var name = vp.Type().Method(i).Name
		if !strings.HasPrefix(name, InjectMethodPrefix) {
			continue
		}

		x.injectFunc(vp.Method(i), opt)
	}

	for vp.Kind() == reflect.Ptr {
		vp = vp.Elem()
	}

	assert.Err(vp.Kind() != reflect.Struct, &errors.Err{
		Msg: "param raw type should be struct",
	})

	x.injectStruct(vp, opt)
	return param
}

func (x *Dix) provide(param interface{}) {
	defer recovery.Raise(func(err error) error {
		return errors.WrapKV(err, "param", repr.String(param))
	})

	assert.If(param == nil, "[param] is null")

	fnVal := reflect.ValueOf(param)
	assert.Err(!fnVal.IsValid() || fnVal.IsZero(), &errors.Err{
		Msg: "param should not be invalid or nil",
	})

	assert.Err(fnVal.Kind() != reflect.Func, &errors.Err{
		Msg: "param should be function type",
	})

	typ := fnVal.Type()
	assert.If(typ.IsVariadic(), "the func of provider variable parameters are not allowed")
	assert.If(typ.NumOut() == 0, "the func of provider output num should not be zero")

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
			panic(&errors.Err{
				Msg:    "incorrect input type",
				Detail: fmt.Sprintf("inTyp=%s kind=%s", inTye, inTye.Kind()),
			})
		}
	}

	switch outTyp := typ.Out(0); outTyp.Kind() {
	case reflect.Slice:
		n.output = &outType{isList: true, typ: outTyp.Elem()}
	case reflect.Map:
		n.output = &outType{isMap: true, typ: outTyp.Elem()}
	case reflect.Ptr, reflect.Interface, reflect.Func:
		n.output = &outType{isList: true, typ: outTyp}
	case reflect.Struct:
		var out = typ.Out(0)
		for i := 0; i < out.NumField(); i++ {
			nn := &node{fn: fnVal, input: n.input[:]}
			switch oo := out.Field(i); oo.Type.Kind() {
			case reflect.Slice:
				nn.output = &outType{isList: true, typ: oo.Type.Elem()}
			case reflect.Map:
				nn.output = &outType{isMap: true, typ: oo.Type.Elem()}
			case reflect.Ptr, reflect.Interface, reflect.Func:
				nn.output = &outType{isList: true, typ: oo.Type}
			}
			x.providers[nn.output.typ] = append(x.providers[nn.output.typ], nn)
		}
		return
	default:
		panic(&errors.Err{
			Msg:    "incorrect output type",
			Detail: fmt.Sprintf("ouTyp=%s kind=%s", outTyp, outTyp.Kind()),
		})
	}

	x.providers[n.output.typ] = append(x.providers[n.output.typ], n)

	dep, ok := x.isCycle()
	assert.Err(ok, &errors.Err{
		Msg:    "provider circular dependency",
		Detail: dep,
	})
	return
}
