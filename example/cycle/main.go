package main

import (
	"fmt"
	
	"github.com/pubgo/dix/di"
	"github.com/pubgo/funk/recovery"
)

func main() {
	defer recovery.Exit()
	defer func() {
		fmt.Println(di.Graph())
	}()

	type (
		A struct{}

		B struct{}

		C struct{}
	)

	di.Provide(func(*B) *A {
		return new(A)
	})

	di.Provide(func(*C) *B {
		return new(B)
	})

	di.Provide(func(*A) *C {
		return new(C)
	})
	di.Inject(func(*C) {})
}
