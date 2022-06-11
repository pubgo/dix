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

	dix.Register(func(h handler) {
		fmt.Println(h())
	})
	dix.Invoke()
	fmt.Println(dix.Graph())
}
