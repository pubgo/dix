package main

import (
	"fmt"
	"github.com/pubgo/dix/di"
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
	di.Provide(func() *c {
		return &c{C: "hello"}
	})

	arg := di.Inject(new(a))
	fmt.Println(arg.B.C.C)
	fmt.Println(di.Graph())
}