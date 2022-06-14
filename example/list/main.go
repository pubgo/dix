package main

import (
	"fmt"

	"github.com/pubgo/xerror"

	"github.com/pubgo/dix"
)

func main() {
	defer xerror.RecoverAndExit()
	type handler func() string
	dix.Register(func() handler {
		return func() string {
			return "hello"
		}
	})

	dix.Register(func() handler {
		return func() string {
			return "world"
		}
	})

	dix.Inject(func(handlers []handler, a handler) {
		fmt.Println("default: ", a())
		for i := range handlers {
			fmt.Println("fn:", handlers[i]())
		}
	})

	type param struct {
		H []handler
		A handler
		M map[string]handler
	}

	hh := dix.Inject(new(param)).(*param)
	fmt.Println("default: ", hh.A())
	fmt.Println("default: ", hh.M["default"]())
	for i := range hh.H {
		fmt.Println("struct:", hh.H[i]())
	}
	fmt.Println(dix.Graph())
}
