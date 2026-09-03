package main

import "testing"

// 锁定 example/inject-method 的契约:方法按字母序执行,
// 单值参数取最后注册者,切片参数按注册顺序聚合全部。
func TestInjectWorker(t *testing.T) {
	w := injectWorker()

	if w.loggerMsg != "secondary" {
		t.Fatalf("logger got %q, want last registered provider's value", w.loggerMsg)
	}
	if len(w.tags) != 2 || w.tags[0].Error() != "primary" || w.tags[1].Error() != "secondary" {
		t.Fatalf("tags = %v, want [primary secondary]", w.tags)
	}
}
