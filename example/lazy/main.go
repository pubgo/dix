package main

import (
	"fmt"

	"github.com/pubgo/dix"
	"github.com/pubgo/funk"
)

func main() {
	defer funk.RecoverAndExit()
	type handler struct{}
	dix.Provider(func() *handler {
		fmt.Println("1")
		return new(handler)
	})

	dix.Provider(func() *handler {
		fmt.Println("2")
		return new(handler)
	})

	dix.Provider(func(_ *handler) *funk.Err {
		return &funk.Err{Msg: "ok"}
	})

	dix.Inject(func(err *funk.Err) {
		fmt.Println(err.Msg)
	})
}
