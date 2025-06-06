package main

import (
	"fmt"

	"github.com/pubgo/dix/di"
	"github.com/pubgo/funk/recovery"
)

func main() {
	defer recovery.Exit()
	defer func() {
		fmt.Println(diglobal.Graph())
	}()

	type handler func() string
	type handlers []handler
	diglobal.Provide(func() handlers {
		return handlers{
			func() string {
				return "hello"
			},
		}
	})

	diglobal.Provide(func() handlers {
		return handlers{
			func() string {
				return "world"
			},
		}
	})

	diglobal.Provide(func() handler {
		return func() string {
			return "world next"
		}
	})

	fmt.Println(diglobal.Graph())

	diglobal.Inject(func(handlers handlers, h handler) {
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

	hh := diglobal.Inject(new(param))
	//  default是最后一个
	fmt.Println("default: ", hh.M["default"]())
	for i := range hh.H {
		fmt.Println("struct:", hh.H[i]())
	}

	diglobal.Inject(func(p param) {
		//  default是最后一个
		fmt.Println("default struct: ", hh.M["default"]())
		for i := range hh.H {
			fmt.Println("struct struct:", hh.H[i]())
		}
	})
}
