package main

import (
	"fmt"

	"github.com/pubgo/dix/dixglobal"
	"github.com/pubgo/funk/recovery"
)

func main() {
	defer recovery.Exit()
	defer func() {
		fmt.Println(dixglobal.Graph())
	}()

	type handler func() string
	type handlers []handler
	dixglobal.Provide(func() handlers {
		return handlers{
			func() string {
				return "hello"
			},
		}
	})

	dixglobal.Provide(func() handlers {
		return handlers{
			func() string {
				return "world"
			},
		}
	})

	dixglobal.Provide(func() handler {
		return func() string {
			return "world next"
		}
	})

	fmt.Println(dixglobal.Graph())

	dixglobal.Inject(func(handlers handlers, h handler) {
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

	hh := dixglobal.Inject(new(param))
	//  default是最后一个
	fmt.Println("default: ", hh.M["default"]())
	for i := range hh.H {
		fmt.Println("struct:", hh.H[i]())
	}

	dixglobal.Inject(func(p param) {
		//  default是最后一个
		fmt.Println("default struct: ", hh.M["default"]())
		for i := range hh.H {
			fmt.Println("struct struct:", hh.H[i]())
		}
	})
}
