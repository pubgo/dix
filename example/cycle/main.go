package main

import (
	"fmt"

	"github.com/pubgo/dix/v2/dixglobal"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic: %v\n", r)
		}
	}()

	type (
		A struct{}

		B struct{}

		C struct{}
	)

	dixglobal.Provide(func(*B) *A {
		return new(A)
	})

	dixglobal.Provide(func(*C) *B {
		return new(B)
	})

	dixglobal.Provide(func(*A) *C {
		return new(C)
	})
	dixglobal.Inject(func(*C) {})
}
