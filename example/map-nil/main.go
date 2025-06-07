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

	fmt.Println("=== Map Nil Demo (No Providers Registered) ===")

	fmt.Println("\n=== Function Injection with Empty Map ===")
	dixglobal.Inject(func(errs map[string]*errors.Err) {
		fmt.Printf("Function injection - error map size: %d\n", len(errs))
		if len(errs) == 0 {
			fmt.Println("No errors available (empty map)")
		} else {
			for key, err := range errs {
				fmt.Printf("  '%s': %s\n", key, err.Msg)
			}
		}
	})

	fmt.Println("\n=== Struct Injection with Empty Map ===")
	type param struct {
		ErrMap map[string]*errors.Err
	}

	p := dixglobal.Inject(new(param))
	fmt.Printf("Struct injection - error map size: %d\n", len(p.ErrMap))
	if len(p.ErrMap) == 0 {
		fmt.Println("Struct field ErrMap is empty")
	} else {
		for key, err := range p.ErrMap {
			fmt.Printf("  '%s': %s\n", key, err.Msg)
		}
	}

	fmt.Println("\n=== 通过 Inject 获取依赖实例演示 ===")
	// 使用 Inject 方法获取空映射
	var errMap map[string]*errors.Err
	dixglobal.Inject(func(em map[string]*errors.Err) {
		errMap = em
	})
	fmt.Printf("error映射大小: %d\n", len(errMap))
	if len(errMap) == 0 {
		fmt.Println("返回空映射")
	}

	fmt.Println("\n=== Dependency Graph (should show no providers) ===")
	graph := dixglobal.Graph()
	fmt.Printf("Providers:\n%s\n", graph.Providers)
}
