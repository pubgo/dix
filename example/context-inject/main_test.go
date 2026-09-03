package main

import (
	"context"
	"testing"
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
}
