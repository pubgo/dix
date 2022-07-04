package dix

import (
	"github.com/pubgo/funk"
	"github.com/pubgo/funk/xerr"
)

func (x *Dix) Register(param any) {
	defer funk.RecoverAndRaise(func(err xerr.XErr) xerr.XErr {
		return err.WrapF("param=%#v", param)
	})

	x.provider(param)
}

func (x *Dix) Provider(param any) {
	defer funk.RecoverAndRaise(func(err xerr.XErr) xerr.XErr {
		return err.WrapF("param=%#v", param)
	})

	x.provider(param)
}

func (x *Dix) Inject(param any, opts ...Option) any {
	defer funk.RecoverAndRaise(func(err xerr.XErr) xerr.XErr {
		return err.WrapF("param=%#v", param)
	})

	return x.inject(param, opts...)
}

func (x *Dix) Dix(opts ...Option) *Dix {
	return x.dix(opts...)
}

func (x *Dix) Graph() *graph {
	return &graph{
		Objects:  x.objectGraph(),
		Provider: x.providerGraph(),
	}
}

type graph struct {
	Objects  string
	Provider string
}
