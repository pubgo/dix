// Function injection: register and inject func dependencies.
//
// Run:
//
//	cd example/func && go run .
package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
)

// Greet is a simple callable dependency.
type Greet func() string

func main() {
	di := dix.New()

	// 1) Register multiple providers for the same func type.
	dix.Provide(di, func() Greet {
		return func() string { return "hello" }
	})
	dix.Provide(di, func() Greet {
		return func() string { return "world" }
	})

	// 2) Inject a single func: dix picks the latest registered one.
	dix.Inject(di, func(g Greet) {
		fmt.Println("single func:", g())
	})

	// 3) Inject all func values as a slice.
	dix.Inject(di, func(all []Greet) {
		fmt.Print("all funcs:")
		for _, fn := range all {
			fmt.Print(" ", fn())
		}
		fmt.Println()
	})
}
