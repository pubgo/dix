package dix

import (
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
