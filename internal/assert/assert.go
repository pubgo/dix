package assert

import "fmt"

func AssertFn(ok bool, fn func() error) {
	if ok {
		panic(fn())
	}
}

func Assert(ok bool, err error) {
	if ok {
		panic(err)
	}
}

func Assertf(ok bool, msg string, args ...interface{}) {
	if ok {
		panic(fmt.Errorf(msg, args...))
	}
}
