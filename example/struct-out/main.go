package main

import (
	"fmt"

	"github.com/pubgo/dix/di"
	"github.com/pubgo/funk/pretty"
	"github.com/pubgo/funk/recovery"
)

type Inline struct {
	M *C1
}

type D struct {
	M C
}

type C interface{}
type C1 struct {
	Name string
}

type Conf struct {
	Inline
	A  *A
	B  *B
	C  C
	D  *D
	D1 *D
}

type A struct {
	Hello string
}

type B struct {
	Hello string
}

func main() {
	defer recovery.Exit(func(err error) error {
		fmt.Println(di.Graph())
		return err
	})

	di.Provide(func() Conf {
		return Conf{
			A: &A{Hello: "hello-a"},
			B: &B{Hello: "hello-b"},
			C: "hello",
			D: &D{
				M: "hello",
			},
			D1: &D{
				M: "hello d1",
			},
			Inline: Inline{M: &C1{Name: "c1"}},
		}
	})

	di.Inject(func(a *A, b *B, cc []C, c1 *C1, c2 []*C1, d *D, dd []*D) {
		pretty.Println(a.Hello, "1", b.Hello, "2", cc, "3", c1, "4", c2, "c2", d, "5", dd)
	})

	fmt.Println(di.Graph())
}
