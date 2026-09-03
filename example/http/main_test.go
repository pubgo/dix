package main

import "testing"

// 锁定 example/http 的端到端装配契约:接口绑定、map/list 聚合、
// 结构体多输出、多层链路(Config → Services → Controllers → Application)。
func TestBuildContainerWiresApplication(t *testing.T) {
	di := buildContainer()

	var app *Application
	if err := di.TryInject(func(a *Application) { app = a }); err != nil {
		t.Fatalf("TryInject(Application): %v", err)
	}

	if app.Config == nil || app.Config.Database == nil || app.Config.Cache == nil || app.Config.HTTP == nil {
		t.Fatalf("config not wired: %+v", app.Config)
	}
	if app.Config.Database.Host != "localhost" || app.Config.Cache.Type != "redis" {
		t.Fatalf("unexpected config values: %+v", app.Config)
	}

	if app.UserController == nil || app.UserController.UserService == nil {
		t.Fatal("user controller chain not wired")
	}
	if app.OrderController == nil || app.OrderController.OrderService == nil ||
		app.OrderController.PaymentService == nil || app.OrderController.NotificationService == nil {
		t.Fatal("order controller chain not wired")
	}

	// 订单服务与用户服务共享同一个 *UserService 实例(容器级单例)。
	if app.OrderController.OrderService.UserService != app.UserController.UserService {
		t.Fatal("*UserService must be singleton-shared across services")
	}

	if len(app.AllServices) != 4 {
		t.Fatalf("AllServices = %d, want 4", len(app.AllServices))
	}

	// 注意:[]Service provider 的产物会以 "default" 分组混入 map 注入,
	// 因此 map[string]Service 的 key 是 4 个命名 key + "default" = 5。
	for _, key := range []string{"user", "order", "payment", "notification"} {
		if app.ServiceMap[key] == nil {
			t.Fatalf("ServiceMap missing key %q: %v", key, keys(app.ServiceMap))
		}
	}
	if len(app.ServiceMap) != 5 {
		t.Fatalf("ServiceMap has %d keys, want 5 (4 named + default)", len(app.ServiceMap))
	}
}

func keys(m map[string]Service) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
