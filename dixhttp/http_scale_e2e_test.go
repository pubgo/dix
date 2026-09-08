package dixhttp

import (
	"net/http/httptest"
	"strings"
	"testing"

	dix "github.com/pubgo/dix/v2"
)

func TestLegacyGraphEmbedsSharedHelpers(t *testing.T) {
	container := dix.New()
	server := NewServer(container)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest("GET", "/", nil))

	if recorder.Code != 200 {
		t.Fatalf("status = %d", recorder.Code)
	}
	body := recorder.Body.String()
	for _, asset := range []string{"static/js/graph_state.mjs", "static/js/legacy/app.js", "模块地图"} {
		if !strings.Contains(body, asset) {
			t.Fatalf("missing %s", asset)
		}
	}
	if strings.Contains(body, `value="panorama"`) || strings.Contains(body, "全景") {
		t.Fatalf("panorama layout must be removed from legacy UI")
	}
	if strings.Contains(body, "static/js/views/graph.js") {
		t.Fatalf("legacy index must not reference next graph.js")
	}
}

func TestInventoryHelpersPresentInBundle(t *testing.T) {
	container := dix.New()
	server := NewServer(container)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest("GET", "/static/js/graph_state.mjs", nil))
	if recorder.Code != 200 {
		t.Fatalf("status = %d", recorder.Code)
	}
	body := recorder.Body.String()
	for _, sym := range []string{
		"buildProviderInventory",
		"buildProvidersPyramidView",
		"buildProviderDependencyGraph",
		"assignProviderPyramidLevels",
		"buildTypesPyramidView",
		"buildTypeDependencyGraph",
		"providerDisplayLabel",
		"HIERARCHICAL_NODE_BUDGET",
		"labelLodVisibleIds",
		"buildModuleMapGraph",
	} {
		if !strings.Contains(body, sym) {
			t.Fatalf("graph_state.mjs missing %s", sym)
		}
	}
	for _, gone := range []string{
		"layoutPanoramaPositions",
		"buildPanoramaProviderGraph",
		"buildPanoramaStructureGraph",
		"resolvePanoramaCameraScale",
		"labelLodByBand",
	} {
		if strings.Contains(body, gone) {
			t.Fatalf("graph_state.mjs still contains removed panorama helper %s", gone)
		}
	}
}

func TestNextRouteRemoved(t *testing.T) {
	container := dix.New()
	server := NewServer(container)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest("GET", "/next", nil))
	if recorder.Code != 404 {
		t.Fatalf("/next status = %d, want 404", recorder.Code)
	}
}
