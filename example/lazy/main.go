// 【功能】惰性求值:provider 只有在其产物被需要时才执行。
//
// 【原理】注册阶段不执行任何 provider;Inject 时沿依赖链按需解析:
//   - 本次注入用不到的 provider 不会执行;
//   - 同一输出类型的多个 provider 会全部执行(按注册顺序),
//     单值注入取最后注册者的产物(本例中 C 拿到的是 B 的 *Service);
//   - 产物缓存在容器内,后续 Inject 不会重复执行 provider。
//
// 该语义由 dixinternal 的 TestPatternLazyResolution 锁定。
//
// 【运行】
//
//	cd example/lazy && go run .
//
// 【预期输出】
//
//	provider A executed
//	provider B executed
//	provider C executed
//	inject got error: ready
//	execution order: [provider-A provider-B provider-C]
package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
)

type Service struct {
	Name string
}

func main() {
	di := dix.New()
	order := make([]string, 0, 3)

	dix.Provide(di, func() *Service {
		order = append(order, "provider-A")
		fmt.Println("provider A executed")
		return &Service{Name: "A"}
	})

	dix.Provide(di, func() *Service {
		order = append(order, "provider-B")
		fmt.Println("provider B executed")
		return &Service{Name: "B"}
	})

	// C 依赖 *Service(单值):解析时 A、B 都会执行,C 拿到最后注册者 B 的产物。
	dix.Provide(di, func(_ *Service) error {
		order = append(order, "provider-C")
		fmt.Println("provider C executed")
		return fmt.Errorf("ready")
	})

	dix.Inject(di, func(err error) {
		fmt.Println("inject got error:", err)
	})

	fmt.Println("execution order:", order)
}
