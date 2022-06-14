package main

import (
	"fmt"

	"github.com/pubgo/dix"
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
	dix.Register(func() *c {
		return &c{C: "hello"}
	})

	arg := dix.Inject(new(a)).(*a)
	fmt.Println(arg.B.C.C)
	fmt.Println(dix.Graph())
}
