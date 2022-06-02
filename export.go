package dix

import (
	"fmt"
)

func (x *dix) Register(param interface{}) {
	defer recovery(func(err *Err) {
		panic(&Err{
			Err:    err,
			Msg:    err.Error(),
			Detail: fmt.Sprintf("param=%#v", param),
		})
	})

	x.register(param)
}

func (x *dix) Inject(param interface{}) {
	defer recovery(func(err *Err) {
		panic(&Err{
			Err:    err,
			Msg:    err.Error(),
			Detail: fmt.Sprintf("param=%#v", param),
		})
	})

	x.inject(param)
}

func (x *dix) Invoke() {
	defer recovery(func(err *Err) {
		panic(&Err{
			Err: err,
			Msg: err.Error(),
		})
	})
	x.invoke()
}

func (x *dix) Graph() string {
	defer recovery(func(err *Err) {
		panic(&Err{
			Err: err,
			Msg: err.Error(),
		})
	})
	return x.graph()
}
