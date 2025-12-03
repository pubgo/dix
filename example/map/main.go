package main

import (
	"errors"
	"fmt"

	"github.com/pubgo/dix/v2/dixglobal"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic: %v\n", r)
		}
	}()

	defer func() {
		fmt.Println(dixglobal.Graph())
	}()

	dixglobal.Provide(func() map[string]error {
		return map[string]error{
			"":      errors.New("default msg"),
			"hello": errors.New("hello"),
		}
	})

	dixglobal.Provide(func() map[string]error {
		return map[string]error{
			"hello": errors.New("hello1"),
		}
	})

	dixglobal.Inject(func(err error, errs map[string]error, errMapList map[string][]error) {
		fmt.Println(err.Error())
		fmt.Println(errs)
		fmt.Println(errMapList)
	})

	type param struct {
		ErrMap     map[string]error
		ErrMapList map[string][]error
	}
	fmt.Println(dixglobal.Inject(new(param)).ErrMap)
	fmt.Println(dixglobal.Inject(new(param)).ErrMapList)
}
