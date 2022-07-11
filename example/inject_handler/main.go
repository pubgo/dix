package main

import (
	"fmt"

	"github.com/pubgo/dix"
	"github.com/pubgo/funk"
)

type handler struct {
}

func (h *handler) DixInjectA(err *funk.Err) {
	fmt.Println("A: ", err.Msg)
}

func (h *handler) DixInjectD(p struct {
	Err *funk.Err
}) {
	fmt.Println("D: ", p.Err.Msg)
}

func (h *handler) DixInjectC(errs []*funk.Err) {
	fmt.Println("C: ", errs)
}

func (h *handler) DixInjectB(err *funk.Err, errs []*funk.Err) {
	fmt.Println("B: ", err.Msg, errs)
}

func main() {
	dix.Provider(func() *funk.Err {
		return &funk.Err{Msg: "<ok>"}
	})

	dix.Provider(func() *funk.Err {
		return &funk.Err{Msg: "<ok 1>"}
	})

	dix.Inject(&handler{})
}
