package dix

import (
	"reflect"
	
	"github.com/pubgo/dix/dixinternal"
)

const (
	InjectMethodPrefix = dixinternal.InjectMethodPrefix
)

type (
	Option  = dixinternal.Option
	Options = dixinternal.Options
	Dix     = dixinternal.Dix
	Graph   = dixinternal.Graph
)

func WithValuesNull() Option {
	return dixinternal.WithValuesNull()
}

func New(opts ...Option) *Dix {
	return dixinternal.New(opts...)
}

func Inject[T any](di *Dix, data T, opts ...Option) T {
	vp := reflect.ValueOf(data)
	if vp.Kind() == reflect.Struct {
		_ = di.Inject(&data, opts...)
	} else {
		_ = di.Inject(data, opts...)
	}

	return data
}

func Provide(di *Dix, data any) {
	di.Provide(data)
}
