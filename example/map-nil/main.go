package main

import (
	"fmt"

	"github.com/pubgo/dix/v2/dixglobal"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic: %v\n", r)
		}
	}()

	dixglobal.Inject(func(errs map[string]error) {
		fmt.Println(errs)
	})

	type param struct {
		ErrMap map[string]error
	}
	fmt.Println(dixglobal.Inject(new(param)).ErrMap)
}
