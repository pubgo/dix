package main

import (
	"fmt"

	"github.com/pubgo/dix/di"
	"github.com/pubgo/funk/recovery"
)

type D struct {
	M C
}

type C interface{}

type Conf struct {
	A  *A
	B  *B
	C  C
	D  *D
	D1 D
}

type A struct {
	Hello string
}

type B struct {
	Hello string
}

func main() {
	defer recovery.Exit()
	di.Provide(func() Conf {
		return Conf{
			A: &A{Hello: "hello-a"},
			B: &B{Hello: "hello-b"},
			C: "hello",
			D: &D{
				M: "hello",
			},
			D1: D{
				M: "hello",
			},
		}
	})

	di.Inject(func(a *A, b *B, cc []C) {
		fmt.Println(a.Hello, b.Hello, cc)
	})

	fmt.Println(di.Graph())
}
