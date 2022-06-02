package assert

import (
	"errors"
	"fmt"
)

func Func(ok bool, fn func() error) {
	if ok {
		panic(fn())
	}
}

func Err(ok bool, err error) {
	if ok {
		panic(err)
	}
}

func Msg(ok bool, msg string, args ...interface{}) {
	if ok {
		panic(fmt.Errorf(msg, args...))
	}
}

func Recovery(fn func(err error)) {
	var err = recover()
	switch err.(type) {
	case nil:
		return
	case error:
		fn(err.(error))
	case string:
		fn(errors.New(err.(string)))
	default:
		fn(fmt.Errorf("%#v", err))
	}
}
