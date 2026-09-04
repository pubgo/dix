package main

import (
	"context"
	"testing"

	"github.com/pubgo/dix/v2/dixtrace"
)

// 锁定 example/context-inject 的契约:注入携带的 ctx 决定 trace 归属——
// 本次注入产生的全部事件共享业务 span 的 trace_id。
func TestInjectWithContext(t *testing.T) {
	traceID, events := injectWithContext(context.Background())

	if traceID == "" {
		t.Fatal("business span should have a trace id")
	}
	if events == 0 {
		t.Fatalf("inject events should be recorded under trace %q", traceID)
	}

	// 调用树:根为 inject.cycle_check / inject 链,可按 trace_id 完整还原。
	tree := dixtrace.QueryTree(traceID)
	if len(tree.Roots) == 0 {
		t.Fatalf("trace tree should have roots for %q", traceID)
	}
	if tree.Roots[0].Event.TraceID != traceID {
		t.Fatalf("tree root trace id = %q, want %q", tree.Roots[0].Event.TraceID, traceID)
	}
}
