package main

import (
	"fmt"

	"github.com/pubgo/funk/recovery"
	"github.com/pubgo/funk/xerr"

	"github.com/pubgo/dix"
)

func main() {
	defer recovery.Exit()

	defer func() {
		fmt.Println(dix.Graph())
	}()
	dix.Provider(func() map[string]*xerr.Err {
		return map[string]*xerr.Err{
			"":      {Msg: "default"},
			"hello": {Msg: "hello"},
		}
	})

	dix.Inject(func(err *xerr.Err, errs map[string]*xerr.Err) {
		fmt.Println(err.Msg)
		fmt.Println(errs)
	})

	type param struct {
		ErrMap map[string]*xerr.Err
	}
	fmt.Println(dix.Inject(new(param)).ErrMap)
}
