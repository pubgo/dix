package main

import (
	"fmt"

	"github.com/pubgo/dix/dixglobal"
	"github.com/pubgo/funk/log"
	"github.com/pubgo/funk/recovery"
)

func main() {
	defer recovery.Exit()
	defer func() {
		fmt.Println("\n=== Final Dependency Graph ===")
		graph := dixglobal.Graph()
		fmt.Printf("Providers:\n%s\n", graph.Providers)
		fmt.Printf("Objects:\n%s\n", graph.Objects)
	}()

	fmt.Println("=== List Nil Demo (No Providers Registered) ===")

	type handler func() string

	fmt.Println("\n=== Function Injection with Empty List ===")
	dixglobal.Inject(func(handlers []handler) {
		log.Printf("Function injection - handlers count: %d", len(handlers))
		if len(handlers) == 0 {
			fmt.Println("No handlers available (empty list)")
		} else {
			for i, h := range handlers {
				fmt.Printf("  handler[%d]: %s\n", i, h())
			}
		}
	})

	fmt.Println("\n=== Struct Injection with Empty List ===")
	type param struct {
		H []handler
	}

	p := dixglobal.Inject(new(param))
	log.Printf("Struct injection - handlers count: %d", len(p.H))
	if len(p.H) == 0 {
		fmt.Println("Struct field H is empty list")
	} else {
		for i, h := range p.H {
			fmt.Printf("  struct.H[%d]: %s\n", i, h())
		}
	}

	fmt.Println("\n=== Struct Parameter Injection with Empty List ===")
	dixglobal.Inject(func(p param) {
		log.Printf("Struct param injection - handlers count: %d", len(p.H))
		if len(p.H) == 0 {
			fmt.Println("Struct parameter H is empty list")
		} else {
			for i, h := range p.H {
				fmt.Printf("  param.H[%d]: %s\n", i, h())
			}
		}
	})

	fmt.Println("\n=== Get API with Empty List ===")
	// 使用Get API获取空列表
	handlers := dixglobal.Get[[]handler]()
	log.Printf("Get API - handlers count: %d", len(handlers))
	if len(handlers) == 0 {
		fmt.Println("Get API returned empty list")
	}

	fmt.Println("\n=== Dependency Graph (should show no providers) ===")
	graph := dixglobal.Graph()
	fmt.Printf("Providers:\n%s\n", graph.Providers)
}
