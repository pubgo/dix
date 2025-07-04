package main

import (
	"fmt"

	"github.com/pubgo/dix/dixglobal"
	"github.com/pubgo/funk/errors"
)

type handler struct{}

func (h *handler) DixInjectA(err *errors.Err) {
	fmt.Println("A: ", err.Msg)
}

func (h *handler) DixInjectD(p struct {
	Err *errors.Err
},
) {
	fmt.Println("D: ", p.Err.Msg)
}

func (h *handler) DixInjectC(errs []*errors.Err) {
	fmt.Println("C: ", errs)
}

func (h *handler) DixInjectB(err *errors.Err, errs []*errors.Err) {
	fmt.Println("B: ", err.Msg, errs)
}

func main() {
	dixglobal.Provide(func() *errors.Err {
		return &errors.Err{Msg: "<ok>"}
	})

	dixglobal.Provide(func() *errors.Err {
		return &errors.Err{Msg: "<ok 1>"}
	})

	dixglobal.Inject(&handler{})
}
