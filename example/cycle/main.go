package main

import (
	"strings"

	"github.com/pubgo/dix/di"
	"github.com/pubgo/funk/generic"
	"github.com/pubgo/funk/recovery"
	"github.com/pubgo/funk/try"
)

func main() {
	defer recovery.Exit()
	type (
		A struct {
		}

		B struct {
		}

		C struct {
		}

		D struct {
		}
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

	var err = try.Try(func() error {
		di.Inject(func(*A) {})
		di.Inject(func(*B) {})
		di.Inject(func(*C) {})
		return nil
	})

	if !generic.IsNil(err) {
		if strings.Contains(err.Error(), "provider circular dependency") {
			return
		}

		panic(err)
	}
}
