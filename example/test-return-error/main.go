// 【功能】错误处理:用 TryProvide/TryInject 返回 error,而不是 panic。
//
// 【原理】三类错误路径,根因都保留在错误链上(errors.Is 可判定):
//   - provider 正常返回:注入成功;
//   - provider 返回非 nil error:注入失败,错误包装为
//     "provider execution failed: ...: 根因";失败的 provider 不缓存,
//     下次 Inject 会重新执行;
//   - 注入函数自身返回 error:错误包装为
//     "injected function returned error: 根因"。
//
// 该语义由 dixinternal 的 TestPatternProviderErrorPropagation 锁定。
//
// 【运行】
//
//	cd example/test-return-error && go run .
//
// 【预期输出】(另有多行 dix 结构化错误日志,时间戳省略)
//
//	inject ok
//	provider error: provider execution failed: main.runProviderError.func1: provider_err
//	inject error: injected function returned error: inject_err
package main

import (
	"fmt"
	"log"

	"github.com/pubgo/dix/v2"
)

func main() {
	runSuccess()
	runProviderError()
	runInjectError()
}

// 正常路径:TryProvide + TryInject 全部成功。
func runSuccess() {
	di := dix.New()
	if err := dix.TryProvide(di, func() (*log.Logger, error) {
		return log.Default(), nil
	}); err != nil {
		log.Fatalf("provide failed: %v", err)
	}

	if err := dix.TryInject(di, func(l *log.Logger) error {
		l.Println("inject ok")
		return nil
	}); err != nil {
		log.Fatalf("inject failed: %v", err)
	}
}

// provider 返回 error:注入失败,根因 provider_err 保留在错误链上。
func runProviderError() {
	di := dix.New()
	_ = dix.TryProvide(di, func() (*log.Logger, error) {
		return nil, fmt.Errorf("provider_err")
	})

	err := dix.TryInject(di, func(l *log.Logger) error {
		return nil
	})
	fmt.Println("provider error:", err)
}

// 注入函数自身返回 error:同样通过 TryInject 的返回值上抛。
func runInjectError() {
	di := dix.New()
	_ = dix.TryProvide(di, func() *log.Logger { return log.Default() })

	err := dix.TryInject(di, func(l *log.Logger) error {
		return fmt.Errorf("inject_err")
	})
	fmt.Println("inject error:", err)
}
