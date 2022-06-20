package main

import (
	"fmt"

	"github.com/pubgo/dix"
	"github.com/pubgo/funk"
)

func main() {
	defer funk.RecoverAndExit()
	defer func() {
		fmt.Println(dix.Graph())
	}()

	type handler func() string
	type handlers []handler
	dix.Provider(func() handlers {
		return handlers{
			func() string {
				return "hello"
			},
		}
	})

	dix.Provider(func() handlers {
		return handlers{
			func() string {
				return "world"
			},
		}
	})

	dix.Provider(func() handler {
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

	hh := dix.Inject(new(param))
	//  default是最后一个
	fmt.Println("default: ", hh.M["default"]())
	for i := range hh.H {
		fmt.Println("struct:", hh.H[i]())
	}

	dix.Inject(func(p param) {
		//  default是最后一个
		fmt.Println("default struct: ", hh.M["default"]())
		for i := range hh.H {
			fmt.Println("struct struct:", hh.H[i]())
		}
	})
}
