package main

import (
	"testing"

	"github.com/pubgo/dix/v2"
)

// 锁定 example/map-nil 的契约:缺失的 map 依赖解析为非 nil 空 map,
// 缺失的单值依赖仍然报错。
func TestInjectMissingErrors(t *testing.T) {
	errs := injectMissingErrors()

	if errs == nil {
		t.Fatal("missing map dependency must resolve to a non-nil empty map")
	}
	if len(errs) != 0 {
		t.Fatalf("errors = %v, want empty", errs)
	}

	// 对照:WithRejectEmptyCollections 时缺失 map 依赖直接报错。
	reject := dix.New(dix.WithRejectEmptyCollections())
	if err := reject.TryInject(func(m map[string]error) {}); err == nil {
		t.Fatal("reject-empty-collections container must fail on missing map dependency")
	}
}
