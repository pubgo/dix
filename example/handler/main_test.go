package main

import "testing"

// 锁定 example/handler 的契约:单值与命名空间 map 并存互不干扰,
// 两次注入(函数注入与结构体注入)拿到同一批单例依赖。
func TestBuildHandler(t *testing.T) {
	di, h := buildHandler()

	if h.Logger == nil {
		t.Fatal("logger not injected")
	}
	if h.Redis == nil || h.Redis.Addr != "127.0.0.1:6379" {
		t.Fatalf("single redis = %+v, want default addr", h.Redis)
	}
	if h.All["cache"] == nil || h.All["cache"].Addr != "127.0.0.1:6380" {
		t.Fatalf("namespaced redis = %+v, want cache addr", h.All["cache"])
	}

	// 函数注入拿到的 *Redis 与结构体注入的是同一个实例(容器级单例)。
	var fnRedis *Redis
	if err := di.TryInject(func(r *Redis) { fnRedis = r }); err != nil {
		t.Fatalf("TryInject: %v", err)
	}
	if fnRedis != h.Redis {
		t.Fatal("same-type dependencies must be singleton-shared across injections")
	}
}
