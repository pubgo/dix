package dix_inter

import (
	"fmt"
	"github.com/pubgo/funk/errors"
	"reflect"

	"github.com/pubgo/funk/assert"
	"github.com/pubgo/funk/generic"
)

// New Dix new
func New(opts ...Option) *Dix {
	return newDix(opts...)
}

func (x *Dix) SetValue(value any, types ...any) {
	assert.If(generic.IsNil(value), "value shoule not be nil")

	var typ = reflect.TypeOf(value)
	var val = reflect.ValueOf(value)
	var typeMap = make(map[reflect.Type]struct{})
	switch inTyp := typ; inTyp.Kind() {
	case reflect.Interface, reflect.Ptr, reflect.Func, reflect.Struct:
		typeMap[inTyp] = struct{}{}
	case reflect.Map:
		var isList = inTyp.Elem().Kind() == reflect.Slice
		typ1 := inTyp.Elem()
		if isList {
			typ1 = typ1.Elem()
		}
		typeMap[typ1] = struct{}{}
	case reflect.Slice:
		typeMap[inTyp.Elem()] = struct{}{}
	default:
		panic(&errors.Err{
			Msg:    "incorrect input type",
			Detail: fmt.Sprintf("inTyp=%s kind=%s", inTyp, inTyp.Kind()),
		})
	}

	for _, t := range types {
		var tt = reflect.TypeOf(t)
		if tt.Elem().Kind() == reflect.Interface {
			tt = tt.Elem()
		}
		typeMap[tt] = struct{}{}
	}

	objects := make(map[outputType]map[group][]reflect.Value)
	for tt := range typeMap {
		for _, oo := range handleOutput(tt, val) {
			if objects[tt] == nil {
				objects[tt] = make(map[group][]reflect.Value)
			}

			for g, o := range oo {
				objects[tt][g] = append(objects[tt][g], o...)
			}
		}
	}

	for a, b := range objects {
		if x.objects[a] == nil {
			x.objects[a] = make(map[group][]reflect.Value)
		}

		for c, d := range b {
			x.objects[a][c] = append(x.objects[a][c], d...)
		}
	}
}

func (x *Dix) Provide(param any) {
	x.provide(param)
}

func (x *Dix) Inject(param any, opts ...Option) any {
	return x.inject(param, opts...)
}

func (x *Dix) Graph() *Graph {
	return &Graph{
		Objects:   x.objectGraph(),
		Providers: x.providerGraph(),
	}
}
