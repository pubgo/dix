package main

import (
	"fmt"

	"github.com/pubgo/dix"
	"github.com/pubgo/funk"
)

func main() {
	defer funk.RecoverAndExit()
	defer func() {
		fmt.Println(dix.Graph())
	}()
	dix.Provider(func() map[string]*funk.Err {
		return map[string]*funk.Err{
			"":      {Msg: "default"},
			"hello": {Msg: "hello"},
		}
	})

	dix.Inject(func(err *funk.Err, errs map[string]*funk.Err) {
		fmt.Println(err.Msg)
		fmt.Println(errs)
	})

	type param struct {
		ErrMap map[string]*funk.Err
	}
	fmt.Println(dix.Inject(new(param)).ErrMap)
}
