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

	type param struct {
		H handler `inject:""`
	}
	fmt.Println(dix.Inject(new(param)).(*param).H())
	fmt.Println(dix.Graph())
}
