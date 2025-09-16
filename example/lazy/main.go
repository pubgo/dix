package main

import (
	"fmt"

	"github.com/pubgo/dix/v2/dixglobal"
	"github.com/pubgo/funk/v2/errors"
	"github.com/pubgo/funk/v2/recovery"
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
