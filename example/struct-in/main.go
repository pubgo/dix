package main

import (
	"fmt"
	"github.com/pubgo/funk/recovery"

	"github.com/pubgo/dix"
	"github.com/pubgo/dix/di"
	"github.com/pubgo/funk/assert"
)

type a struct {
	B b
}

type b struct {
	C *c
}

type c struct {
	C string
}

func main() {
	defer recovery.Exit()

	di.Provide(func() *c {
		return &c{C: "hello"}
	})

	arg := di.Inject(new(a))
	assert.If(arg.B.C.C != "hello", "not match")
	fmt.Println(arg.B.C.C)
	fmt.Println(di.Graph())

	di.Provide(func(a a1, di *dix.Dix, dd map[string][]*dix.Dix) *a2 {
		fmt.Println(dd)
		return &a2{Hello: "a2", di: di}
	})

	di.Inject(func(a *a2) {
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
