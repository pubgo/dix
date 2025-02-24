package dix_internal

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/pubgo/funk/assert"
	"github.com/pubgo/funk/errors"
	"github.com/pubgo/funk/log"
	"github.com/pubgo/funk/recovery"
	"github.com/pubgo/funk/stack"
)

func newDix(opts ...Option) *Dix {
	option := Options{AllowValuesNull: true}
	defer recovery.Raise(func(err error) error {
		return errors.WrapKV(err, "options", option)
	})

	for i := range opts {
		opts[i](&option)
	}

	option.Check()

	c := &Dix{
		option:      option,
		providers:   make(map[outputType][]*node),
		objects:     make(map[outputType]map[group][]value),
		initializer: map[reflect.Value]bool{},
	}

	c.provide(func() *Dix { return c })

	return c
}

type Dix struct {
	option      Options
	providers   map[outputType][]*node
	objects     map[outputType]map[group][]value
	initializer map[reflect.Value]bool
}

func (x *Dix) Option() Options {
	return x.option
}

func (x *Dix) getOutputTypeValues(outTyp outputType, opt Options) map[group][]value {
	switch outTyp.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Func:
	default:
		assert.Must(errors.Err{
			Msg:    "provider type kind error, the supported type kinds are <ptr,interface,func>",
			Detail: fmt.Sprintf("type=%s kind=%s", outTyp, outTyp.Kind()),
		})
	}

	if len(x.providers[outTyp]) == 0 {
		logger.Warn().
			Str("type", outTyp.String()).
			Str("kind", outTyp.Kind().String()).
			Msg("provider not found, please check whether the provider imports or type error")
	}

	if x.objects[outTyp] == nil {
		x.objects[outTyp] = make(map[group][]value)
	}

	for _, n := range x.providers[outTyp] {
		if x.initializer[n.fn] {
			continue
		}

		var input []reflect.Value
		for _, in := range n.inputList {
			val := x.getValue(in.typ, opt, in.isMap, in.isList, outTyp)
			input = append(input, val)
		}

		var now = time.Now()
		var fnStack = stack.CallerWithFunc(n.fn)
		fnCall := n.call(input)
		logger.Debug().
			Str("cost", time.Since(now).String()).
			Str("provider", fnStack.String()).
			Msgf("eval provider func %s.%s", filepath.Base(fnStack.Pkg), fnStack.Name)

		x.initializer[n.fn] = true

		objects := make(map[outputType]map[group][]value)
		for outT, groupValue := range handleOutput(outTyp, fnCall[0]) {
			if n.output.isMap {
				if _, ok := objects[outT]; ok {
					logger.Info().
						Str("type", outTyp.String()).
						Str("key", outT.String()).
						Msg("type value exists")
				}
			}

			if objects[outT] == nil {
				objects[outT] = make(map[group][]value)
			}

			for g, o := range groupValue {
				objects[outT][g] = append(objects[outT][g], o...)
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
	}

	return x.objects[outTyp]
}

func (x *Dix) getProviderStack(typ reflect.Type) []string {
	var stacks []string
	for _, n := range x.providers[typ] {
		stacks = append(stacks, stack.CallerWithFunc(n.fn).String())
	}
	return stacks
}

func (x *Dix) getValue(typ reflect.Type, opt Options, isMap, isList bool, parents ...reflect.Type) reflect.Value {
	if typ.Kind() == reflect.Struct {
		v := reflect.New(typ)
		x.injectStruct(v.Elem(), opt)
		return v.Elem()
	}

	valMap := x.getOutputTypeValues(typ, opt)
	switch {
	case isMap:
		if !opt.AllowValuesNull && len(valMap) == 0 {
			logger.Panic().
				Any("options", opt).
				Str("type", typ.String()).
				Any("providers", x.getProviderStack(typ)).
				Any("parents", fmt.Sprintf("%q", parents)).
				Str("type-kind", typ.Kind().String()).
				Msg("provider value not found")
		}

		return makeMap(typ, valMap, isList)
	case isList:
		if !opt.AllowValuesNull && len(valMap[defaultKey]) == 0 {
			err := &errors.Err{
				Msg:    "provider value not found",
				Detail: fmt.Sprintf("type=%s kind=%s allValues=%v", typ, typ.Kind(), valMap),
			}

			logger.Panic().Err(err).
				Any("options", opt).
				Any("values", valMap[defaultKey]).
				Any("providers", x.getProviderStack(typ)).
				Any("parents", fmt.Sprintf("%q", parents)).
				Str("type", typ.String()).
				Str("type-kind", typ.Kind().String()).
				Msg(err.Msg)
		}

		return makeList(typ, valMap[defaultKey])
	default:
		if valList, ok := valMap[defaultKey]; !ok || len(valList) == 0 {
			logger.Panic().
				Any("options", opt).
				Any("values", valMap[defaultKey]).
				Str("type", typ.String()).
				Any("providers", x.getProviderStack(typ)).
				Any("parents", fmt.Sprintf("%q", parents)).
				Str("type-kind", typ.Kind().String()).
				Msg("provider value not found")
		} else {
			// 最后一个value
			val := valList[len(valList)-1]
			if val.IsZero() {
				err := &errors.Err{
					Msg:    "provider value is nil",
					Detail: fmt.Sprintf("type=%s kind=%s value=%v", typ, typ.Kind(), val.Interface()),
				}

				logger.Panic().Err(err).
					Any("options", opt).
					Any("values", valList).
					Any("providers", x.getProviderStack(typ)).
					Any("parents", fmt.Sprintf("%q", parents)).
					Str("type", typ.String()).
					Str("type-kind", typ.Kind().String()).
					Msg(err.Msg)
			}
			return val
		}
	}

	panic("unknown type")
}

func (x *Dix) injectFunc(vp reflect.Value, opt Options) {
	var inTypes []*inType
	for i := 0; i < vp.Type().NumIn(); i++ {
		switch inTyp := vp.Type().In(i); inTyp.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Func, reflect.Struct:
			inTypes = append(inTypes, &inType{typ: inTyp})
		case reflect.Map:
			isList := inTyp.Elem().Kind() == reflect.Slice
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
		input = append(input, x.getValue(in.typ, opt, in.isMap, in.isList, vp.Type()))
	}
	vp.Call(input)
}

func (x *Dix) injectStruct(vp reflect.Value, opt Options) {
	tp := vp.Type()
	for i := 0; i < tp.NumField(); i++ {
		field := tp.Field(i)
		if !vp.Field(i).CanSet() && field.Type.Kind() != reflect.Struct {
			continue
		}

		switch field.Type.Kind() {
		case reflect.Struct:
			x.injectStruct(vp.Field(i), opt)
		case reflect.Interface, reflect.Ptr, reflect.Func:
			vp.Field(i).Set(x.getValue(field.Type, opt, false, false, vp.Type()))
		case reflect.Map:
			isList := field.Type.Elem().Kind() == reflect.Slice
			typ := field.Type.Elem()
			if isList {
				typ = typ.Elem()
			}
			vp.Field(i).Set(x.getValue(typ, opt, true, isList, vp.Type()))
		case reflect.Slice:
			vp.Field(i).Set(x.getValue(field.Type.Elem(), opt, false, true, vp.Type()))
		default:
			panic(&errors.Err{
				Msg:    "incorrect input type",
				Detail: fmt.Sprintf("inTyp=%s kind=%s", field.Type, field.Type.Kind()),
			})
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
		name := vp.Type().Method(i).Name
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

func (x *Dix) handleProvide(fnVal reflect.Value, out reflect.Type, in []*inType) {
	n := &node{fn: fnVal, inputList: in}
	switch outTyp := out; outTyp.Kind() {
	case reflect.Slice:
		n.output = &outType{isList: true, typ: outTyp.Elem()}
		x.providers[n.output.typ] = append(x.providers[n.output.typ], n)
	case reflect.Map:
		n.output = &outType{isMap: true, typ: outTyp.Elem()}
		if n.output.typ.Kind() == reflect.Slice {
			n.output.isList = true
			n.output.typ = n.output.typ.Elem()
		}
		x.providers[n.output.typ] = append(x.providers[n.output.typ], n)
	case reflect.Ptr, reflect.Interface, reflect.Func:
		n.output = &outType{typ: outTyp}
		x.providers[n.output.typ] = append(x.providers[n.output.typ], n)
	case reflect.Struct:
		for i := 0; i < outTyp.NumField(); i++ {
			x.handleProvide(fnVal, outTyp.Field(i).Type, in)
		}
	default:
		log.Error().Msgf("incorrect output type, ouTyp=%s kind=%s fnVal=%s", outTyp, outTyp.Kind(), fnVal.String())
	}
}

func (x *Dix) getProvideAllInputs(typ reflect.Type) []*inType {
	var input []*inType
	switch inTye := typ; inTye.Kind() {
	case reflect.Interface, reflect.Ptr, reflect.Func:
		input = append(input, &inType{typ: inTye})
	case reflect.Struct:
		for j := 0; j < inTye.NumField(); j++ {
			input = append(input, x.getProvideAllInputs(inTye.Field(j).Type)...)
		}
	case reflect.Map:
		tt := &inType{typ: inTye.Elem(), isMap: true, isList: inTye.Elem().Kind() == reflect.Slice}
		if tt.isList {
			tt.typ = tt.typ.Elem()
		}
		input = append(input, tt)
	case reflect.Slice:
		input = append(input, &inType{typ: inTye.Elem(), isList: true})
	default:
		log.Error().Msgf("incorrect input type, inTyp=%s kind=%s", inTye, inTye.Kind())
	}
	return input
}

func (x *Dix) getProvideInput(typ reflect.Type) []*inType {
	var input []*inType
	switch inTye := typ; inTye.Kind() {
	case reflect.Interface, reflect.Ptr, reflect.Func, reflect.Struct:
		input = append(input, &inType{typ: inTye})
	case reflect.Map:
		tt := &inType{typ: inTye.Elem(), isMap: true, isList: inTye.Elem().Kind() == reflect.Slice}
		if tt.isList {
			tt.typ = tt.typ.Elem()
		}
		input = append(input, tt)
	case reflect.Slice:
		input = append(input, &inType{typ: inTye.Elem(), isList: true})
	default:
		log.Error().Msgf("incorrect input type, inTyp=%s kind=%s", inTye, inTye.Kind())
	}
	return input
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
		input = append(input, x.getProvideInput(typ.In(i))...)
	}

	// The return value can only have one
	// TODO Add the second parameter, support for error
	x.handleProvide(fnVal, typ.Out(0), input)
}
