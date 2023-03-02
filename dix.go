package dix

import (
	"github.com/pubgo/dix/internal/dix_inter"
)

type (
	Option  = dix_inter.Option
	Options = dix_inter.Options
	Dix     = dix_inter.Dix
	Graph   = dix_inter.Graph
)

var (
	WithValuesNull = dix_inter.WithValuesNull
	New            = dix_inter.New
)

const (
	InjectMethodPrefix = dix_inter.InjectMethodPrefix
)
