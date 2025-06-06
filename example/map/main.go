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
		fmt.Println(diglobal.Graph())
	}()

	diglobal.Provide(func() map[string]*errors.Err {
		return map[string]*errors.Err{
			"":      {Msg: "default msg"},
			"hello": {Msg: "hello"},
		}
	})

	diglobal.Provide(func() map[string]*errors.Err {
		return map[string]*errors.Err{
			"hello": {Msg: "hello1"},
		}
	})

	diglobal.Inject(func(err *errors.Err, errs map[string]*errors.Err, errMapList map[string][]*errors.Err) {
		fmt.Println(err.Msg)
		fmt.Println(errs)
		fmt.Println(errMapList)
	})

	type param struct {
		ErrMap     map[string]*errors.Err
		ErrMapList map[string][]*errors.Err
	}
	fmt.Println(diglobal.Inject(new(param)).ErrMap)
	fmt.Println(diglobal.Inject(new(param)).ErrMapList)
}
