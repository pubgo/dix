// List nil/empty handling: inject slice fields even when no provider exists.
//
// Run:
//
//	cd example/list-nil && go run .
package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
)

type Handler func() string

func main() {
	di := dix.New(dix.WithValuesNull())

	type App struct {
		Handlers []Handler
	}

	app := &App{}
	dix.Inject(di, app)

	fmt.Printf("Handlers is nil: %v\n", app.Handlers == nil)
	fmt.Printf("Handlers length: %d\n", len(app.Handlers))

	dix.Inject(di, func(handlers []Handler) {
		fmt.Printf("function inject length: %d\n", len(handlers))
	})
}
