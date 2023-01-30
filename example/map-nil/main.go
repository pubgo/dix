package main

import (
	"fmt"

	"github.com/pubgo/dix/di"
	"github.com/pubgo/funk/errors"
	"github.com/pubgo/funk/recovery"
)

func main() {
	defer recovery.Exit()

	defer func() {
		fmt.Println(di.Graph())
	}()

	di.Inject(func(errs map[string]*errors.Err) {
		fmt.Println(errs)
	})

	type param struct {
		ErrMap map[string]*errors.Err
	}
	fmt.Println(di.Inject(new(param)).ErrMap)
}
