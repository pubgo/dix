package dix_internal

// New Dix new
func New(opts ...Option) *Dix {
	return newDix(opts...)
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
