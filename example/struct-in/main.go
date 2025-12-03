package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
	"github.com/pubgo/dix/v2/dixglobal"
)

type a struct {
	b
	B b
}

type b struct {
	C *c
}

type c struct {
	C string
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic: %v\n", r)
		}
	}()

	dixglobal.Provide(func() *c {
		return &c{C: "hello"}
	})

	arg := dixglobal.Inject(new(a))
	if arg.C.C != "hello" {
		panic("not match")
	}
	fmt.Println(arg.C.C)
	fmt.Println(arg.B.C.C)
	fmt.Println(dixglobal.Graph())

	dixglobal.Provide(func(a a1, di *dix.Dix, dd map[string][]*dix.Dix) *a2 {
		fmt.Println(dd)
		return &a2{Hello: "a2", di: di}
	})

	dixglobal.Inject(func(a *a2) {
		fmt.Println(a.Hello)
		fmt.Println(a.di.Option())
	})
}

type a1 struct {
	b
}

type a2 struct {
	Hello string
	di    *dix.Dix
}
