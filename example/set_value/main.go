package main

import (
	"fmt"

	"github.com/pubgo/dix/di"
)

type V interface {
	Hello()
}

type vs struct {
}

func (*vs) Hello() {
	fmt.Println("hello")
}

func main() {
	fmt.Println(di.Graph())
	var vv = new(vs)
	fmt.Printf("%p\n", vv)
	di.SetValue(vv, (*V)(nil))
	di.SetValue([]*vs{vv}, (*V)(nil))
	di.SetValue(map[string][]*vs{"group": {vv}}, (*V)(nil))
	di.Inject(func(a1 *vs, a2 map[string][]*vs, a3 map[string][]V) {
		fmt.Printf("%p\n", a1)
		fmt.Println(a2)
		fmt.Println(a3)
	})
	fmt.Println(di.Graph())
}
