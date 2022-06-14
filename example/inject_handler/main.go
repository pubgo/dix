package main

import (
	"fmt"
	"github.com/pubgo/dix"
	"github.com/pubgo/xerror"
)

type handler struct {
}

func (h *handler) Inject(err *xerror.Err) {
	fmt.Println(err.Msg)
}

func main() {
	dix.Provider(func() *xerror.Err {
		return &xerror.Err{Msg: "ok"}
	})

	dix.Inject(&handler{})
}
