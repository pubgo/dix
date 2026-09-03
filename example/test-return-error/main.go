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
// 该语义由 dixinternal 的 TestPatternProviderErrorPropagation
// 与本目录 main_test.go 的 TestRunErrorScenarios 锁定。
//
// 【运行】
//
//	cd example/test-return-error && go run .
//
// 【预期输出】(另有多行 dix 结构化错误日志与 "inject ok",时间戳省略)
//
//	provider error: provider execution failed: main.runProviderError.func1: provider_err
//	inject error: injected function returned error: inject_err
package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/pubgo/dix/v2"
)

func main() {
	if err := runSuccess(); err != nil {
		log.Fatalf("run failed: %v", err)
	}

	fmt.Println("provider error:", runProviderError())
	fmt.Println("inject error:", runInjectError())
}

// 正常路径:TryProvide + TryInject 全部成功,返回 nil。
func runSuccess() error {
	di := dix.New()
	if err := dix.TryProvide(di, func() (*log.Logger, error) {
		return log.Default(), nil
	}); err != nil {
		return err
	}

	return dix.TryInject(di, func(l *log.Logger) error {
		l.Println("inject ok")
		return nil
	})
}

// provider 返回 error:注入失败,根因 provider_err 保留在错误链上。
func runProviderError() error {
	di := dix.New()
	_ = dix.TryProvide(di, func() (*log.Logger, error) {
		return nil, errors.New("provider_err")
	})

	return dix.TryInject(di, func(l *log.Logger) error {
		return nil
	})
}

// errInjectSentinel 是注入函数返回的根因错误,
// 包级定义便于测试用 errors.Is 验证错误链。
var errInjectSentinel = errors.New("inject_err")

// 注入函数自身返回 error:同样通过 TryInject 的返回值上抛。
func runInjectError() error {
	di := dix.New()
	_ = dix.TryProvide(di, func() *log.Logger { return log.Default() })

	return dix.TryInject(di, func(l *log.Logger) error {
		return errInjectSentinel
	})
}
