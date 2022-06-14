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

	dix.Register(func() handler {
		return func() string {
			return "world next"
		}
	})

	fmt.Println(dix.Graph())

	dix.Inject(func(handlers handlers, h handler) {
		// h为默认的, 最后一个注册的
		fmt.Println("the last: default: ", h())
		for i := range handlers {
			fmt.Println("fn:", handlers[i]())
		}
	})

	type param struct {
		H handlers
		M map[string]handler
	}

	hh := dix.Inject(new(param)).(*param)
	//  default是最后一个
	fmt.Println("default: ", hh.M["default"]())
	for i := range hh.H {
		fmt.Println("struct:", hh.H[i]())
	}
	fmt.Println(dix.Graph())
}
