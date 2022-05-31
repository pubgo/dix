package dix

import (
	"testing"
)

func TestCycle(t *testing.T) {
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

	c.register(func(*A) *C {
		return new(C)
	})
}
