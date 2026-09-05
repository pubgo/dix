package dixhttp

import (
	"net/http/httptest"
	"strings"
	"testing"

	dix "github.com/pubgo/dix/v2"
)

func TestNextGraphDefaultsToModuleFirstAssets(t *testing.T) {
	container := dix.New()
	server := NewServer(container)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest("GET", "/next", nil))

	if recorder.Code != 200 {
		t.Fatalf("status = %d", recorder.Code)
	}
	body := recorder.Body.String()
	for _, asset := range []string{"static/js/views/graph.js", "static/js/main.js"} {
		if !strings.Contains(body, asset) {
			t.Fatalf("missing asset %s", asset)
		}
	}
}
