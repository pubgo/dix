package main

import (
	"fmt"

	"github.com/pubgo/dix/di"
)

type C interface{}

type Conf struct {
	A *A
	B *B
	C C
}

type A struct {
	Hello string
}

type B struct {
	Hello string
}

func main() {
	di.Provide(func() Conf {
		return Conf{
			A: &A{Hello: "hello-a"},
			B: &B{Hello: "hello-b"},
			C: "hello",
		}
	})

	di.Inject(func(a *A, b *B, cc C) {
		fmt.Println(a.Hello, b.Hello, cc)
	})
	fmt.Println(di.Graph())
}
