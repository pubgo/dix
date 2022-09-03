package main

import (
	"strings"

	"github.com/pubgo/dix/di"
	"github.com/pubgo/funk/recovery"
	"github.com/pubgo/funk/result"
	"github.com/pubgo/funk/xtry"
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

	xtry.Try(func() {
		di.Inject(func(*A) {})
		di.Inject(func(*B) {})
		di.Inject(func(*C) {})
	}).Do(func(err result.Error) {
		if strings.Contains(err.Err().Error(), "provider circular dependency") {
			return
		}

		err.Must()
	})
}
