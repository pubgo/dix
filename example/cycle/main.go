// Cycle detection: dix rejects circular dependencies before injection.
//
// Run:
//
//	cd example/cycle && go run .
package main

import (
	"fmt"
	"log"

	"github.com/pubgo/dix/v2"
)

type (
	ServiceA struct{}
	ServiceB struct{}
	ServiceC struct{}
)

func main() {
	di := dix.New()

	// A -> B -> C -> A (cycle)
	dix.Provide(di, func(*ServiceB) *ServiceA { return &ServiceA{} })
	dix.Provide(di, func(*ServiceC) *ServiceB { return &ServiceB{} })
	dix.Provide(di, func(*ServiceA) *ServiceC { return &ServiceC{} })

	// Prefer TryInject in production to avoid panic.
	if err := dix.TryInject(di, func(*ServiceC) {}); err != nil {
		log.Println("cycle detected:", err)
		return
	}

	fmt.Println("unexpected: cycle was not detected")
}
