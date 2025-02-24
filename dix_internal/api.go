package dix_internal

import (
	"os"
	"runtime/debug"
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
		debug.PrintStack()
		logger.Fatal().Str("cycle", dep).Msg("provider circular dependency")
		os.Exit(1)
	}
	return x.inject(param, opts...)
}

func (x *Dix) Graph() *Graph {
	return &Graph{
		Objects:   x.objectGraph(),
		Providers: x.providerGraph(),
	}
}
