package main

import (
	"fmt"

	"github.com/pubgo/dix/dixglobal"
	"github.com/pubgo/funk/errors"
	"github.com/pubgo/funk/recovery"
)

type handler struct{}

func (h *handler) DixInjectA(err *errors.Err) {
	fmt.Println("Method A injected with error:", err.Msg)
}

func (h *handler) DixInjectD(p struct {
	Err *errors.Err
}) {
	fmt.Println("Method D injected with struct error:", p.Err.Msg)
}

func (h *handler) DixInjectC(errs []*errors.Err) {
	fmt.Printf("Method C injected with %d errors:\n", len(errs))
	for i, err := range errs {
		fmt.Printf("  [%d]: %s\n", i, err.Msg)
	}
}

func (h *handler) DixInjectB(err *errors.Err, errs []*errors.Err) {
	fmt.Printf("Method B injected with default error: %s\n", err.Msg)
	fmt.Printf("Method B injected with %d errors in list:\n", len(errs))
	for i, e := range errs {
		fmt.Printf("  [%d]: %s\n", i, e.Msg)
	}
}

func main() {
	defer recovery.Exit()

	fmt.Println("=== Registering Error Providers ===")

	// 注册第一个错误提供者
	dixglobal.Provide(func() *errors.Err {
		return &errors.Err{Msg: "<ok>"}
	})

	// 注册第二个错误提供者
	dixglobal.Provide(func() *errors.Err {
		return &errors.Err{Msg: "<ok 1>"}
	})

	fmt.Println("\n=== Dependency Graph ===")
	graph := dixglobal.Graph()
	fmt.Printf("Providers:\n%s\n", graph.Providers)

	fmt.Println("\n=== Method Injection ===")
	// 注入到handler实例，这会调用所有DixInject*方法
	h := &handler{}
	dixglobal.Inject(h)

	fmt.Println("\n=== 通过 Inject 获取依赖实例演示 ===")
	// 使用 Inject 方法获取依赖实例
	var defaultErr *errors.Err
	dixglobal.Inject(func(err *errors.Err) {
		defaultErr = err
	})
	fmt.Println("获取默认error:", defaultErr.Msg)

	// 批量获取多个依赖
	var singleErr *errors.Err
	var errorList []*errors.Err
	dixglobal.Inject(func(err *errors.Err, errs []*errors.Err) {
		singleErr = err
		errorList = errs
	})
	fmt.Printf("批量获取依赖: 单个error=%s, 列表长度=%d\n",
		singleErr.Msg, len(errorList))

	fmt.Println("\n=== Final Dependency Graph ===")
	finalGraph := dixglobal.Graph()
	fmt.Printf("Objects:\n%s\n", finalGraph.Objects)
}
