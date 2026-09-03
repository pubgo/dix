package main

import "testing"

// 锁定 example/map 的契约:多个 provider 的命名空间在注入时合并。
func TestBuildDatabases(t *testing.T) {
	dbs := buildDatabases()

	want := map[string]string{
		"master":    "postgres://master/db",
		"slave":     "postgres://slave/db",
		"analytics": "postgres://analytics/db",
	}
	for key, dsn := range want {
		if dbs[key] == nil || dbs[key].DSN != dsn {
			t.Fatalf("namespace %q = %v, want dsn %q", key, dbs[key], dsn)
		}
	}
}
