package dix_inter

// New Dix new
func New(opts ...Option) *Dix {
	di := newDix(opts...)
	di.provide(func() *Dix { return di })
	return di
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
