package main

import (
	"fmt"

	"github.com/pubgo/dix/di"
	"github.com/pubgo/funk/recovery"
)

func main() {
	defer recovery.Exit()

	type handler func() string
	di.Provide(func() handler {
		return func() string {
			return "hello"
		}
	})

	di.Provide(func() handler {
		return func() string {
			return "world"
		}
	})

	type param struct {
		H    handler
		List []handler
	}

	fmt.Println(di.Graph())

	fmt.Println("struct: ", di.Inject(new(param)).H())
	di.Inject(func(h handler, list []handler) {
		fmt.Println("inject: ", h())
		fmt.Println("inject: ", list)
	})

	di.Inject(func(p param) {
		fmt.Println("inject struct: ", p.H())
		fmt.Println("inject struct: ", p.List)
	})

	fmt.Println(di.Graph())
}
