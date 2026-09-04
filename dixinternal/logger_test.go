package dixinternal

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestDITraceLogsInProvideFlow(t *testing.T) {
	t.Setenv(diTraceEnv, "true")

	originalLogger := logger
	var buf bytes.Buffer
	logger = slog.New(slog.NewTextHandler(&buf, nil)).WithGroup(getLogPackage())
	defer func() {
		logger = originalLogger
	}()

	type provideTraceDep struct{}
	d := New()
	buf.Reset()

	d.Provide(func() *provideTraceDep { return &provideTraceDep{} })

	output := buf.String()

	if !strings.Contains(output, "di_trace provide.start") {
		t.Fatalf("expected provide.start trace log, got: %s", output)
	}

	if !strings.Contains(output, "di_trace provide.signature") {
		t.Fatalf("expected provide.signature trace log, got: %s", output)
	}

	if !strings.Contains(output, "di_trace provide.register.output.done") {
		t.Fatalf("expected provide.register.output.done trace log, got: %s", output)
	}
}

func TestDiagFileNotConfiguredKeepsOriginalScheme(t *testing.T) {
	t.Setenv(diTraceEnv, "false")
	t.Setenv(diagFileEnv, "")

	originalLogger := logger
	var buf bytes.Buffer
	logger = slog.New(slog.NewTextHandler(&buf, nil)).WithGroup(getLogPackage())
	defer func() {
		logger = originalLogger
		resetDiagFileWriterForTest()
	}()

	type dep struct{}
	d := New()
	d.Provide(func() *dep { return &dep{} })
	d.Inject(func(*dep) {})

	output := buf.String()
	if strings.Contains(output, "di_trace ") {
		t.Fatalf("expected no trace logs on console when DIX_TRACE_DI is disabled, got: %s", output)
	}
}

func TestDiagFileConfiguredCollectsTraceAndError(t *testing.T) {
	diagPath := filepath.Join(t.TempDir(), "dix-diag.jsonl")
	t.Setenv(diagFileEnv, diagPath)
	t.Setenv(diTraceEnv, "false")

	originalLogger := logger
	var buf bytes.Buffer
	logger = slog.New(slog.NewTextHandler(&buf, nil)).WithGroup(getLogPackage())
	defer func() {
		logger = originalLogger
		resetDiagFileWriterForTest()
	}()

	type missingDep struct{}
	d := New()
	_ = d.TryInject(func(*missingDep) {})

	if strings.Contains(buf.String(), "di_trace ") {
		t.Fatalf("expected no console trace when DIX_TRACE_DI is disabled, got: %s", buf.String())
	}

	contentBytes, err := os.ReadFile(diagPath)
	if err != nil {
		t.Fatalf("expected diagnostic file to be created, got error: %v", err)
	}
	content := string(contentBytes)

	if !strings.Contains(content, `"kind":"trace"`) {
		t.Fatalf("expected diagnostic file to include trace records, got: %s", content)
	}

	if !strings.Contains(content, `"kind":"error"`) {
		t.Fatalf("expected diagnostic file to include error records, got: %s", content)
	}

	if strings.Contains(content, `"kind":"llm"`) {
		t.Fatal("llm records must no longer be written (channel removed)")
	}

	if !strings.Contains(content, `"record_id":`) {
		t.Fatalf("expected diagnostic file to include record_id metadata, got: %s", content)
	}

	if !strings.Contains(content, `"source":"dix"`) {
		t.Fatalf("expected diagnostic file to include source metadata, got: %s", content)
	}
}
