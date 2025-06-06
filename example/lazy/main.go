package main

import (
	"fmt"

	"github.com/pubgo/dix/dixglobal"
	"github.com/pubgo/funk/errors"
	"github.com/pubgo/funk/recovery"
)

func main() {
	defer recovery.Exit()

	type handler struct{}
	dixglobal.Provide(func() *handler {
		fmt.Println("1")
		return new(handler)
	})

	dixglobal.Provide(func() *handler {
		fmt.Println("2")
		return new(handler)
	})

	dixglobal.Provide(func(_ *handler) *errors.Err {
		return &errors.Err{Msg: "ok"}
	})

	dixglobal.Inject(func(err *errors.Err) {
		fmt.Println(err.Msg)
	})
}
