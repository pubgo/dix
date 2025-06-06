package main

import (
	"fmt"

	"github.com/pubgo/dix/di"
	"github.com/pubgo/funk/errors"
	"github.com/pubgo/funk/recovery"
)

func main() {
	defer recovery.Exit()

	type handler struct{}
	diglobal.Provide(func() *handler {
		fmt.Println("1")
		return new(handler)
	})

	diglobal.Provide(func() *handler {
		fmt.Println("2")
		return new(handler)
	})

	diglobal.Provide(func(_ *handler) *errors.Err {
		return &errors.Err{Msg: "ok"}
	})

	diglobal.Inject(func(err *errors.Err) {
		fmt.Println(err.Msg)
	})
}
