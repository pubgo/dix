package main

import (
	"errors"
	"fmt"

	"github.com/pubgo/dix/v2/dixglobal"
)

type handler struct{}

func (h *handler) DixInjectA(err error) {
	fmt.Println("A: ", err.Error())
}

func (h *handler) DixInjectD(p struct {
	Err error
},
) {
	fmt.Println("D: ", p.Err.Error())
}

func (h *handler) DixInjectC(errs []error) {
	for i, e := range errs {
		fmt.Printf("C[%d]: %s\n", i, e.Error())
	}
}

func (h *handler) DixInjectB(err error, errs []error) {
	fmt.Println("B: ", err.Error(), errs)
}

func main() {
	dixglobal.Provide(func() error {
		return errors.New("<ok>")
	})

	dixglobal.Provide(func() error {
		return errors.New("<ok 1>")
	})

	dixglobal.Inject(&handler{})
}
