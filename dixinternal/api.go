package dixinternal

import (
	"errors"
	"reflect"
)

// New Dix new
func New(opts ...Option) *Dix {
	return newDix(opts...)
}

func (dix *Dix) Provide(param any) {
	dix.provide(param)
}

func (dix *Dix) Inject(param any, opts ...Option) any {
	if dep, ok := dix.isCycle(); ok {
		logger.Error("dependency cycle detected", "cycle_path", dep, "component", reflect.TypeOf(param).String())
		panic(errors.New("circular dependency: " + dep))
	}

	if err := dix.inject(param, opts...); err != nil {
		panic(err)
	}
	return param
}

// Graph generates dependency graphs with default options
func (dix *Dix) Graph() *Graph {
	return &Graph{
		Objects:       dix.objectGraph(),
		Providers:     dix.providerGraph(),
		ProviderTypes: dix.providerGraphTypes(),
	}
}

// GraphWithOptions generates dependency graphs with custom options
func (dix *Dix) GraphWithOptions(opts *GraphOptions) *Graph {
	return &Graph{
		Objects:       dix.objectGraph(),
		Providers:     dix.providerGraphWithOptions(opts),
		ProviderTypes: dix.providerGraphTypesWithOptions(opts),
	}
}
