package main

import (
	"fmt"

	"github.com/pubgo/dix/dixglobal"
	"github.com/pubgo/funk/recovery"
)

func main() {
	defer recovery.Exit()

	type handler func() string

	// 注册多个handler提供者
	dixglobal.Provide(func() handler {
		return func() string {
			return "hello"
		}
	})

	dixglobal.Provide(func() handler {
		return func() string {
			return "world"
		}
	})

	type param struct {
		H    handler
		List []handler
	}

	fmt.Println("=== Dependency Graph ===")
	graph := dixglobal.Graph()
	fmt.Printf("Providers:\n%s\n", graph.Providers)
	fmt.Printf("Objects:\n%s\n", graph.Objects)

	// 使用结构体注入
	p := dixglobal.Inject(new(param))
	fmt.Println("struct inject result:", p.H())
	fmt.Printf("struct inject list length: %d\n", len(p.List))

	// 使用函数注入
	dixglobal.Inject(func(h handler, list []handler) {
		fmt.Println("func inject result:", h())
		fmt.Printf("func inject list length: %d\n", len(list))
		for i, fn := range list {
			fmt.Printf("  list[%d]: %s\n", i, fn())
		}
	})

	// 使用结构体参数注入
	dixglobal.Inject(func(p param) {
		fmt.Println("struct param inject result:", p.H())
		fmt.Printf("struct param inject list length: %d\n", len(p.List))
		for i, fn := range p.List {
			fmt.Printf("  param.List[%d]: %s\n", i, fn())
		}
	})

	fmt.Println("\n=== 通过 Inject 获取依赖实例演示 ===")
	// 使用 Inject 方法获取依赖实例
	var h handler
	dixglobal.Inject(func(handler handler) {
		h = handler
	})
	fmt.Println("获取handler:", h())

	fmt.Println("\n=== Final Dependency Graph ===")
	finalGraph := dixglobal.Graph()
	fmt.Printf("Providers:\n%s\n", finalGraph.Providers)
	fmt.Printf("Objects:\n%s\n", finalGraph.Objects)
}
