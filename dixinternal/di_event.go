package dixinternal

import (
	"sort"
	"time"

	"github.com/pubgo/dix/v2/dixtrace"
)

// di_event.go 把 DI 过程中的点事件(point events)统一发布到 tracer 事件流,
// console(DIX_TRACE_DI)与 diag file(DIX_DIAG_FILE)是它的两个订阅者。
// 此前这两路各自直连埋点(双轨),现收敛为"一次发布、多处订阅"。

// emitDIEvent 发布一个 DI 点事件到容器 trace 通道(私有 tracer 或全局)。
func (dix *Dix) emitDIEvent(event string, args ...any) {
	dixtrace.EmitTo(dix.traceTracer, dixtrace.Event{
		Operation:   "di",
		Event:       event,
		ContainerID: dix.containerID,
		OccurredAt:  time.Now().UnixNano(),
		Attrs:       dixtrace.TraceToAttrs(args...),
	})
}

// consoleDISink 订阅点事件,在 DIX_TRACE_DI 开启时输出 `di_trace <event>` 日志。
type consoleDISink struct{}

func (consoleDISink) Write(e dixtrace.Event) {
	if e.Operation != "di" || !shouldTraceDependencyFlow() || logger == nil {
		return
	}
	logger.Info("di_trace "+e.Event, kvArgs(e.Attrs)...)
}

// diagTraceSink 订阅点事件,写入 DIX_DIAG_FILE 的 kind:trace 记录。
type diagTraceSink struct{}

func (diagTraceSink) Write(e dixtrace.Event) {
	if e.Operation != "di" {
		return
	}
	emitDiagFileTraceEvent(e.Event, kvArgs(e.Attrs)...)
}

// installDISinks 把 DI 点事件订阅者挂到容器私有 tracer 上。
func installDISinks(tr *dixtrace.Tracer) {
	if tr == nil {
		return
	}
	tr.AddSink(consoleDISink{})
	tr.AddSink(diagTraceSink{})
}

func init() {
	// 全局 tracer 默认挂载订阅者;console 是否输出由 DIX_TRACE_DI 逐条判定,
	// diag file 是否落盘由 emitDiagFileTraceEvent 内部按环境变量判定。
	dixtrace.AddDefaultSink(consoleDISink{})
	dixtrace.AddDefaultSink(diagTraceSink{})
}

// kvArgs 把属性 map 展开为 slog 风格的 key,value 交替参数(key 排序,输出稳定)。
func kvArgs(m map[string]any) []any {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys)*2)
	for _, k := range keys {
		out = append(out, k, m[k])
	}
	return out
}
