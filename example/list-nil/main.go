package main

import (
	"fmt"

	"github.com/pubgo/dix/di"

	"github.com/pubgo/funk/recovery"
)

func main() {
	defer recovery.Exit()
	defer func() {
		fmt.Println(di.Graph())
	}()

	type handler func() string

	di.Inject(func(handlers []handler) {
		for i := range handlers {
			fmt.Println("fn:", handlers[i]())
		}
	})

	type param struct {
		H []handler
	}

	hh := di.Inject(new(param))
	for i := range hh.H {
		fmt.Println("struct:", hh.H[i]())
	}

	di.Inject(func(p param) {
		for i := range hh.H {
			fmt.Println("struct struct:", hh.H[i]())
		}
	})

	fmt.Println(di.Graph())
}
