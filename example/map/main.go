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

	di.Provide(func() map[string]*xerr.Err {
		return map[string]*xerr.Err{
			"":      {Msg: "default"},
			"hello": {Msg: "hello"},
		}
	})

	di.Inject(func(err *xerr.Err, errs map[string]*xerr.Err) {
		fmt.Println(err.Msg)
		fmt.Println(errs)
	})

	type param struct {
		ErrMap map[string]*xerr.Err
	}
	fmt.Println(di.Inject(new(param)).ErrMap)
}
