// Error handling: use TryProvide/TryInject instead of panic/recover.
//
// Run:
//
//	cd example/test-return-error && go run .
package main

import (
	"fmt"
	"log"

	"github.com/pubgo/dix/v2"
)

func main() {
	runSuccess()
	runProviderError()
	runInjectError()
}

func runSuccess() {
	di := dix.New()
	if err := dix.TryProvide(di, func() (*log.Logger, error) {
		return log.Default(), nil
	}); err != nil {
		log.Fatalf("provide failed: %v", err)
	}

	if err := dix.TryInject(di, func(l *log.Logger) error {
		l.Println("inject ok")
		return nil
	}); err != nil {
		log.Fatalf("inject failed: %v", err)
	}
}

func runProviderError() {
	di := dix.New()
	_ = dix.TryProvide(di, func() (*log.Logger, error) {
		return nil, fmt.Errorf("provider_err")
	})

	err := dix.TryInject(di, func(l *log.Logger) error {
		return nil
	})
	fmt.Println("provider error:", err)
}

func runInjectError() {
	di := dix.New()
	_ = dix.TryProvide(di, func() *log.Logger { return log.Default() })

	err := dix.TryInject(di, func(l *log.Logger) error {
		return fmt.Errorf("inject_err")
	})
	fmt.Println("inject error:", err)
}
