package dixhttp

import (
	"testing"

	"github.com/pubgo/dix/v2/dixinternal"
)

func TestPackageInfoUsesResolvedOutputPackage(t *testing.T) {
	details := []dixinternal.ProviderDetails{
		{OutputType: "*main.Plugin[main.RoleReader]", OutputPkg: "main"},
		{OutputType: "*billing.Client", OutputPkg: "example/billing"},
	}
	packages := buildPackageInfos(details)
	if len(packages) != 2 {
		t.Fatalf("packages = %+v", packages)
	}
	if packages[0].Name != "example/billing" || packages[1].Name != "main" {
		t.Fatalf("malformed generic package path retained or rows not sorted: %+v", packages)
	}
}

func TestAggregateProviderInfosKeepsCollidingInputPkgs(t *testing.T) {
	details := []dixinternal.ProviderDetails{
		{
			RegistrationID: 1,
			OutputType:     "*app.Application",
			OutputPkg:      "example/http/app",
			FunctionName:   "app.Provide",
			InputTypes: []string{
				"*handler.Handler",
				"*handler.Handler",
				"*plugins.Platform",
			},
			InputPkgs: []string{
				"example/http/domain/billing/handler",
				"example/http/domain/inventory/handler",
				"example/http/plugins",
			},
		},
	}
	_, providers := aggregateProviderInfos(details, "", 0)
	if len(providers) != 1 {
		t.Fatalf("providers = %d, want 1", len(providers))
	}
	p := providers[0]
	if len(p.InputTypes) != 3 || len(p.InputPkgs) != 3 {
		t.Fatalf("inputs collapsed: types=%v pkgs=%v", p.InputTypes, p.InputPkgs)
	}
	if p.InputPkgs[0] != "example/http/domain/billing/handler" || p.InputPkgs[1] != "example/http/domain/inventory/handler" {
		t.Fatalf("input pkgs = %v", p.InputPkgs)
	}
}

func TestDependencyPackageFilterUsesOutputPkg(t *testing.T) {
	details := []dixinternal.ProviderDetails{
		{
			FunctionName: "analytics.NewClient",
			OutputType:   "*analytics.Client",
			OutputPkg:    "github.com/pubgo/dix/example/http/domain/analytics/infra",
			FunctionPkg:  "github.com/pubgo/dix/example/http/domain/analytics/infra",
		},
		{
			FunctionName: "billing.NewClient",
			OutputType:   "*billing.Client",
			OutputPkg:    "github.com/pubgo/dix/example/http/domain/billing/infra",
			FunctionPkg:  "github.com/pubgo/dix/example/http/domain/billing/infra",
		},
	}
	data := buildDependencyData(details, nil, "github.com/pubgo/dix/example/http/domain/analytics/infra", 0)
	if len(data.Providers) != 1 {
		t.Fatalf("providers = %+v, want 1 analytics provider", data.Providers)
	}
	if data.Providers[0].OutputType != "*analytics.Client" {
		t.Fatalf("got provider %+v", data.Providers[0])
	}
	// extractPackage("*analytics.Client") == "analytics" should match path suffix
	byShort := buildDependencyData(details, nil, "analytics", 0)
	if len(byShort.Providers) != 1 {
		t.Fatalf("short name analytics should match OutputPkg path suffix, got %+v", byShort.Providers)
	}
	byMain := buildDependencyData(details, nil, "main", 0)
	if len(byMain.Providers) != 0 {
		t.Fatalf("filter main must not match domain/.../analytics via substring, got %+v", byMain.Providers)
	}
}
