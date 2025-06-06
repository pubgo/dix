package main

import (
	"fmt"

	"github.com/pubgo/dix/di"
	"github.com/pubgo/funk/log"
	"github.com/pubgo/funk/recovery"
)

func main() {
	defer recovery.Exit()
	defer func() {
		fmt.Println(diglobal.Graph())
	}()

	type handler func() string

	diglobal.Inject(func(handlers []handler) {
		log.Printf("handlers: %d", len(handlers))
		for i := range handlers {
			fmt.Println("fn:", handlers[i]())
		}
	})

	type param struct {
		H []handler
	}

	hh := diglobal.Inject(new(param))
	log.Printf("handlers: %d", len(hh.H))
	for i := range hh.H {
		fmt.Println("struct:", hh.H[i]())
	}

	diglobal.Inject(func(p param) {
		log.Printf("handlers: %d", len(p.H))
		for i := range p.H {
			fmt.Println("struct struct:", p.H[i]())
		}
	})

	fmt.Println(diglobal.Graph())
}
