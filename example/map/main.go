package main

import (
	"fmt"

	"github.com/pubgo/dix/dixglobal"
	"github.com/pubgo/funk/errors"
	"github.com/pubgo/funk/recovery"
)

func main() {
	defer recovery.Exit()

	defer func() {
		fmt.Println("\n=== Final Dependency Graph ===")
		graph := dixglobal.Graph()
		fmt.Printf("Providers:\n%s\n", graph.Providers)
		fmt.Printf("Objects:\n%s\n", graph.Objects)
	}()

	// 注册第一个错误映射提供者
	dixglobal.Provide(func() map[string]*errors.Err {
		return map[string]*errors.Err{
			"":      {Msg: "default msg"},
			"hello": {Msg: "hello"},
		}
	})

	// 注册第二个错误映射提供者
	dixglobal.Provide(func() map[string]*errors.Err {
		return map[string]*errors.Err{
			"hello": {Msg: "hello1"},
		}
	})

	fmt.Println("=== Function Injection ===")
	dixglobal.Inject(func(err *errors.Err, errs map[string]*errors.Err, errMapList map[string][]*errors.Err) {
		fmt.Println("Default error:", err.Msg)
		fmt.Println("Error map:")
		for key, e := range errs {
			fmt.Printf("  '%s': %s\n", key, e.Msg)
		}
		fmt.Println("Error map list:")
		for key, errList := range errMapList {
			fmt.Printf("  '%s': [", key)
			for i, e := range errList {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("'%s'", e.Msg)
			}
			fmt.Println("]")
		}
	})

	fmt.Println("\n=== Struct Injection ===")
	type param struct {
		ErrMap     map[string]*errors.Err
		ErrMapList map[string][]*errors.Err
	}

	p := dixglobal.Inject(new(param))
	fmt.Println("Struct ErrMap:")
	for key, err := range p.ErrMap {
		fmt.Printf("  '%s': %s\n", key, err.Msg)
	}
	fmt.Println("Struct ErrMapList:")
	for key, errList := range p.ErrMapList {
		fmt.Printf("  '%s': [", key)
		for i, e := range errList {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("'%s'", e.Msg)
		}
		fmt.Println("]")
	}

	fmt.Println("\n=== Get API ===")
	// 使用Get API获取实例
	defaultErr := dixglobal.Get[*errors.Err]()
	fmt.Println("Get default error:", defaultErr.Msg)

	errMap := dixglobal.Get[map[string]*errors.Err]()
	fmt.Println("Get error map:")
	for key, err := range errMap {
		fmt.Printf("  '%s': %s\n", key, err.Msg)
	}
}
