package main

import (
	"strings"
	"testing"
)

// 锁定 example/cycle 的契约:注入前环检测生效,错误信息包含化简后的环路径。
//
// 注意:环路径的起点取决于依赖图的遍历顺序(Go map 随机),
// 同一个环可能以 A/B/C 任一节点开头轮换,但环成员与先后关系固定。
func TestDetectCycle(t *testing.T) {
	err := detectCycle()
	if err == nil {
		t.Fatal("cycle must be detected")
	}
	if !strings.Contains(err.Error(), "circular dependency") {
		t.Fatalf("error should report circular dependency, got: %v", err)
	}

	// 环 A -> B -> C -> A 的三种等价轮换,命中其一即通过。
	rotations := []string{
		"*main.ServiceA -> *main.ServiceB -> *main.ServiceC -> *main.ServiceA",
		"*main.ServiceB -> *main.ServiceC -> *main.ServiceA -> *main.ServiceB",
		"*main.ServiceC -> *main.ServiceA -> *main.ServiceB -> *main.ServiceC",
	}
	for _, want := range rotations {
		if strings.Contains(err.Error(), want) {
			return
		}
	}
	t.Fatalf("cycle path should be a rotation of A->B->C->A, got: %v", err)
}
