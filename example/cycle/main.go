package main

import (
	"fmt"

	"github.com/pubgo/dix/di"
	"github.com/pubgo/funk/recovery"
)

func main() {
	defer recovery.Exit()
	defer func() {
		fmt.Println(diglobal.Graph())
	}()

	type (
		A struct{}

		B struct{}

		C struct{}
	)

	diglobal.Provide(func(*B) *A {
		return new(A)
	})

	diglobal.Provide(func(*C) *B {
		return new(B)
	})

	diglobal.Provide(func(*A) *C {
		return new(C)
	})
	diglobal.Inject(func(*C) {})
}
