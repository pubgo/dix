package main

import "testing"

// 锁定 example/struct-in 的契约:导出指针字段递归解析,嵌套依赖逐层填充。
func TestBuildApp(t *testing.T) {
	app := buildApp()

	if app.DB == nil || app.DB.Config == nil {
		t.Fatalf("nested fields not resolved: %+v", app.DB)
	}
	if app.DB.Config.DSN != "postgres://localhost/app" {
		t.Fatalf("DSN = %q, want value from provider chain", app.DB.Config.DSN)
	}
	if app.Metadata == nil || app.Metadata.Version != "v1" {
		t.Fatalf("metadata not injected: %+v", app.Metadata)
	}
}
