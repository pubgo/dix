package main

import (
	"fmt"

	"github.com/pubgo/dix/dixglobal"
	"github.com/pubgo/dix/dixinternal"
	"github.com/pubgo/funk/assert"
	"github.com/pubgo/funk/recovery"
)

type a struct {
	b
	B b
}

type b struct {
	C *c
}

type c struct {
	C string
}

func main() {
	defer recovery.Exit()

	// 注册c的提供者
	dixglobal.Provide(func() *c {
		return &c{C: "hello"}
	})

	// 注入结构体a
	arg := dixglobal.Inject(new(a))
	assert.If(arg.C.C != "hello", "not match")
	fmt.Println("Embedded field C:", arg.C.C)
	fmt.Println("Field B.C:", arg.B.C.C)

	fmt.Println("\n=== Dependency Graph ===")
	graph := dixglobal.Graph()
	fmt.Printf("Providers:\n%s\n", graph.Providers)

	// 注册a2的提供者，展示复杂依赖
	dixglobal.Provide(func(a1 a1, container dixinternal.Container, containerMap map[string][]dixinternal.Container) *a2 {
		fmt.Printf("Received container map with %d entries\n", len(containerMap))
		for key, containers := range containerMap {
			fmt.Printf("  Key '%s': %d containers\n", key, len(containers))
		}
		return &a2{Hello: "a2", container: container}
	})

	// 使用函数注入a2
	dixglobal.Inject(func(a *a2) {
		fmt.Println("a2.Hello:", a.Hello)
		fmt.Println("a2.container options:", a.container.Option())
	})

	// 使用Get API获取实例
	c1 := dixglobal.Get[*c]()
	fmt.Println("Get API result:", c1.C)

	a2Instance := dixglobal.Get[*a2]()
	fmt.Println("Get a2 instance:", a2Instance.Hello)

	fmt.Println("\n=== Final Dependency Graph ===")
	finalGraph := dixglobal.Graph()
	fmt.Printf("Providers:\n%s\n", finalGraph.Providers)
}

type a1 struct {
	b
}

type a2 struct {
	Hello     string
	container dixinternal.Container
}
