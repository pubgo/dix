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
	if strings.Contains(body, "static/js/views/graph.js") {
		t.Fatalf("legacy index must not reference next graph.js")
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
