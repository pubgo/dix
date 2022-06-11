package main

import (
	"fmt"

	"github.com/pubgo/dix"
	"github.com/pubgo/xerror"
)

func main() {
	dix.Register(func() map[string]*xerror.Err {
		return map[string]*xerror.Err{
			"":      {Msg: "default"},
			"hello": {Msg: "hello"},
		}
	})

	dix.Register(func(err *xerror.Err, errs map[string]*xerror.Err) {
		fmt.Println(err.Msg)
		fmt.Println(errs)
	})
	dix.Invoke()
	fmt.Println(dix.Graph())
}