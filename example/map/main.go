// Map injection (namespace): group dependencies by map key.
//
// Run:
//
//	cd example/map && go run .
package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
)

type Database struct {
	DSN string
}

func main() {
	di := dix.New()

	// Multiple providers can contribute to the same map[string]*T.
	dix.Provide(di, func() map[string]*Database {
		return map[string]*Database{
			"master": {DSN: "postgres://master/db"},
			"slave":  {DSN: "postgres://slave/db"},
		}
	})

	dix.Provide(di, func() map[string]*Database {
		return map[string]*Database{
			"analytics": {DSN: "postgres://analytics/db"},
		}
	})

	dix.Inject(di, func(dbs map[string]*Database) {
		fmt.Println("master:", dbs["master"].DSN)
		fmt.Println("slave:", dbs["slave"].DSN)
		fmt.Println("analytics:", dbs["analytics"].DSN)
	})
}
