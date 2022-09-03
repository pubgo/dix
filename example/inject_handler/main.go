package main

import (
	"fmt"
	"github.com/pubgo/dix/di"

	"github.com/pubgo/funk/xerr"
)

type handler struct {
}

func (h *handler) DixInjectA(err *xerr.Err) {
	fmt.Println("A: ", err.Msg)
}

func (h *handler) DixInjectD(p struct {
	Err *xerr.Err
}) {
	fmt.Println("D: ", p.Err.Msg)
}

func (h *handler) DixInjectC(errs []*xerr.Err) {
	fmt.Println("C: ", errs)
}

func (h *handler) DixInjectB(err *xerr.Err, errs []*xerr.Err) {
	fmt.Println("B: ", err.Msg, errs)
}

func main() {
	di.Provide(func() *xerr.Err {
		return &xerr.Err{Msg: "<ok>"}
	})

	di.Provide(func() *xerr.Err {
		return &xerr.Err{Msg: "<ok 1>"}
	})

	di.Inject(&handler{})
}
