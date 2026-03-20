package dixinternal

import (
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
