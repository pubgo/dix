package main

import (
	"strings"

	"github.com/pubgo/dix"
	"github.com/pubgo/xerror"
)

func main() {
	defer xerror.RecoverAndExit()
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

	var c = dix.New()
	c.Register(func(*B) *A {
		return new(A)
	})

	c.Register(func(*C) *B {
		return new(B)
	})

	c.Register(func(*D) *C {
		return new(C)
	})

	xerror.TryCatch(func() {
		c.Register(func(*A) *C {
			return new(C)
		})
	}, func(err error) {
		if strings.Contains(err.Error(), "provider circular dependency") {
			return
		}
		xerror.Panic(err)
	})
}
