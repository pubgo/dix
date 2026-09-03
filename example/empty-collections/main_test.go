package main

import (
	"testing"

	"github.com/pubgo/dix/v2"
)

// 锁定 example/empty-collections 的契约:
// 缺失集合依赖解析为非 nil 空集合;单值缺失恒报错;Reject 模式反转容错语义。
func TestInjectMissingCollections(t *testing.T) {
	errs, handlers := injectMissingCollections()

	if errs == nil || len(errs) != 0 {
		t.Fatalf("missing map must resolve to non-nil empty map, got %#v", errs)
	}
	if handlers == nil || len(handlers) != 0 {
		t.Fatalf("missing slice must resolve to non-nil empty slice, got %#v", handlers)
	}

	// 对照:单值依赖缺失必须报错,与集合容错开关无关。
	tolerant := dix.New(dix.WithValuesNull())
	type Single struct{ Dep *Handler }
	if err := tolerant.TryInject(&Single{}); err == nil {
		t.Fatal("missing single-value dependency must fail even with AllowValuesNull")
	}

	// 对照:WithRejectEmptyCollections 时缺失集合依赖直接报错。
	reject := dix.New(dix.WithRejectEmptyCollections())
	if err := reject.TryInject(func(m map[string]error, hs []Handler) {}); err == nil {
		t.Fatal("reject-empty-collections container must fail on missing collections")
	}
}
