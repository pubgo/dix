package main

import (
	"fmt"

	"github.com/pubgo/dix"
	"github.com/pubgo/xerror"
)

func main() {
	defer xerror.RecoverAndExit()
	type handler struct{}
	dix.Provider(func() *handler {
		fmt.Println("1")
		return new(handler)
	})

	dix.Provider(func() *handler {
		fmt.Println("2")
		return new(handler)
	})

	dix.Provider(func(_ *handler) *xerror.Err {
		return &xerror.Err{Msg: "ok"}
	})

	dix.Inject(func(err *xerror.Err) {
		fmt.Println(err.Msg)
	})
}
