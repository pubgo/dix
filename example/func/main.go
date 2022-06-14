package main

import (
	"fmt"

	"github.com/pubgo/dix"
)

func main() {
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

	type param struct {
		H handler
	}

	fmt.Println("struct: ", dix.Inject(new(param)).(*param).H())
	dix.Inject(func(h handler) {
		fmt.Println("inject: ", h())
	})
	fmt.Println(dix.Graph())
}
