package main

import (
	"fmt"

	"github.com/pubgo/dix/dixglobal"
	"github.com/pubgo/funk/recovery"
)

func main() {
	defer recovery.Exit()
	defer func() {
		fmt.Println(dixglobal.Graph())
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
