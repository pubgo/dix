// Lazy resolution: providers run only when their output is first needed.
//
// Run:
//
//	cd example/lazy && go run .
package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
)

type Service struct {
	Name string
}

func main() {
	di := dix.New()
	order := make([]string, 0, 2)

	dix.Provide(di, func() *Service {
		order = append(order, "provider-A")
		fmt.Println("provider A executed")
		return &Service{Name: "A"}
	})

	dix.Provide(di, func() *Service {
		order = append(order, "provider-B")
		fmt.Println("provider B executed")
		return &Service{Name: "B"}
	})

	// Provider C depends on *Service. Only one *Service provider chain is used.
	dix.Provide(di, func(_ *Service) error {
		order = append(order, "provider-C")
		fmt.Println("provider C executed")
		return fmt.Errorf("ready")
	})

	dix.Inject(di, func(err error) {
		fmt.Println("inject got error:", err)
	})

	fmt.Println("execution order:", order)
}
