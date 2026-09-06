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

func TestDependencyPackageFilterUsesOutputPkg(t *testing.T) {
	details := []dixinternal.ProviderDetails{
		{
			FunctionName: "analytics.NewClient",
			OutputType:   "*analytics.Client",
			OutputPkg:    "github.com/pubgo/dix/example/http/domain/analytics",
			FunctionPkg:  "github.com/pubgo/dix/example/http/domain/analytics",
		},
		{
			FunctionName: "billing.NewClient",
			OutputType:   "*billing.Client",
			OutputPkg:    "github.com/pubgo/dix/example/http/domain/billing",
			FunctionPkg:  "github.com/pubgo/dix/example/http/domain/billing",
		},
	}
	data := buildDependencyData(details, nil, "github.com/pubgo/dix/example/http/domain/analytics", 0)
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
