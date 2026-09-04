package main

import "testing"

// 锁定 example/http 的端到端装配契约:十个域模块 + 插件族 + 聚合根,
// 全链路(Config → Client → Repo → Service → Handler)可解析、已实例化。
func TestBuildContainerWiresApplication(t *testing.T) {
	di := buildContainer()

	var app *Application
	if err := di.TryInject(func(a *Application) { app = a }); err != nil {
		t.Fatalf("TryInject(Application): %v", err)
	}

	if app.Logger == nil {
		t.Fatal("logger not wired")
	}
	services := map[string]any{
		"billing":   app.Billing,
		"inventory": app.Inventory,
		"shipping":  app.Shipping,
		"identity":  app.Identity,
		"analytics": app.Analytics,
		"notify":    app.Notify,
		"search":    app.Search,
		"storage":   app.Storage,
		"media":     app.Media,
		"workflow":  app.Workflow,
	}
	for name, svc := range services {
		if svc == nil {
			t.Fatalf("domain service %s not wired", name)
		}
	}
	if len(app.Plugins) != 120 {
		t.Fatalf("plugins = %d, want 120", len(app.Plugins))
	}

	// 领域链路抽检:计费服务 → 仓储 → 客户端 → 配置
	if app.Billing.Repo == nil || app.Billing.Repo.Client == nil || app.Billing.Repo.Client.Config == nil {
		t.Fatalf("billing chain not resolved: %+v", app.Billing)
	}
	if app.Billing.Repo.Client.Config.Env != "prod" {
		t.Fatalf("config value = %+v", app.Billing.Repo.Client.Config)
	}
}
