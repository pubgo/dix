package main

import (
	"fmt"

	"github.com/pubgo/dix/v2/dixglobal"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic: %v\n", r)
		}
	}()
	defer func() {
		fmt.Println(dixglobal.Graph())
	}()

	type handler func() string

	dixglobal.Inject(func(handlers []handler) {
		fmt.Printf("handlers: %d\n", len(handlers))
		for i := range handlers {
			fmt.Println("fn:", handlers[i]())
		}
	})

	type param struct {
		H []handler
	}

	hh := dixglobal.Inject(new(param))
	fmt.Printf("handlers: %d\n", len(hh.H))
	for i := range hh.H {
		fmt.Println("struct:", hh.H[i]())
	}

	dixglobal.Inject(func(p param) {
		fmt.Printf("handlers: %d\n", len(p.H))
		for i := range p.H {
			fmt.Println("struct struct:", p.H[i]())
		}
	})

	fmt.Println(dixglobal.Graph())
}
