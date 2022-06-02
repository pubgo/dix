package dix

import (
	"fmt"

	"github.com/pubgo/dix/internal/assert"
)

func (x *dix) Register(param interface{}) {
	defer assert.Recovery(func(err error) {
		panic(&Err{
			Err:    err,
			Msg:    err.Error(),
			Detail: fmt.Sprintf("param=%#v", param),
		})
	})

	x.register(param)
}

func (x *dix) Inject(param interface{}) {
	defer assert.Recovery(func(err error) {
		panic(&Err{
			Err:    err,
			Msg:    err.Error(),
			Detail: fmt.Sprintf("param=%#v", param),
		})
	})

	x.inject(param)
}

func (x *dix) Invoke() {
	defer assert.Recovery(func(err error) {
		panic(&Err{
			Err: err,
			Msg: err.Error(),
		})
	})
	x.invoke()
}

func (x *dix) Graph() string {
	defer assert.Recovery(func(err error) {
		panic(&Err{
			Err: err,
			Msg: err.Error(),
		})
	})
	return x.graph()
}
