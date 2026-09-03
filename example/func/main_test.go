package main

import "testing"

// 锁定 example/func 的契约:单值取最后注册的 provider,列表按注册顺序聚合。
func TestBuildGreets(t *testing.T) {
	single, all := buildGreets()

	if single() != "world" {
		t.Fatalf("single = %q, want last registered provider's value", single())
	}
	if len(all) != 2 || all[0]() != "hello" || all[1]() != "world" {
		t.Fatalf("all = [%s %s], want registration order [hello world]", all[0](), all[1]())
	}
}
