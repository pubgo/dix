package main

import (
	"errors"
	"strings"
	"testing"
)

// 锁定 example/error-handling 的契约:
// 正常路径返回 nil;两类失败路径的错误链都保留原始根因。
func TestRunErrorScenarios(t *testing.T) {
	if err := runSuccess(); err != nil {
		t.Fatalf("runSuccess = %v, want nil", err)
	}

	err := runProviderError()
	if !strings.Contains(err.Error(), "provider execution failed") || !strings.Contains(err.Error(), "provider_err") {
		t.Fatalf("provider error should wrap root cause, got: %v", err)
	}

	err = runInjectError()
	if !errors.Is(err, errInjectSentinel) {
		t.Fatalf("inject callback error lost sentinel: %v", err)
	}
}
