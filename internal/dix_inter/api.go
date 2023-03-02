package dix_inter

import (
	"github.com/pubgo/funk/assert"
)

// New Dix new
func New(opts ...Option) *Dix { return newDix(opts...) }
func (x *Dix) Provide(param any) error {
	return x.provide(param)
}

func (x *Dix) Inject(param any, opts ...Option) any {
	return assert.Must1(x.inject(param, opts...))
}

func (x *Dix) Graph() *Graph {
	return &Graph{
		Objects:   x.objectGraph(),
		Providers: x.providerGraph(),
	}
}
