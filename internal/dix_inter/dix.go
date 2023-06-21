package dix_inter

import (
	"fmt"
	"reflect"
	"strings"

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
		providers: make(map[outputType][]*node),
		objects:   make(map[outputType]map[group][]value),
	}

	return c
}

type Dix struct {
	option    Options
	providers map[outputType][]*node
	objects   map[outputType]map[group][]value
}

func (x *Dix) Option() Options {
	return x.option
}

func (x *Dix) handleOutput(outType outputType, out reflect.Value) map[outputType]map[group][]value {
	var rr = make(map[outputType]map[group][]value)
	if !out.IsValid() || out.IsZero() {
		return rr
	}

	switch out.Kind() {
	case reflect.Map:
		if rr[outType] == nil {
			rr[outType] = make(map[group][]value)
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

			rr[outType][mapK] = append(rr[nil][mapK], val)
		}
	case reflect.Slice:
		if rr[outType] == nil {
			rr[outType] = make(map[group][]value)
		}

		for i := 0; i < out.Len(); i++ {
			var val = out.Index(i)
			if !val.IsValid() || val.IsNil() {
				continue
			}

			rr[outType][defaultKey] = append(rr[nil][defaultKey], val)
		}
	case reflect.Struct:
		for i := 0; i < out.NumField(); i++ {
			for typ, vv := range x.handleOutput(out.Field(i).Type(), out.Field(i)) {
				if rr[typ] == nil {
					rr[typ] = vv
				}

				for g, v := range vv {
					rr[typ][g] = append(rr[typ][g], v...)
				}
			}
		}
	default:
		if rr[outType] == nil {
			rr[outType] = make(map[group][]value)
		}

		if out.IsValid() && !out.IsNil() {
			rr[outType][defaultKey] = []value{out}
		}
	}
	return rr
}

func (x *Dix) evalProvider(typ outputType, opt Options) map[group][]value {
	switch typ.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Func:
	default:
		assert.Must(errors.Err{
			Msg:    "provider type kind error, the supported type kinds are <ptr,interface,func>",
			Detail: fmt.Sprintf("type=%s kind=%s", typ, typ.Kind()),
		})
	}

	if len(x.providers[typ]) == 0 {
		log.Warn().
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

	log.Debug().
		Str("type", typ.String()).
		Str("kind", typ.Kind().String()).
		Int("providers", len(x.providers[typ])).
		Msg("eval type value")
	objects := make(map[outputType]map[group][]value)
	for _, n := range x.providers[typ] {
		var input []reflect.Value
		for _, in := range n.input {
			var val = x.getValue(in.typ, opt, in.isMap, in.isList)
			input = append(input, val)
		}

		for k, oo := range x.handleOutput(typ, n.call(input)[0]) {
			if n.output.isMap {
				if _, ok := objects[k]; ok {
					log.Info().
						Str("type", typ.String()).
						Str("key", k.String()).
						Msg("type value exists")
				}
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
			log.Panic().
				Any("options", opt).
				Any("values", valMap).
				Str("type", typ.String()).
				Str("type-kind", typ.Kind().String()).
				Msg("provider value not found")
		}

		return makeMap(typ, valMap, isList)
	case isList:
		valMap := x.evalProvider(typ, opt)
		if !opt.AllowValuesNull && len(valMap[defaultKey]) == 0 {
			var err = &errors.Err{
				Msg:    "provider value not found",
				Detail: fmt.Sprintf("type=%s kind=%s allValues=%v", typ, typ.Kind(), valMap),
			}

			log.Panic().Err(err).
				Any("options", opt).
				Any("values", valMap[defaultKey]).
				Str("type", typ.String()).
				Str("type-kind", typ.Kind().String()).
				Msg(err.Msg)
		}

		return makeList(typ, valMap[defaultKey])
	case typ.Kind() == reflect.Struct:
		var v = reflect.New(typ)
		x.injectStruct(v.Elem(), opt)
		return v.Elem()
	default:
		valMap := x.evalProvider(typ, opt)
		if valList, ok := valMap[defaultKey]; !ok || len(valList) == 0 {
			log.Panic().
				Any("options", opt).
				Any("values", valMap[defaultKey]).
				Str("type", typ.String()).
				Str("type-kind", typ.Kind().String()).
				Msg("provider value not found")
		} else {
			// 最后一个value
			var val = valList[len(valList)-1]
			if val.IsZero() {
				var err = &errors.Err{
					Msg:    "provider value is nil",
					Detail: fmt.Sprintf("type=%s kind=%s value=%v", typ, typ.Kind(), val.Interface()),
				}

				log.Panic().Err(err).
					Any("options", opt).
					Any("values", valList).
					Str("type", typ.String()).
					Str("type-kind", typ.Kind().String()).
					Msg(err.Msg)
			}
			return val
		}
	}

	return reflect.Value{}
}

func (x *Dix) injectFunc(vp reflect.Value, opt Options) {
	var inTypes []*inType
	for i := 0; i < vp.Type().NumIn(); i++ {
		switch inTyp := vp.Type().In(i); inTyp.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Func, reflect.Struct:
			inTypes = append(inTypes, &inType{typ: inTyp})
		case reflect.Map:
			var isList = inTyp.Elem().Kind() == reflect.Slice
			typ := inTyp.Elem()
			if isList {
				typ = typ.Elem()
			}
			inTypes = append(inTypes, &inType{typ: typ, isMap: true, isList: isList})
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
			var isList = field.Type.Elem().Kind() == reflect.Slice
			typ := field.Type.Elem()
			if isList {
				typ = typ.Elem()
			}
			vp.Field(i).Set(x.getValue(typ, opt, true, isList))
		case reflect.Slice:
			vp.Field(i).Set(x.getValue(field.Type.Elem(), opt, false, true))
		}
	}
}

