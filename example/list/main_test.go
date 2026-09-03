package main

import "testing"

// 锁定 example/list 的契约:列表按注册顺序聚合、切片元素顺序保持,
// 单值与列表注入互不干扰。
func TestBuildChain(t *testing.T) {
	latest, chain := buildChain()

	want := []string{"[auth] x", "[log] x", "[trace] x"}
	if len(chain) != len(want) {
		t.Fatalf("chain size = %d, want %d", len(chain), len(want))
	}
	for i, w := range want {
		if got := chain[i]("x"); got != w {
			t.Fatalf("chain[%d] = %q, want %q", i, got, w)
		}
	}
	if latest("x") != "[trace] x" {
		t.Fatalf("latest = %q, want the last registered provider", latest("x"))
	}
}
