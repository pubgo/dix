package main

import (
	"fmt"

	"github.com/pubgo/dix/dixglobal"
	"github.com/pubgo/funk/recovery"
)

func main() {
	defer recovery.Exit()

	type handler func() string
	dixglobal.Provide(func() handler {
		return func() string {
			return "hello"
		}
	})

	dixglobal.Provide(func() handler {
		return func() string {
			return "world"
		}
	})

	type param struct {
		H    handler
		List []handler
	}

	fmt.Println(dixglobal.Graph())

	fmt.Println("struct: ", dixglobal.Inject(new(param)).H())
	dixglobal.Inject(func(h handler, list []handler) {
		fmt.Println("inject: ", h())
		fmt.Println("inject: ", list)
	})

	dixglobal.Inject(func(p param) {
		fmt.Println("inject struct: ", p.H())
		fmt.Println("inject struct: ", p.List)
	})

	fmt.Println(dixglobal.Graph())
}
