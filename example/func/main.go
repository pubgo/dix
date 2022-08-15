package main

import (
	"fmt"

	"github.com/pubgo/funk/recovery"

	"github.com/pubgo/dix"
)

func main() {
	defer recovery.Exit()
	
	type handler func() string
	dix.Provider(func() handler {
		return func() string {
			return "hello"
		}
	})

	dix.Provider(func() handler {
		return func() string {
			return "world"
		}
	})

	type param struct {
		H    handler
		List []handler
	}

	fmt.Println(dix.Graph())

	fmt.Println("struct: ", dix.Inject(new(param)).H())
	dix.Inject(func(h handler, list []handler) {
		fmt.Println("inject: ", h())
		fmt.Println("inject: ", list)
	})

	dix.Inject(func(p param) {
		fmt.Println("inject struct: ", p.H())
		fmt.Println("inject struct: ", p.List)
	})

	fmt.Println(dix.Graph())
}
