package main

import (
	"fmt"

	"github.com/pubgo/dix"
	"github.com/pubgo/xerror"
)

func main() {
	defer xerror.RecoverAndExit()
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

	fmt.Println(dix.Graph())
}
