package main

import (
	"fmt"

	"github.com/pubgo/dix"
	"github.com/pubgo/xerror"
)

type handler struct {
}

func (h *handler) DixInjectA(err *xerror.Err) {
	fmt.Println("A: ", err.Msg)
}

func (h *handler) DixInjectC(errs []*xerror.Err) {
	fmt.Println("C: ", errs)
}

func (h *handler) DixInjectB(err *xerror.Err, errs []*xerror.Err) {
	fmt.Println("B: ", err.Msg, errs)
}

func main() {
	dix.Provider(func() *xerror.Err {
		return &xerror.Err{Msg: "<ok>"}
	})

	dix.Provider(func() *xerror.Err {
		return &xerror.Err{Msg: "<ok 1>"}
	})

	dix.Inject(&handler{})
}
