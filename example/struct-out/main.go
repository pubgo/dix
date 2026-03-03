package main

import (
	"fmt"

	"github.com/pubgo/dix/v2/dixglobal"
)

type Inline struct {
	M *C1
}

type D struct {
	M C
}

type (
	C  interface{}
	C1 struct {
		Name string
	}
)

type Conf struct {
	Data string
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
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("panic: %v", r)
			}
			fmt.Printf("panic: %v\n", err)
		}
	}()

	dixglobal.Provide(func() Conf {
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
				"default4": {
					{
						M: "hello d4",
					},
				},
			},
			Inline: Inline{M: &C1{Name: "c1"}},
		}
	})

	dixglobal.Inject(func(a *A, b *B, cc []C, c1 *C1, c2 []*C1, d *D, dd []*D, dm map[string]*D, d5 map[string][]*D) {
		fmt.Println(a.Hello)
		fmt.Println(b.Hello)
		fmt.Println(cc)
		fmt.Println(c1)
		fmt.Println(c2)
		fmt.Println(d)
		fmt.Println(dd)
		fmt.Println(dm)
		fmt.Println(d5)
	})
}
