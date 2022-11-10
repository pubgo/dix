package main

import (
	"fmt"

	"github.com/pubgo/dix/di"
	"github.com/pubgo/funk/recovery"
	"github.com/pubgo/funk/xerr"
)

func main() {
	defer recovery.Exit()

	defer func() {
		fmt.Println(di.Graph())
	}()

	di.Inject(func(errs map[string]*xerr.Err) {
		fmt.Println(errs)
	})

	type param struct {
		ErrMap map[string]*xerr.Err
	}
	fmt.Println(di.Inject(new(param)).ErrMap)
}
