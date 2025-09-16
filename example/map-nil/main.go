package main

import (
	"fmt"

	"github.com/pubgo/dix/v2/dixglobal"
	"github.com/pubgo/funk/v2/errors"
	"github.com/pubgo/funk/v2/recovery"
)

func main() {
	defer recovery.Exit()

	defer func() {
		fmt.Println(dixglobal.Graph())
	}()

	dixglobal.Inject(func(errs map[string]*errors.Err) {
		fmt.Println(errs)
	})

	type param struct {
		ErrMap map[string]*errors.Err
	}
	fmt.Println(dixglobal.Inject(new(param)).ErrMap)
}
