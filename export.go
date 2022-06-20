package dix

import (
	"github.com/pubgo/funk"
)

func (x *dix) Register(param any) {
	defer funk.RecoverAndRaise(func(err funk.XErr) funk.XErr {
		return err.WrapF("param=%#v", param)
	})

	x.provider(param)
}

func (x *dix) Provider(param any) {
	defer funk.RecoverAndRaise(func(err funk.XErr) funk.XErr {
		return err.WrapF("param=%#v", param)
	})

	x.provider(param)
}

func (x *dix) Inject(param any, opts ...Option) any {
	defer funk.RecoverAndRaise(func(err funk.XErr) funk.XErr {
		return err.WrapF("param=%#v", param)
	})

	return x.inject(param, opts...)
}

func (x *dix) Graph() *graph {
	return &graph{
		Objects:  x.objectGraph(),
		Provider: x.providerGraph(),
	}
}

type graph struct {
	Objects  string
	Provider string
}
