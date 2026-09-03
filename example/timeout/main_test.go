package main

import (
	"strings"
	"testing"
)

// 锁定 example/timeout 的契约:超时使注入失败;已超时的 provider
// 不再重试,后续注入快速失败并给出明确提示。
func TestBuildWithTimeout(t *testing.T) {
	firstErr, secondErr := buildWithTimeout()

	if firstErr == nil || !strings.Contains(firstErr.Error(), "provider execution timeout after 100ms") {
		t.Fatalf("first inject = %v, want timeout error", firstErr)
	}
	if secondErr == nil || !strings.Contains(secondErr.Error(), "timed out previously and will not be re-executed") {
		t.Fatalf("second inject = %v, want no-retry error", secondErr)
	}
}
