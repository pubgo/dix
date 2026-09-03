package main

import (
	"strings"
	"testing"
)

// 锁定 example/cycle 的契约:注入前环检测生效,错误信息包含化简后的环路径。
// 环路径是确定性的:起点取环成员中类型名字典序最小者(dixinternal 的
// TestDetectCycleDeterministicOrder 锁定该语义)。
func TestDetectCycle(t *testing.T) {
	err := detectCycle()
	if err == nil {
		t.Fatal("cycle must be detected")
	}

	want := "circular dependency: *main.ServiceA -> *main.ServiceB -> *main.ServiceC -> *main.ServiceA"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("cycle error should contain deterministic path %q, got: %v", want, err)
	}
}
