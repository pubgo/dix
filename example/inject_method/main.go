// 【功能】方法注入:结构体的 DixInject 前缀方法自动接收依赖。
//
// 【原理】对结构体指针注入时,dix 先扫描 DixInject 前缀方法:
//   - 方法按名称字母序依次执行(本例 Logger 先于 Tags);
//   - 执行时机在字段注入之前:方法运行时结构体字段还是零值;
//   - 方法参数解析规则与函数注入一致:
//     单值参数取最后注册的 provider,切片参数按注册顺序聚合全部。
//
// 该语义由 dixinternal 的 TestPatternMethodInjection 锁定。
//
// 【运行】
//
//	cd example/inject_method && go run .
//
// 【预期输出】
//
//	logger dependency: secondary
//	tag dependencies: [0]=primary [1]=secondary
package main

import (
	"errors"
	"fmt"

	"github.com/pubgo/dix/v2"
)

type Worker struct{}

// DixInject 前缀方法在 Worker 创建后自动调用,依赖通过参数传入。
// 单值参数:多个 error provider 时取最后注册的 "secondary"。
func (w *Worker) DixInjectLogger(err error) {
	fmt.Println("logger dependency:", err.Error())
}

// 切片参数:聚合全部 error 产物,顺序与注册顺序一致。
func (w *Worker) DixInjectTags(errs []error) {
	fmt.Print("tag dependencies:")
	for i, e := range errs {
		fmt.Printf(" [%d]=%s", i, e.Error())
	}
	fmt.Println()
}

func main() {
	di := dix.New()

	dix.Provide(di, func() error { return errors.New("primary") })
	dix.Provide(di, func() error { return errors.New("secondary") })

	dix.Inject(di, &Worker{})
}
