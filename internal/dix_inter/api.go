package dix_inter

import (
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
	typeMap[typ] = struct{}{}
	for _, t := range types {
		var tt = reflect.TypeOf(t)
		if tt.Elem().Kind() == reflect.Interface {
			tt = tt.Elem()
		}
		typeMap[tt] = struct{}{}
	}

	objects := make(map[outputType]map[group][]reflect.Value)
	for tt := range typeMap {
		for k, oo := range handleOutput(tt, val) {
			if objects[k] == nil {
				objects[k] = make(map[group][]reflect.Value)
			}

			for g, o := range oo {
				objects[k][g] = append(objects[k][g], o...)
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
