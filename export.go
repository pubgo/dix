package dix

import (
	"github.com/pubgo/xerror"
)

func (x *dix) Register(param interface{}) {
	defer xerror.RecoverAndRaise(func(err xerror.XErr) xerror.XErr {
		return err.WrapF("param=%#v", param)
	})

	x.register(param)
}

func (x *dix) Inject(param interface{}) {
	defer xerror.RecoverAndRaise(func(err xerror.XErr) xerror.XErr {
		return err.WrapF("param=%#v", param)
	})

	x.inject(param)
}

func (x *dix) Invoke() { x.invoke() }

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
