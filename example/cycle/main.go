package main

import (
	"strings"

	"github.com/pubgo/funk"
	"github.com/pubgo/funk/assert"
	"github.com/pubgo/funk/recovery"
	"github.com/pubgo/funk/xerr"

	"github.com/pubgo/dix"
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

		assert.Must(err)
	})
}