func (x *Dix) inject(param interface{}, opts ...Option) interface{} {
	defer recovery.Raise(func(err error) error {
		return errors.WrapKV(err, "param", fmt.Sprintf("%#v", param))
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
		assert.If(vp.Type().NumOut() != 0, "func output num should be zero")
		assert.If(vp.Type().NumIn() == 0, "func input num should not be zero")
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

func (x *Dix) handleStructProvide(fnVal reflect.Value, out reflect.Type, in []*inType) {
	n := &node{fn: fnVal, input: in}
	switch outTyp := out; outTyp.Kind() {
	case reflect.Slice:
		n.output = &outType{isList: true, typ: outTyp.Elem()}
		x.providers[n.output.typ] = append(x.providers[n.output.typ], n)
	case reflect.Map:
		n.output = &outType{isMap: true, typ: outTyp.Elem()}
		x.providers[n.output.typ] = append(x.providers[n.output.typ], n)
	case reflect.Ptr, reflect.Interface, reflect.Func:
		n.output = &outType{isList: true, typ: outTyp}
		x.providers[n.output.typ] = append(x.providers[n.output.typ], n)
	case reflect.Struct:
		log.Debug().Str("name", outTyp.Name()).Msg("struct info")
		for i := 0; i < outTyp.NumField(); i++ {
			x.handleStructProvide(fnVal, outTyp.Field(i).Type, in)
		}
	default:
		panic(&errors.Err{
			Msg:    "incorrect output type",
			Detail: fmt.Sprintf("ouTyp=%s kind=%s", outTyp, outTyp.Kind()),
		})
	}
}

func (x *Dix) provide(param interface{}) {
	defer recovery.Raise(func(err error) error {
		return errors.WrapKV(err, "param", fmt.Sprintf("%#v", param))
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

	var input []*inType
	for i := 0; i < typ.NumIn(); i++ {
		switch inTye := typ.In(i); inTye.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Func, reflect.Struct:
			input = append(input, &inType{typ: inTye})
		case reflect.Map:
			var isList = inTye.Elem().Kind() == reflect.Slice
			input = append(input, &inType{typ: inTye.Elem(), isMap: true, isList: isList})
		case reflect.Slice:
			input = append(input, &inType{typ: inTye.Elem(), isList: true})
		default:
			panic(&errors.Err{
				Msg:    "incorrect input type",
				Detail: fmt.Sprintf("inTyp=%s kind=%s", inTye, inTye.Kind()),
			})
		}
	}

	x.handleStructProvide(fnVal, typ.Out(0), input)

	dep, ok := x.isCycle()
	assert.Err(ok, &errors.Err{
		Msg:    "provider circular dependency",
		Detail: dep,
	})
	return
}
