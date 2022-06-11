package dix

import (
	"testing"

	"github.com/pubgo/xerror"
)

func TestCycle(t *testing.T) {
	defer xerror.RecoverTest(t, true)
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

	var c = newDix()
	c.register(func(*B) *A {
		return new(A)
	})
	c.register(func(*C) *B {
		return new(B)
	})
	c.register(func(*D) *C {
		return new(C)
	})

	c.Register(func(*A) *C {
		return new(C)
	})
}
