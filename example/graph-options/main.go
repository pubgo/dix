package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
	"github.com/pubgo/dix/v2/dixglobal"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic: %v\n", r)
		}
	}()

	// Register some providers
	dixglobal.Provide(func() *Config {
		return &Config{Endpoint: "localhost:8080"}
	})

	dixglobal.Provide(func(c *Config) *Database {
		return &Database{Config: c}
	})

	dixglobal.Provide(func(db *Database) *Service {
		return &Service{DB: db}
	})

	dixglobal.Provide(func(s *Service) *Handler {
		return &Handler{Svc: s}
	})

	// Generate full dependency graph (default)
	fmt.Println("=== Full Dependency Graph ===")
	fmt.Println(dixglobal.Graph().ProviderTypes)

	// Generate graph with options
	fmt.Println("\n=== Graph with Max Depth 2 ===")
	opts := dix.NewGraphOptions()
	opts.MaxDepth = 2
	fmt.Println(dixglobal.GraphWithOptions(opts).ProviderTypes)

	// Generate graph grouped by package
	fmt.Println("\n=== Graph Grouped by Package ===")
	opts = dix.NewGraphOptions()
	opts.GroupByPackage = true
	fmt.Println(dixglobal.GraphWithOptions(opts).ProviderTypes)

	// Generate graph with package filtering
	fmt.Println("\n=== Graph Filtering 'config' Package ===")
	opts = dix.NewGraphOptions()
	opts.FilterPackages = []string{"main"} // In this example, all types are in main package
	fmt.Println(dixglobal.GraphWithOptions(opts).ProviderTypes)
}

type Config struct {
	Endpoint string
}

type Database struct {
	Config *Config
}

type Service struct {
	DB *Database
}

type Handler struct {
	Svc *Service
}