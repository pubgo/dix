package dixinternal

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestCurrentLLMDiagMode(t *testing.T) {
	t.Setenv(llmDiagModeEnv, "")
	if got := currentLLMDiagMode(); got != llmDiagModeHuman {
		t.Fatalf("expected human for empty env, got %s", got)
	}

	t.Setenv(llmDiagModeEnv, "ONLY")
	if got := currentLLMDiagMode(); got != llmDiagModeMachine {
		t.Fatalf("expected machine for ONLY env, got %s", got)
	}

	t.Setenv(llmDiagModeEnv, "machine-only")
	if got := currentLLMDiagMode(); got != llmDiagModeMachine {
		t.Fatalf("expected machine for machine-only env, got %s", got)
	}

	t.Setenv(llmDiagModeEnv, "unknown")
	if got := currentLLMDiagMode(); got != llmDiagModeHuman {
		t.Fatalf("expected human for unknown env, got %s", got)
	}
}

func TestLLMDiagOnlyModeSuppressesHumanLogs(t *testing.T) {
	type missingDep struct{}

	t.Setenv(llmDiagModeEnv, llmDiagModeMachine)
	originalLogger := logger
	logger = createDefaultLogger()
	defer func() {
		logger = originalLogger
	}()

	d := New()
	output := captureStderr(t, func() {
		_ = d.TryInject(func(*missingDep) {})
	})

	if !strings.Contains(output, "DIX_LLM_DIAG ") {
		t.Fatalf("expected output to contain machine diagnostic line, got: %s", output)
	}

	if strings.Contains(strings.ToLower(output), "try inject failed") {
		t.Fatalf("expected human warn logs to be suppressed in only mode, got: %s", output)
	}
}

func TestShouldTraceDependencyFlow(t *testing.T) {
	t.Setenv(diTraceEnv, "")
	if shouldTraceDependencyFlow() {
		t.Fatal("expected empty DIX_TRACE_DI to be disabled")
	}

	t.Setenv(diTraceEnv, "true")
	if !shouldTraceDependencyFlow() {
		t.Fatal("expected DIX_TRACE_DI=true to enable trace")
	}

	t.Setenv(diTraceEnv, "1")
	if !shouldTraceDependencyFlow() {
		t.Fatal("expected DIX_TRACE_DI=1 to enable trace")
	}

	t.Setenv(diTraceEnv, "off")
	if shouldTraceDependencyFlow() {
		t.Fatal("expected DIX_TRACE_DI=off to disable trace")
	}
}

func TestDITraceLogsInInjectFlow(t *testing.T) {
	t.Setenv(diTraceEnv, "true")
	t.Setenv(llmDiagModeEnv, llmDiagModeHuman)

	originalLogger := logger
	var buf bytes.Buffer
	logger = slog.New(slog.NewTextHandler(&buf, nil)).WithGroup(getLogPackage())
	defer func() {
		logger = originalLogger
	}()

	type traceDep struct{}
	d := New()
	d.Provide(func() *traceDep { return &traceDep{} })

	d.Inject(func(*traceDep) {})

	output := buf.String()

	if !strings.Contains(output, "di_trace inject.start") {
		t.Fatalf("expected inject.start trace log, got: %s", output)
	}

	if !strings.Contains(output, "di_trace resolve.value.search_provider.start") {
		t.Fatalf("expected provider search trace log, got: %s", output)
	}

	if !strings.Contains(output, "di_trace provider.call.start") {
		t.Fatalf("expected provider.call.start trace log, got: %s", output)
	}
}
