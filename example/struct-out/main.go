package main

import (
	"fmt"

	"github.com/pubgo/dix/dixglobal"
	"github.com/pubgo/funk/recovery"
)

type Inline struct {
	M *C1
}

type D struct {
	M C
}

type (
	C  interface{}
	C1 struct {
		Name string
	}
)

type Conf struct {
	Data string
	Inline
	A  *A
	B  *B
	C  C
	D  *D
	D1 *D
	D2 map[string]*D
	D3 []*D
	D4 map[string][]*D
}

type A struct {
	Hello string
}

type B struct {
	Hello string
}

func main() {
	defer recovery.Exit(func(err error) error {
		fmt.Println("\n=== Error Recovery - Final Graph ===")
		graph := dixglobal.Graph()
		fmt.Printf("Providers:\n%s\n", graph.Providers)
		return err
	})

	fmt.Println("=== Registering Complex Struct Provider ===")

	// 注册一个复杂的结构体提供者，包含多种类型的字段
	dixglobal.Provide(func() Conf {
		return Conf{
			Data: "configuration data",
			A:    &A{Hello: "hello-a"},
			B:    &B{Hello: "hello-b"},
			C:    "hello interface",
			D: &D{
				M: "hello D",
			},
			D1: &D{
				M: "hello D1",
			},
			D2: map[string]*D{
				"default1": {
					M: "hello D2 default",
				},
				"custom": {
					M: "hello D2 custom",
				},
			},
			D3: []*D{
				{
					M: "hello D3 item 1",
				},
				{
					M: "hello D3 item 2",
				},
			},
			D4: map[string][]*D{
				"default4": {
					{
						M: "hello D4 default item 1",
					},
					{
						M: "hello D4 default item 2",
					},
				},
			},
			Inline: Inline{M: &C1{Name: "inline C1"}},
		}
	})

	fmt.Println("\n=== Dependency Graph After Registration ===")
	graph := dixglobal.Graph()
	fmt.Printf("Providers:\n%s\n", graph.Providers)

	fmt.Println("\n=== Function Injection ===")
	dixglobal.Inject(func(a *A, b *B, cc []C, c1 *C1, c2 []*C1, d *D, dd []*D, dm map[string]*D, d5 map[string][]*D) {
		fmt.Println("Injected A:", a.Hello)
		fmt.Println("Injected B:", b.Hello)
		fmt.Printf("Injected C list length: %d\n", len(cc))
		for i, c := range cc {
			fmt.Printf("  cc[%d]: %v\n", i, c)
		}
		fmt.Println("Injected C1:", c1.Name)
		fmt.Printf("Injected C1 list length: %d\n", len(c2))
		for i, c := range c2 {
			fmt.Printf("  c2[%d]: %s\n", i, c.Name)
		}
		fmt.Println("Injected D:", d.M)
		fmt.Printf("Injected D list length: %d\n", len(dd))
		for i, d := range dd {
			fmt.Printf("  dd[%d]: %v\n", i, d.M)
		}
		fmt.Printf("Injected D map length: %d\n", len(dm))
		for key, d := range dm {
			fmt.Printf("  dm['%s']: %v\n", key, d.M)
		}
		fmt.Printf("Injected D map list length: %d\n", len(d5))
		for key, dList := range d5 {
			fmt.Printf("  d5['%s']: [", key)
			for i, d := range dList {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("%v", d.M)
			}
			fmt.Println("]")
		}
	})

	fmt.Println("\n=== 通过 Inject 获取依赖实例演示 ===")
	// 使用 Inject 方法获取依赖实例
	var a *A
	var b *B
	var c1 *C1
	dixglobal.Inject(func(aInst *A, bInst *B, c1Inst *C1) {
		a = aInst
		b = bInst
		c1 = c1Inst
	})
	fmt.Println("A:", a.Hello)
	fmt.Println("B:", b.Hello)
	fmt.Println("C1:", c1.Name)

	fmt.Println("\n=== Final Dependency Graph ===")
	finalGraph := dixglobal.Graph()
	fmt.Printf("Providers:\n%s\n", finalGraph.Providers)
	fmt.Printf("Objects:\n%s\n", finalGraph.Objects)
}
