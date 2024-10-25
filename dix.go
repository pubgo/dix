package dix

import (
	"reflect"

	"github.com/pubgo/dix/dix_internal"
)

const (
	InjectMethodPrefix = dix_internal.InjectMethodPrefix
)

type (
	Option  = dix_internal.Option
	Options = dix_internal.Options
	Dix     = dix_internal.Dix
	Graph   = dix_internal.Graph
)

func WithValuesNull() Option {
	return dix_internal.WithValuesNull()
}

func New(opts ...Option) *Dix {
	return dix_internal.New(opts...)
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
