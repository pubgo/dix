package main

import (
	"fmt"
	"strings"

	"github.com/pubgo/dix/dixglobal"
	"github.com/pubgo/funk/recovery"
)

func main() {
	defer recovery.Exit(func(err error) error {
		if strings.Contains(err.Error(), "circular dependency detected") {
			return nil
		}
		return err
	})
	
	defer func() {
		fmt.Println("\n=== Final Dependency Graph ===")
		graph := dixglobal.Graph()
		fmt.Printf("Providers:\n%s\n", graph.Providers)
		fmt.Printf("Objects:\n%s\n", graph.Objects)
	}()

	type (
		A struct{ Name string }
		B struct{ Name string }
		C struct{ Name string }
	)

	fmt.Println("=== Registering Circular Dependencies ===")

	// 这些提供者形成循环依赖：A -> B -> C -> A
	fmt.Println("Registering A provider (depends on B)")
	dixglobal.Provide(func(b *B) *A {
		return &A{Name: "A depends on " + b.Name}
	})

	fmt.Println("Registering B provider (depends on C)")
	dixglobal.Provide(func(c *C) *B {
		return &B{Name: "B depends on " + c.Name}
	})

	fmt.Println("Registering C provider (depends on A)")

	dixglobal.Provide(func(a *A) *C {
		return &C{Name: "C depends on " + a.Name}
	})
}
