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

	dixglobal.Provide(func() map[string]*errors.Err {
		return map[string]*errors.Err{
			"":      {Msg: "default msg"},
			"hello": {Msg: "hello"},
		}
	})

	dixglobal.Provide(func() map[string]*errors.Err {
		return map[string]*errors.Err{
			"hello": {Msg: "hello1"},
		}
	})

	dixglobal.Inject(func(err *errors.Err, errs map[string]*errors.Err, errMapList map[string][]*errors.Err) {
		fmt.Println(err.Msg)
		fmt.Println(errs)
		fmt.Println(errMapList)
	})

	type param struct {
		ErrMap     map[string]*errors.Err
		ErrMapList map[string][]*errors.Err
	}
	fmt.Println(dixglobal.Inject(new(param)).ErrMap)
	fmt.Println(dixglobal.Inject(new(param)).ErrMapList)
}
