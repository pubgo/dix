package main

import (
	"context"
	"testing"

	"github.com/pubgo/dix/v2"
)

// 锁定 example/inject-generic 的契约:InjectTContext 构造并填充结构体,
// 嵌套字段递归解析;非 struct 类型 fail-fast panic。
func TestInjectGenericApp(t *testing.T) {
	app := buildApp(context.Background())

	if app.DB == nil || app.DB.Config == nil || app.DB.Config.DSN != "postgres://localhost/app" {
		t.Fatalf("nested fields not resolved: %+v", app.DB)
	}
	if app.Metadata == nil || app.Metadata.Version != "v1" {
		t.Fatalf("metadata not injected: %+v", app.Metadata)
	}
}

// 对照:T 非结构体时 panic,而不是静默返回零值。
func TestInjectGenericRejectsNonStruct(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("InjectT with non-struct T must panic")
		}
	}()
	_ = dix.InjectT[int](dix.New())
}
