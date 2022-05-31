package assert

import "fmt"

func Fn(ok bool, fn func() error) {
	if ok {
		panic(fn())
	}
}

func Assert(ok bool, err error) {
	if ok {
		panic(err)
	}
}

func Fmt(ok bool, msg string, args ...interface{}) {
	if ok {
		panic(fmt.Errorf(msg, args...))
	}
}
