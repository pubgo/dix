package main

import (
	"fmt"

	"github.com/pubgo/xerror"

	"github.com/pubgo/dix"
)

func main() {
	defer xerror.RecoverAndExit()
	type handler func() string
	type handlers []handler
	dix.Register(func() handlers {
		return handlers{
			func() string {
				return "hello"
			},
		}
	})

	dix.Register(func() handlers {
		return handlers{
			func() string {
				return "world"
			},
		}
	})

	fmt.Println(dix.Graph())

	dix.Inject(func(handlers handlers) {
		for i := range handlers {
			fmt.Println("fn:", handlers[i]())
		}
	})

	type param struct {
		H handlers
		M map[string]handler
	}

	hh := dix.Inject(new(param)).(*param)
	fmt.Println("default: ", hh.M["default"]())
	for i := range hh.H {
		fmt.Println("struct:", hh.H[i]())
	}
	fmt.Println(dix.Graph())
}
