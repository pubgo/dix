package main

import (
	"strings"

	"github.com/pubgo/funk/recovery"
	"github.com/pubgo/funk/result"
	"github.com/pubgo/funk/xtry"

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

	xtry.Try(func() {
		c.Register(func(*A) *C {
			return new(C)
		})
	}).Do(func(err result.Error) {
		if strings.Contains(err.Err().Error(), "provider circular dependency") {
			return
		}

		err.Must()
	})
}
