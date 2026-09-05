package dixhttp

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	dix "github.com/pubgo/dix/v2"
	"github.com/pubgo/dix/v2/dixinternal"
)

type issueTarget struct{}

func TestHandleIssuesReturnsActionableDiagnosticFeed(t *testing.T) {
	container := dix.New()
	dix.Provide(container, func() (*issueTarget, error) {
		return nil, errors.New("database unavailable")
	})
	if err := container.TryInject(func(*issueTarget) {}); err == nil {
		t.Fatal("expected injection failure")
	}

	server := NewServer(container)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest("GET", "/api/issues", nil))
	if recorder.Code != 200 {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var issues []IssueInfo
	if err := json.Unmarshal(recorder.Body.Bytes(), &issues); err != nil {
		t.Fatal(err)
	}
	if len(issues) == 0 {
		t.Fatal("expected at least one issue")
	}
	if issues[0].Severity != "error" {
		t.Fatalf("first issue = %+v", issues[0])
	}
}

func TestBuildIssuesMergesErrorsAndSlowProviders(t *testing.T) {
	details := []dixinternal.ProviderDetails{
		{
			OutputType:     "*app.SlowService",
			OutputPkg:      "app/slow",
			FunctionName:   "app.NewSlowService",
			RegistrationID: 7,
			ProviderID:     "provider_7_*app.SlowService",
		},
	}
	recent := []dixinternal.RecentError{
		{
			ErrorType:          "dependency_cycle",
			Component:          "func(*app.Cycle)",
			Message:            "circular dependency",
			RootCause:          "cycle A -> B -> A",
			Hint:               "split the cycle",
			OccurredAtUnixNano: 30,
		},
	}
	stats := []dixinternal.ProviderRuntimeStats{
		{
			FunctionName:      "app.NewSlowService",
			OutputType:        "*app.SlowService",
			RegistrationID:    7,
			ProviderID:        "provider_7_*app.SlowService",
			CallCount:         3,
			AverageDuration:   20 * time.Millisecond,
			LastDuration:      30 * time.Millisecond,
			LastRunAtUnixNano: 20,
		},
	}

	issues := buildIssues(details, recent, stats, 10*time.Millisecond, 10)
	if len(issues) != 2 {
		t.Fatalf("issues = %+v", issues)
	}
	if issues[0].Severity != "error" || issues[0].Kind != "inject_error" {
		t.Fatalf("error issue should sort first: %+v", issues[0])
	}
	if issues[1].Severity != "slow" || issues[1].Kind != "provider_slow" {
		t.Fatalf("slow issue mismatch: %+v", issues[1])
	}
	if issues[1].ProviderID != details[0].ProviderID {
		t.Fatalf("slow issue provider mismatch: %+v", issues[1])
	}
	if issues[1].Module != "app/slow" {
		t.Fatalf("module = %q", issues[1].Module)
	}
}

func TestBuildIssuesLimitsAndSortsStably(t *testing.T) {
	base := time.Now().UnixNano()
	recent := make([]dixinternal.RecentError, 0, 4)
	for i := 0; i < 4; i++ {
		recent = append(recent, dixinternal.RecentError{
			ErrorType:          "inject_failed",
			Component:          "component-" + strconv.Itoa(i),
			Message:            "message-" + strconv.Itoa(i),
			OccurredAtUnixNano: base - int64(i),
		})
	}
	issues := buildIssues(nil, recent, nil, time.Second, 2)
	if len(issues) != 2 {
		t.Fatalf("issues = %d, want 2", len(issues))
	}
	if issues[0].OccurredAtUnixNano < issues[1].OccurredAtUnixNano {
		t.Fatalf("issues are not latest first: %+v", issues)
	}
}
