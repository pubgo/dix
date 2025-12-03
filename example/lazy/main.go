package main

import (
	"errors"
	"fmt"

	"github.com/pubgo/dix/v2/dixglobal"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic: %v\n", r)
		}
	}()

	type handler struct{}
	dixglobal.Provide(func() *handler {
		fmt.Println("1")
		return new(handler)
	})

	dixglobal.Provide(func() *handler {
		fmt.Println("2")
		return new(handler)
	})

	dixglobal.Provide(func(_ *handler) error {
		return errors.New("ok")
	})

	dixglobal.Inject(func(err error) {
		fmt.Println(err.Error())
	})
}
