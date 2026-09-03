package main

import (
	"strings"
	"testing"
)

// 锁定 example/lazy 的契约:注册阶段不执行 provider;
// 注入时同类型多 provider 按注册顺序全部执行,单值取最后注册者。
func TestRunLazy(t *testing.T) {
	order, injectedErr := runLazy()

	if injectedErr == nil || injectedErr.Error() != "ready" {
		t.Fatalf("injected error = %v, want provider C's error", injectedErr)
	}
	if got, want := strings.Join(order, ","), "provider-A,provider-B,provider-C"; got != want {
		t.Fatalf("execution order = %q, want %q", got, want)
	}
}
