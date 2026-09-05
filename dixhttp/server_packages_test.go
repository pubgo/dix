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
