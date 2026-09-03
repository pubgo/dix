package main

import "testing"

// 锁定 example/struct-out 的契约:一次 provider 调用产出多个依赖,
// 且各字段共享同一底层实例。
func TestBuildServices(t *testing.T) {
	user, order := buildServices()

	if user == nil || order == nil {
		t.Fatalf("services not injected: user=%v order=%v", user, order)
	}
	if user.DB == nil || order.DB == nil {
		t.Fatalf("database not injected: %+v %+v", user.DB, order.DB)
	}
	if user.DB != order.DB {
		t.Fatal("UserSvc.DB and OrderSvc.DB must share the same *Database instance")
	}
	if user.DB.Config == nil || user.DB.Config.DSN != "postgres://localhost/shop" {
		t.Fatalf("nested config not resolved: %+v", user.DB.Config)
	}
}
