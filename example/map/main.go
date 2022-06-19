package main

import (
	"fmt"

	"github.com/pubgo/dix"
	"github.com/pubgo/xerror"
)

func main() {
	dix.Provider(func() map[string]*xerror.Err {
		return map[string]*xerror.Err{
			"":      {Msg: "default"},
			"hello": {Msg: "hello"},
		}
	})

	dix.Inject(func(err *xerror.Err, errs map[string]*xerror.Err) {
		fmt.Println(err.Msg)
		fmt.Println(errs)
	})

	type param struct {
		ErrMap map[string]*xerror.Err
	}
	fmt.Println(dix.Inject(new(param)).ErrMap)
	fmt.Println(dix.Graph())
}
