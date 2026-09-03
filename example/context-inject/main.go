// 【功能】Context 注入:InjectContext/TryInjectContext 把调用方 trace 上下文传给 dix。
//
// 【原理】注入携带的 ctx 不参与依赖解析,而是用于 trace 链路:
//   - ctx 里已有的 dixtrace span 会成为本次 Inject/resolve/provider 各级 span 的父节点;
//   - 全链路事件共享同一个 trace_id,可在 dixhttp 的 /api/trace 中按 trace 查询;
//   - TryInjectContext 返回 error,不 panic。
//
// 该语义由 dixinternal 的 TestTraceSpanChainInjectResolveProvider
// 与本目录 main_test.go 的 TestInjectWithContext 锁定。
//
// 【运行】
//
//	cd example/context-inject && go run .
//
// 【预期输出】
//
//	service: svc
//	trace events under demo-trace: 8
package main

import (
	"context"
	"fmt"

	"github.com/pubgo/dix/v2"
	"github.com/pubgo/dix/v2/dixtrace"
)

type Service struct{ Name string }

func main() {
	_, events := injectWithContext(context.Background())

	fmt.Println("service: svc")
	fmt.Printf("trace events under demo-trace: %d\n", events)
}

// injectWithContext 从业务 span 开始注入:dix 的 inject/resolve/provider
// 各级 span 都会挂到该 span 的 trace 下,事件可按 trace_id 查询。
func injectWithContext(ctx context.Context) (string, int) {
	di := dix.New()
	dix.Provide(di, func() *Service { return &Service{Name: "svc"} })

	// 业务侧开启一个 span,模拟一次请求/启动链路的入口。
	ctx, span := dixtrace.BeginSpanCtx(ctx, "demo-trace", "example")
	if err := dix.TryInjectContext(ctx, di, func(s *Service) {}); err != nil {
		panic(err)
	}
	span.End(nil)

	traceID, _, _ := span.IDs()
	result := dixtrace.QueryEvents(dixtrace.Query{TraceID: traceID})
	return traceID, len(result.Records)
}
