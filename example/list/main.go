package main

import (
	"fmt"

	"github.com/pubgo/dix/dixglobal"
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

	type handler func() string
	type handlers []handler

	fmt.Println("=== Registering Providers ===")

	// 注册handlers切片提供者
	dixglobal.Provide(func() handlers {
		return handlers{
			func() string {
				return "hello"
			},
		}
	})

	dixglobal.Provide(func() handlers {
		return handlers{
			func() string {
				return "world"
			},
		}
	})

	// 注册单个handler提供者
	dixglobal.Provide(func() handler {
		return func() string {
			return "world next"
		}
	})

	fmt.Println("\n=== Initial Dependency Graph ===")
	graph := dixglobal.Graph()
	fmt.Printf("Providers:\n%s\n", graph.Providers)

	fmt.Println("\n=== Function Injection ===")
	dixglobal.Inject(func(handlers handlers, h handler) {
		// h为默认的，最后一个注册的
		fmt.Println("Default handler result:", h())
		fmt.Printf("Handlers list length: %d\n", len(handlers))
		for i, fn := range handlers {
			fmt.Printf("  handlers[%d]: %s\n", i, fn())
		}
	})

	fmt.Println("\n=== Struct Injection ===")
	type param struct {
		H handlers
		M map[string]handler
	}

	p := dixglobal.Inject(new(param))
	fmt.Printf("Struct handlers length: %d\n", len(p.H))
	for i, fn := range p.H {
		fmt.Printf("  struct.H[%d]: %s\n", i, fn())
	}

	fmt.Printf("Struct map length: %d\n", len(p.M))
	for key, fn := range p.M {
		fmt.Printf("  struct.M['%s']: %s\n", key, fn())
	}

	fmt.Println("\n=== Struct Parameter Injection ===")
	dixglobal.Inject(func(p param) {
		fmt.Printf("Param handlers length: %d\n", len(p.H))
		for i, fn := range p.H {
			fmt.Printf("  param.H[%d]: %s\n", i, fn())
		}

		fmt.Printf("Param map length: %d\n", len(p.M))
		for key, fn := range p.M {
			fmt.Printf("  param.M['%s']: %s\n", key, fn())
		}
	})

	fmt.Println("\n=== Get API ===")
	// 使用Get API获取实例
	defaultHandler := dixglobal.Get[handler]()
	fmt.Println("Get default handler:", defaultHandler())

	handlersList := dixglobal.Get[handlers]()
	fmt.Printf("Get handlers list length: %d\n", len(handlersList))
	for i, fn := range handlersList {
		fmt.Printf("  Get handlers[%d]: %s\n", i, fn())
	}
}
