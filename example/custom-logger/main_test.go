package main

import (
	"strings"
	"testing"
)

// 锁定 example/custom-logger 的契约:SetLog 后 dix 的诊断日志
// 流经自定义 handler:缺失依赖会触发 provider-not-found 与
// try-inject-failed 两条告警。
func TestCollectDixLogs(t *testing.T) {
	records := collectDixLogs()

	var notFound, injectFailed bool
	for _, line := range records {
		switch {
		case strings.Contains(line, "provider not found"):
			notFound = true
		case strings.Contains(line, "try inject failed"):
			injectFailed = true
		}
	}
	if !notFound || !injectFailed {
		t.Fatalf("records should contain both warnings, got %d records: %v", len(records), records)
	}
}
