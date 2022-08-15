package main

import (
	"fmt"

	"github.com/pubgo/funk/recovery"
	"github.com/pubgo/funk/xerr"

	"github.com/pubgo/dix"
)

func main() {
	defer recovery.Exit()

	type handler struct{}
	dix.Provider(func() *handler {
		fmt.Println("1")
		return new(handler)
	})

	dix.Provider(func() *handler {
		fmt.Println("2")
		return new(handler)
	})

	dix.Provider(func(_ *handler) *xerr.Err {
		return &xerr.Err{Msg: "ok"}
	})

	dix.Inject(func(err *xerr.Err) {
		fmt.Println(err.Msg)
	})
}
