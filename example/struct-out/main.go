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
	D2 map[string]*D
	D3 []*D
	D4 map[string][]*D
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
			D2: map[string]*D{
				"default1": {
					M: "hello d2",
				},
			},
			D3: []*D{
				{
					M: "hello d3",
				},
			},
			D4: map[string][]*D{
				"default4": []*D{
					{
						M: "hello d4",
					},
				},
			},
			Inline: Inline{M: &C1{Name: "c1"}},
		}
	})

	di.Inject(func(a *A, b *B, cc []C, c1 *C1, c2 []*C1, d *D, dd []*D, dm map[string]*D, d5 map[string][]*D) {
		pretty.Println(a.Hello)
		pretty.Println(b.Hello)
		pretty.Println(cc)
		pretty.Println(c1)
		pretty.Println(c2)
		pretty.Println(d)
		pretty.Println(dd)
		pretty.Println(dm)
		pretty.Println(d5)
	})

	fmt.Println(di.Graph())
}
