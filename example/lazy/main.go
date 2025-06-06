package main

import (
	"fmt"

	"github.com/pubgo/dix/dixglobal"
	"github.com/pubgo/funk/errors"
	"github.com/pubgo/funk/recovery"
)

func main() {
	defer recovery.Exit()

	fmt.Println("=== Registering Providers (Lazy Loading Demo) ===")

	type handler struct{}

	// 注册第一个handler提供者
	dixglobal.Provide(func() *handler {
		fmt.Println("Creating handler instance 1")
		return new(handler)
	})

	// 注册第二个handler提供者（会覆盖第一个作为默认）
	dixglobal.Provide(func() *handler {
		fmt.Println("Creating handler instance 2")
		return new(handler)
	})

	// 注册依赖于handler的错误提供者
	dixglobal.Provide(func(_ *handler) *errors.Err {
		fmt.Println("Creating error instance (depends on handler)")
		return &errors.Err{Msg: "ok"}
	})

	fmt.Println("\n=== Dependency Graph ===")
	graph := dixglobal.Graph()
	fmt.Printf("Providers:\n%s\n", graph.Providers)

	fmt.Println("\n=== Function Injection (triggers lazy loading) ===")
	dixglobal.Inject(func(err *errors.Err) {
		fmt.Println("Injected error message:", err.Msg)
	})

	fmt.Println("\n=== Get API Demonstration ===")
	// 使用Get API获取实例
	h := dixglobal.Get[*handler]()
	fmt.Printf("Get handler: %p\n", h)

	err := dixglobal.Get[*errors.Err]()
	fmt.Println("Get error:", err.Msg)

	fmt.Println("\n=== Final Dependency Graph ===")
	finalGraph := dixglobal.Graph()
	fmt.Printf("Objects:\n%s\n", finalGraph.Objects)
}
