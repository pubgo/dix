// Map nil/empty handling: inject map fields even when no provider exists.
//
// Run:
//
//	cd example/map-nil && go run .
package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
)

func main() {
	// WithValuesNull allows nil/empty collections instead of failing hard.
	di := dix.New(dix.WithValuesNull())

	type App struct {
		Errors map[string]error
	}

	app := &App{}
	dix.Inject(di, app)

	fmt.Printf("Errors is nil: %v\n", app.Errors == nil)
	fmt.Printf("Errors length: %d\n", len(app.Errors))

	dix.Inject(di, func(errs map[string]error) {
		fmt.Printf("function inject length: %d\n", len(errs))
	})
}
