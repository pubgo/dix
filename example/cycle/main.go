package main

import (
	"strings"

	"github.com/pubgo/dix"
	"github.com/pubgo/funk"
	"github.com/pubgo/funk/xerr"
)

func main() {
	defer funk.RecoverAndExit()
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

	funk.TryCatch(func() {
		c.Register(func(*A) *C {
			return new(C)
		})
	}, func(err xerr.XErr) {
		if strings.Contains(err.Error(), "provider circular dependency") {
			return
		}

		funk.Must(err)
	})
}
