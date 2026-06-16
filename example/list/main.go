// List injection: aggregate multiple providers of the same type into a slice.
//
// Run:
//
//	cd example/list && go run .
package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
)

type Middleware func(string) string

func main() {
	di := dix.New()

	dix.Provide(di, func() []Middleware {
		return []Middleware{
			func(s string) string { return "[auth] " + s },
		}
	})

	dix.Provide(di, func() []Middleware {
		return []Middleware{
			func(s string) string { return "[log] " + s },
		}
	})

	// Single value injection uses the latest registered provider.
	dix.Provide(di, func() Middleware {
		return func(s string) string { return "[trace] " + s }
	})

	dix.Inject(di, func(latest Middleware, chain []Middleware) {
		fmt.Println("latest:", latest("request"))
		fmt.Println("chain size:", len(chain))
		for i, mw := range chain {
			fmt.Printf("  middleware[%d]: %s\n", i, mw("request"))
		}
	})
}
