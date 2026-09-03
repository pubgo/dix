package main

import (
	"strings"
	"testing"

	"github.com/pubgo/dix/v2"
)

// 锁定 example/global 的契约:全局容器注册即可注入,Version 非空。
func TestBuildGlobalApp(t *testing.T) {
	if !strings.HasPrefix(dix.Version(), "v2.") {
		t.Fatalf("dix.Version() = %q, want v2.x", dix.Version())
	}

	srv := buildGlobalApp()
	if srv == nil || srv.Endpoint != "localhost:8080" {
		t.Fatalf("global app = %+v, want injected server", srv)
	}
}
