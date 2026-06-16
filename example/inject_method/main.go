// Method injection: call DixInject* methods after struct creation.
//
// Run:
//
//	cd example/inject_method && go run .
package main

import (
	"errors"
	"fmt"

	"github.com/pubgo/dix/v2"
)

type Worker struct{}

// Methods prefixed with DixInject receive dependencies after Worker is created.
func (w *Worker) DixInjectLogger(err error) {
	fmt.Println("logger dependency:", err.Error())
}

func (w *Worker) DixInjectTags(errs []error) {
	fmt.Print("tag dependencies:")
	for i, e := range errs {
		fmt.Printf(" [%d]=%s", i, e.Error())
	}
	fmt.Println()
}

func main() {
	di := dix.New()

	dix.Provide(di, func() error { return errors.New("primary") })
	dix.Provide(di, func() error { return errors.New("secondary") })

	dix.Inject(di, &Worker{})
}
