package dixinternal

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
		providers:   make(map[outputType][]*providerFn),
		objects:     make(map[outputType]map[group][]value),
		initializer: map[reflect.Value]bool{},
	}

	c.provide(func() *Dix { return c })

	return c
}

type Dix struct {
	option      Options
	providers   map[outputType][]*providerFn
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

		logger.Debug().
			Str("provider", fnStack.String()).
			Msgf("start eval provider func %s.%s", filepath.Base(fnStack.Pkg), fnStack.Name)

		fnCall := n.call(input)
		x.initializer[n.fn] = true
		logger.Debug().
			Str("cost", time.Since(now).String()).
			Str("provider", fnStack.String()).
			Msgf("eval provider ok, func %s.%s", filepath.Base(fnStack.Pkg), fnStack.Name)

		if n.hasError && len(fnCall) > 1 && !fnCall[1].IsNil() {
			if err, ok := fnCall[1].Interface().(error); ok && err != nil {
				panic(&errors.Err{
					Msg:    fmt.Sprintf("failed to do provider, err=%s", err.Error()),
					Detail: fmt.Sprintf("func=%s", fnStack.String()),
				})
			}
		}

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
	assert.If(vp.Type().NumOut() > 1, "func output num should <=1")
	assert.If(vp.Type().NumIn() == 0, "func input num should not be zero")
	var hasErrorReturn bool
	if vp.Type().NumOut() == 1 {
		// 如果有一个返回值，必须是 error 类型
		errorType := vp.Type().Out(0)
		if !errorType.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			panic(&errors.Err{
				Msg:    "injectable function can only return error type",
				Detail: fmt.Sprintf("return_type=%s", errorType.String()),
			})
		}
		hasErrorReturn = true
	}

	var inTypes []*providerInputType
	for i := 0; i < vp.Type().NumIn(); i++ {
		switch inTyp := vp.Type().In(i); inTyp.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Func, reflect.Struct:
			inTypes = append(inTypes, &providerInputType{typ: inTyp, isStruct: inTyp.Kind() == reflect.Struct})
		case reflect.Map:
			isList := inTyp.Elem().Kind() == reflect.Slice
			typ := inTyp.Elem()
			if isList {
				typ = typ.Elem()
			}
			inTypes = append(inTypes, &providerInputType{typ: typ, isMap: true, isList: isList})
		case reflect.Slice:
			inTypes = append(inTypes, &providerInputType{typ: inTyp.Elem(), isList: true})
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

	results := vp.Call(input)
	// 如果函数有 error 返回值，检查并处理
	if hasErrorReturn && len(results) > 0 && !results[0].IsNil() {
		errorValue := results[0]
		if funcErr, ok := errorValue.Interface().(error); ok {
			panic(errors.Wrapf(funcErr, "injected function returned error"))
		}
	}
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

func (x *Dix) inject(param interface{}, opts ...Option) (gErr error) {
	defer recovery.Err(&gErr, func(err error) error {
		return errors.WrapKV(err, "param", fmt.Sprintf("%#v", param))
	})

	if param == nil {
		return errors.New("nil injection parameter")
	}

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
	return nil
}

func (x *Dix) handleProvide(fnVal reflect.Value, out reflect.Type, in []*providerInputType) {
	hasError := false
	if fnVal.Type().NumOut() == 2 {
		errorType := fnVal.Type().Out(1)
		if errorType.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			hasError = true
		} else {
			panic(&errors.Err{
				Msg:    "second return value must be error type",
				Detail: fmt.Sprintf("actual_type=%s, fn=%v", errorType.String(), fnVal.String()),
			})
		}
	}

	n := &providerFn{fn: fnVal, inputList: in, hasError: hasError}
	switch outTyp := out; outTyp.Kind() {
	case reflect.Slice:
		n.output = &providerOutputType{isList: true, typ: outTyp.Elem()}
		x.providers[n.output.typ] = append(x.providers[n.output.typ], n)
	case reflect.Map:
		n.output = &providerOutputType{isMap: true, typ: outTyp.Elem()}
		if n.output.typ.Kind() == reflect.Slice {
			n.output.isList = true
			n.output.typ = n.output.typ.Elem()
		}
		x.providers[n.output.typ] = append(x.providers[n.output.typ], n)
	case reflect.Ptr, reflect.Interface, reflect.Func:
		n.output = &providerOutputType{typ: outTyp}
		x.providers[n.output.typ] = append(x.providers[n.output.typ], n)
	case reflect.Struct:
		// n.output.isStruct = true
		for i := 0; i < outTyp.NumField(); i++ {
			if !outTyp.Field(i).IsExported() {
				continue
			}

			typ := outTyp.Field(i).Type
			if !isSupportedType(typ) {
				continue
			}

			x.handleProvide(fnVal, typ, in)
		}
	default:
		log.Error().Msgf("incorrect output type, ouTyp=%s kind=%s fnVal=%s", outTyp, outTyp.Kind(), fnVal.String())
	}
}

func (x *Dix) getProvideInput(typ reflect.Type) []*providerInputType {
	var input []*providerInputType
	switch inTye := typ; inTye.Kind() {
	case reflect.Interface, reflect.Ptr, reflect.Func, reflect.Struct:
		input = append(input, &providerInputType{typ: inTye})
	case reflect.Map:
		tt := &providerInputType{typ: inTye.Elem(), isMap: true, isList: inTye.Elem().Kind() == reflect.Slice}
		if tt.isList {
			tt.typ = tt.typ.Elem()
		}
		input = append(input, tt)
	case reflect.Slice:
		input = append(input, &providerInputType{typ: inTye.Elem(), isList: true})
	default:
		log.Error().Msgf("incorrect input type, inTyp=%s kind=%s", inTye, inTye.Kind())
	}
	return input
}

// Provide registers the constructor with the container.
// The constructor must be a function that returns at least one value (or an error).
// Arguments of the constructor are treated as dependencies,
// and return values are treated as results that can be injected elsewhere.
// Provide panics if the constructor is not a function or does not have the required signature.
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
	assert.If(typ.NumOut() > 2, "the func of provider output num should >= two")

	var input []*providerInputType
	for i := 0; i < typ.NumIn(); i++ {
		input = append(input, x.getProvideInput(typ.In(i))...)
	}

	// The return value can only have one
	// TODO Add the second parameter, support for error
	x.handleProvide(fnVal, typ.Out(0), input)
}
