package main

import (
	"fmt"

	"github.com/pubgo/xerror"

	"github.com/pubgo/dix"
)

func main() {
	defer xerror.RecoverAndExit()
	type handler func() string
	dix.Register(func() []handler {
		return []handler{
			func() string {
				return "hello"
			},
			func() string {
				return "world"
			},
		}
	})

	dix.Register(func() []handler {
		return []handler{
			func() string {
				return "hello1"
			},
			func() string {
				return "world2"
			},
		}
	})

	dix.Inject(func(handlers []handler) {
		for i := range handlers {
			fmt.Println("fn:", handlers[i]())
		}
	})

	type param struct {
		H []handler
	}

	hh := dix.Inject(new(param)).(*param)
	for i := range hh.H {
		fmt.Println("struct:", hh.H[i]())
	}
	fmt.Println(dix.Graph())
}
