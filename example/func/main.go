package main

import (
	"fmt"

	"github.com/pubgo/dix/di"
	"github.com/pubgo/funk/recovery"
)

func main() {
	defer recovery.Exit()

	type handler func() string
	diglobal.Provide(func() handler {
		return func() string {
			return "hello"
		}
	})

	diglobal.Provide(func() handler {
		return func() string {
			return "world"
		}
	})

	type param struct {
		H    handler
		List []handler
	}

	fmt.Println(diglobal.Graph())

	fmt.Println("struct: ", diglobal.Inject(new(param)).H())
	diglobal.Inject(func(h handler, list []handler) {
		fmt.Println("inject: ", h())
		fmt.Println("inject: ", list)
	})

	diglobal.Inject(func(p param) {
		fmt.Println("inject struct: ", p.H())
		fmt.Println("inject struct: ", p.List)
	})

	fmt.Println(diglobal.Graph())
}
