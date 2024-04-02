package dix_inter

// New Dix new
func New(opts ...Option) *Dix {
	return newDix(opts...)
}

func (x *Dix) SetValue(value any, types ...any) {
	x.setValue(value, types...)
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
