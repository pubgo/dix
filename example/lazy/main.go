package main

import (
	"fmt"
	"github.com/pubgo/dix/di"

	"github.com/pubgo/funk/recovery"
	"github.com/pubgo/funk/xerr"
)

func main() {
	defer recovery.Exit()

	type handler struct{}
	di.Provide(func() *handler {
		fmt.Println("1")
		return new(handler)
	})

	di.Provide(func() *handler {
		fmt.Println("2")
		return new(handler)
	})

	di.Provide(func(_ *handler) *xerr.Err {
		return &xerr.Err{Msg: "ok"}
	})

	di.Inject(func(err *xerr.Err) {
		fmt.Println(err.Msg)
	})
}
