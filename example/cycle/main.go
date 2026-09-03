// 【功能】循环依赖检测:dix 在注入前拒绝 A -> B -> C -> A 的依赖环。
//
// 【原理】Provide 阶段构建依赖图并缓存;Inject 阶段先做环检测,
// 命中环则直接报错(错误信息包含化简后的环路径),
// 不会像运行时递归那样栈溢出。
//
// 该语义由 dixinternal 的 TestCycleDetection
// 与本目录 main_test.go 的 TestDetectCycle 锁定。
//
// 【运行】
//
//	cd example/cycle && go run .
//
// 【预期输出】(日志时间戳省略;环路径起点可能轮换,但环成员与顺序关系固定)
//
//	cycle detected: circular dependency: *main.ServiceA -> *main.ServiceB -> *main.ServiceC -> *main.ServiceA
package main

import (
	"fmt"
	"log"

	"github.com/pubgo/dix/v2"
)

type (
	ServiceA struct{}
	ServiceB struct{}
	ServiceC struct{}
)

func main() {
	if err := detectCycle(); err != nil {
		log.Println("cycle detected:", err)
		return
	}

	fmt.Println("unexpected: cycle was not detected")
}

// detectCycle 注册 A -> B -> C -> A 的循环依赖并尝试注入,
// 返回环检测错误(nil 表示未检出,不应发生)。
func detectCycle() error {
	di := dix.New()

	dix.Provide(di, func(*ServiceB) *ServiceA { return &ServiceA{} })
	dix.Provide(di, func(*ServiceC) *ServiceB { return &ServiceB{} })
	dix.Provide(di, func(*ServiceA) *ServiceC { return &ServiceC{} })

	// 生产路径建议用 TryInject:出错返回 error 而不是 panic。
	return dix.TryInject(di, func(*ServiceC) {})
}
