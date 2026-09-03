package main

import (
	"testing"

	"github.com/pubgo/dix/v2"
)

// 锁定 example/list-nil 的契约:缺失的 slice 依赖解析为非 nil 空切片,
// 缺失的单值依赖(指针/接口/函数)仍然报错。
func TestInjectMissingHandlers(t *testing.T) {
	handlers := injectMissingHandlers()

	if handlers == nil {
		t.Fatal("missing slice dependency must resolve to a non-nil empty slice")
	}
	if len(handlers) != 0 {
		t.Fatalf("handlers = %v, want empty", handlers)
	}

	// 对照:单值依赖缺失必须报错,与集合容错开关无关。
	di := dix.New(dix.WithValuesNull())
	type Single struct{ Dep *Handler }
	if err := di.TryInject(&Single{}); err == nil {
		t.Fatal("missing single-value dependency must fail even with AllowValuesNull")
	}

	// 对照:WithRejectEmptyCollections 时缺失切片依赖直接报错。
	reject := dix.New(dix.WithRejectEmptyCollections())
	if err := reject.TryInject(func(hs []Handler) {}); err == nil {
		t.Fatal("reject-empty-collections container must fail on missing slice dependency")
	}
}
