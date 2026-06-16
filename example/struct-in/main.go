// Struct injection: fill struct fields (including nested structs).
//
// Run:
//
//	cd example/struct-in && go run .
package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
)

type Config struct {
	DSN string
}

type Database struct {
	Config *Config
}

type Metadata struct {
	Version string
}

type App struct {
	DB       *Database
	Metadata *Metadata
}

func main() {
	di := dix.New()

	dix.Provide(di, func() *Config {
		return &Config{DSN: "postgres://localhost/app"}
	})

	dix.Provide(di, func(cfg *Config) *Database {
		return &Database{Config: cfg}
	})

	dix.Provide(di, func() *Metadata {
		return &Metadata{Version: "v1"}
	})

	// Inject into a struct pointer: nested pointer fields are resolved recursively.
	app := &App{}
	dix.Inject(di, app)

	fmt.Println("app.DB.Config.DSN =", app.DB.Config.DSN)
	fmt.Println("app.Metadata.Version =", app.Metadata.Version)
}
