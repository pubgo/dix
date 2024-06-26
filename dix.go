package dix

import (
	"reflect"

	"github.com/pubgo/dix/internal/dix_inter"
)

const (
	InjectMethodPrefix = dix_inter.InjectMethodPrefix
)

type (
	Option  = dix_inter.Option
	Options = dix_inter.Options
	Dix     = dix_inter.Dix
	Graph   = dix_inter.Graph
)

func WithValuesNull() Option {
	return dix_inter.WithValuesNull()
}

func New(opts ...Option) *Dix {
	return dix_inter.New(opts...)
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
