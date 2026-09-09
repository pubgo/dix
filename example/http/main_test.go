package main

import (
	"testing"

	"github.com/pubgo/dix/example/http/app"
)

// 锁定 example/http 的端到端装配契约:十个域模块 + 插件族 + 聚合根。
func TestBuildContainerWiresApplication(t *testing.T) {
	di := buildContainer()

	var application *app.Application
	if err := di.TryInject(func(a *app.Application) { application = a }); err != nil {
		t.Fatalf("TryInject(Application): %v", err)
	}

	if application.Logger == nil {
		t.Fatal("logger not wired")
	}
	services := map[string]any{
		"billing":   application.Billing,
		"inventory": application.Inventory,
		"shipping":  application.Shipping,
		"identity":  application.Identity,
		"analytics": application.Analytics,
		"notify":    application.Notify,
		"search":    application.Search,
		"storage":   application.Storage,
		"media":     application.Media,
		"workflow":  application.Workflow,
	}
	for name, svc := range services {
		if svc == nil {
			t.Fatalf("domain service %s not wired", name)
		}
	}
	if len(application.Plugins) != 40 {
		t.Fatalf("plugins = %d, want 40 (20 keys + 20 worker names)", len(application.Plugins))
	}

	if application.Billing.Repo == nil || application.Billing.Repo.Client == nil || application.Billing.Repo.Client.Config == nil {
		t.Fatalf("billing chain not resolved: %+v", application.Billing)
	}
	if application.Billing.Repo.Client.Config.Env != "prod" {
		t.Fatalf("config value = %+v", application.Billing.Repo.Client.Config)
	}
}
