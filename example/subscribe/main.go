package main

import (
	"fmt"
	"reflect"

	"github.com/pubgo/dix"
)

func main() {
	dix.Sub(func(ns string, val reflect.Value) {
		fmt.Printf("%s: %#v\n", ns, val.Interface())
	})

	type handler func() string
	dix.Provider(func() handler {
		return func() string {
			return "hello"
		}
	})

	dix.Provider(func() handler {
		return func() string {
			return "world"
		}
	})

	dix.Sub(func(ns string, val reflect.Value) {
		fmt.Printf("%s: %#v\n", ns, val.Interface())
	})

	dix.Inject(func(h handler, hh []handler) {

	})
}
