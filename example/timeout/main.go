// 【功能】provider 超时控制:执行超时的 provider 不重试,慢 provider 告警。
//
// 【原理】WithProviderTimeout 限制单个 provider 的执行时长:
//   - 超时后本次注入失败,provider 被标记为"已超时";
//   - 已超时的 provider 不会被执行第二次(孤儿调用可能仍在跑,重试会重复副作用);
//   - 后续注入快速失败,错误提示调大超时或重建容器;
//   - WithSlowProviderThreshold:执行超过阈值(未超时)时输出慢告警日志。
//
// 该语义由 dixinternal 的 TestProviderTimeout/TestProviderTimeoutNotRetried
// 与本目录 main_test.go 的 TestBuildWithTimeout 锁定。
//
// 【运行】
//
//	cd example/timeout && go run .
//
// 【预期输出】(另有 dix 结构化超时错误日志,时间戳省略)
//
//	first inject: provider execution timeout after 100ms
//	second inject: provider ... timed out previously and will not be re-executed
package main

import (
	"fmt"
	"time"

	"github.com/pubgo/dix/v2"
)

type Remote struct{ Ready bool }

func main() {
	firstErr, secondErr := buildWithTimeout()

	fmt.Println("first inject:", firstErr)
	fmt.Println("second inject:", secondErr)
}

// buildWithTimeout 注册一个 200ms 的慢 provider,容器超时 100ms:
// 返回首次注入错误(超时)与再次注入错误(已标记超时,不再执行)。
func buildWithTimeout() (error, error) {
	di := dix.New(
		dix.WithProviderTimeout(100*time.Millisecond),
		dix.WithSlowProviderThreshold(50*time.Millisecond),
	)

	dix.Provide(di, func() *Remote {
		time.Sleep(200 * time.Millisecond) // 模拟慢外部依赖
		return &Remote{Ready: true}
	})

	firstErr := di.TryInject(func(r *Remote) {})
	secondErr := di.TryInject(func(r *Remote) {})
	return firstErr, secondErr
}
