package dix_internal

import (
	"reflect"

	"github.com/pubgo/funk/assert"
	"github.com/pubgo/funk/errors"
)

// New Dix new
func New(opts ...Option) *Dix {
	return newDix(opts...)
}

func (x *Dix) Provide(param any) {
	x.provide(param)
}

func (x *Dix) Inject(param any, opts ...Option) any {
	if dep, ok := x.isCycle(); ok {
		logger.Error().
			Str("cycle_path", dep).
			Str("component", reflect.TypeOf(param).String()).
			Msg("dependency cycle detected")
		assert.Must(errors.New("circular dependency: " + dep))
	}

	assert.Must(x.inject(param, opts...))
	return param
}

func (x *Dix) Graph() *Graph {
	return &Graph{
		Objects:   x.objectGraph(),
		Providers: x.providerGraph(),
	}
}
