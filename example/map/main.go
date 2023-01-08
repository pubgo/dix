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

	di.Provide(func() map[string]*errors.Err {
		return map[string]*errors.Err{
			"":      {Msg: "default"},
			"hello": {Msg: "hello"},
		}
	})

	di.Inject(func(err *errors.Err, errs map[string]*errors.Err) {
		fmt.Println(err.Msg)
		fmt.Println(errs)
	})

	type param struct {
		ErrMap map[string]*errors.Err
	}
	fmt.Println(di.Inject(new(param)).ErrMap)
}
