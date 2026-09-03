package main

import (
	"context"
	"strings"
	"testing"

	"github.com/pubgo/dix/v2/dixcontext"
)

// 锁定 example/context-container 的契约:
// 容器经 ctx 传递后在调用链深处可用;无容器时 GetOrNil 安全返回 nil。
func TestHandleRequestWithContainer(t *testing.T) {
	if got := handleRequest(context.Background()); !strings.Contains(got, "postgres://localhost/app") {
		t.Fatalf("request handler = %q, want injected dsn", got)
	}

	if dixcontext.GetOrNil(context.Background()) != nil {
		t.Fatal("GetOrNil without container must return nil")
	}
}
