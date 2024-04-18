package main

import (
	"fmt"
	"strings"

	"github.com/pubgo/dix/di"
	"github.com/pubgo/funk/generic"
	"github.com/pubgo/funk/recovery"
	"github.com/pubgo/funk/try"
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

	err := try.Try(func() error {
		di.Provide(func(*A) *C {
			return new(C)
		})
		return nil
	})

	if !generic.IsNil(err) {
		if strings.Contains(err.Error(), "provider circular dependency") {
			return
		}

		panic(err)
	}
}
