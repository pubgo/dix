package dixtrace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMemorySinkQuery(t *testing.T) {
	sink := NewMemorySink(16)
	sink.Write(Event{ID: 1, OccurredAt: 100, Event: "provide.start", Operation: "provide", Status: "ok", Component: "A"})
	sink.Write(Event{ID: 2, OccurredAt: 101, Event: "inject.start", Operation: "inject", Status: "ok", Component: "B"})
	sink.Write(Event{ID: 3, OccurredAt: 102, Event: "provider.call.failed", Operation: "provider", Status: "error", Component: "C"})

	result := sink.Query(Query{Operation: "provider", Status: "error", Limit: 10})
	if result.Total != 1 {
		t.Fatalf("expected total=1, got %d", result.Total)
	}
	if result.Returned != 1 || len(result.Records) != 1 {
		t.Fatalf("expected returned=1, got returned=%d len=%d", result.Returned, len(result.Records))
	}
	if result.Records[0].ID != 3 {
		t.Fatalf("expected id=3, got %d", result.Records[0].ID)
	}
}

func TestBeginSpanCtxPropagatesAcrossGoroutine(t *testing.T) {
	ResetForTest()

	ctx, root := BeginSpanCtx(context.Background(), "inject", "comp")
	done := make(chan struct{})

	go func() {
		defer close(done)
		_, child := BeginSpanCtx(ctx, "resolve.value", "dep")
		child.End(nil)
	}()

	<-done
	root.End(nil)

	result := QueryEvents(Query{Event: "span.start", Limit: 2000})
	if result.Total == 0 {
		t.Fatal("expected span.start events")
	}

	var injectStart, resolveStart Event
	for _, rec := range result.Records {
		if rec.Event != "span.start" {
			continue
		}
		switch rec.Operation {
		case "inject":
			injectStart = rec
		case "resolve.value":
			resolveStart = rec
		}
	}

	if injectStart.SpanID == "" || resolveStart.SpanID == "" {
		t.Fatalf("missing expected spans: inject=%q resolve=%q", injectStart.SpanID, resolveStart.SpanID)
	}
	if resolveStart.ParentSpanID != injectStart.SpanID {
		t.Fatalf("expected resolve.value parent=%s, got %s", injectStart.SpanID, resolveStart.ParentSpanID)
	}
	if resolveStart.TraceID != injectStart.TraceID {
		t.Fatalf("expected same trace id, inject=%s resolve=%s", injectStart.TraceID, resolveStart.TraceID)
	}
}

func TestTraceIDSequenceOnlyIncrementsOnRootSpan(t *testing.T) {
	ResetForTest()

	ctx, root := BeginSpanCtx(context.Background(), "root", "comp")
	_, child := BeginSpanCtx(ctx, "child", "comp")
	child.End(nil)
	root.End(nil)

	_, anotherRoot := BeginSpanCtx(context.Background(), "another.root", "comp")
	anotherRoot.End(nil)

	result := QueryEvents(Query{Event: "span.start", Limit: 2000})
	if result.Total == 0 {
		t.Fatal("expected span.start events")
	}

	var rootStart, childStart, anotherRootStart Event
	for _, rec := range result.Records {
		if rec.Event != "span.start" {
			continue
		}
		switch rec.Operation {
		case "root":
			rootStart = rec
		case "child":
			childStart = rec
		case "another.root":
			anotherRootStart = rec
		}
	}

	if rootStart.TraceID == "" || childStart.TraceID == "" || anotherRootStart.TraceID == "" {
		t.Fatalf("missing spans: root=%q child=%q another=%q", rootStart.TraceID, childStart.TraceID, anotherRootStart.TraceID)
	}
	// 随机 TraceID 语义:根 span 各得随机 ID(32 hex),子 span 继承父 ID。
	if len(rootStart.TraceID) != 32 || rootStart.TraceID == anotherRootStart.TraceID {
		t.Fatalf("root trace ids should be distinct 32-hex, got root=%q another=%q", rootStart.TraceID, anotherRootStart.TraceID)
	}
	if childStart.TraceID != rootStart.TraceID {
		t.Fatalf("child should keep parent trace id, child=%s root=%s", childStart.TraceID, rootStart.TraceID)
	}
}

func TestMemorySinkQueryLimitReturnsLatestAndPaginationCursor(t *testing.T) {
	sink := NewMemorySink(16)
	sink.Write(Event{ID: 1, OccurredAt: 100, Event: "e1"})
	sink.Write(Event{ID: 2, OccurredAt: 200, Event: "e2"})
	sink.Write(Event{ID: 3, OccurredAt: 300, Event: "e3"})

	first := sink.Query(Query{Limit: 2})
	if first.Total != 3 || first.Returned != 2 {
		t.Fatalf("expected total=3 returned=2, got total=%d returned=%d", first.Total, first.Returned)
	}
	if len(first.Records) != 2 || first.Records[0].ID != 3 || first.Records[1].ID != 2 {
		t.Fatalf("expected latest first [3,2], got %+v", first.Records)
	}
	if first.NextBefore != 2 {
		t.Fatalf("expected next_before_id=2, got %d", first.NextBefore)
	}

	second := sink.Query(Query{Limit: 2, BeforeID: first.NextBefore})
	if second.Total != 1 || second.Returned != 1 {
		t.Fatalf("expected second page total=1 returned=1, got total=%d returned=%d", second.Total, second.Returned)
	}
	if len(second.Records) != 1 || second.Records[0].ID != 1 {
		t.Fatalf("expected second page [1], got %+v", second.Records)
	}
}

func TestFileSinkAlwaysTruncatesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.jsonl")
	if err := os.WriteFile(path, []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("failed to seed trace file: %v", err)
	}

	sink := NewFileSink(path)
	sink.Write(Event{ID: 1, OccurredAt: 1, Event: "span.start"})

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read trace file: %v", err)
	}
	if strings.Contains(string(b), "seed\n") {
		t.Fatalf("expected existing content to be truncated, got: %q", string(b))
	}
}

func TestAppendFileSinkKeepsExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "diag.jsonl")
	if err := os.WriteFile(path, []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("failed to seed diag file: %v", err)
	}

	sink := newAppendFileSink(path)
	sink.Write(Event{ID: 1, OccurredAt: 1, Event: "span.start"})

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read diag file: %v", err)
	}
	content := string(b)
	if !strings.Contains(content, "seed\n") {
		t.Fatalf("expected existing content to be kept, got: %q", content)
	}
	if !strings.Contains(content, "\"event\":\"span.start\"") {
		t.Fatalf("expected appended span event, got: %q", content)
	}
}

func TestResolveTraceFilePathFromEnvPreferTrace(t *testing.T) {
	t.Setenv("DIX_TRACE_FILE", "trace.jsonl")
	t.Setenv("DIX_DIAG_FILE", "diag.jsonl")

	path, appendOnly := resolveTraceFilePathFromEnv()
	if path != "trace.jsonl" {
		t.Fatalf("expected trace path, got %q", path)
	}
	if appendOnly {
		t.Fatalf("expected truncate mode for explicit trace file")
	}
}

func TestResolveTraceFilePathFromEnvNoFallback(t *testing.T) {
	t.Setenv("DIX_TRACE_FILE", "")
	t.Setenv("DIX_DIAG_FILE", "diag.jsonl")

	// The trace file sink must not fall back to DIX_DIAG_FILE: the diag file
	// uses a different JSON schema and is read by dixinternal record readers.
	path, appendOnly := resolveTraceFilePathFromEnv()
	if path != "" {
		t.Fatalf("expected trace sink to be disabled without DIX_TRACE_FILE, got %q", path)
	}
	if appendOnly {
		t.Fatalf("expected append mode to be irrelevant when disabled")
	}
}

func TestTraceIDIsRandomHex(t *testing.T) {
	ResetForTest()
	s1 := BeginSpan("op", "c")
	s2 := BeginSpan("op", "c")
	t1, _, _ := s1.IDs()
	t2, _, _ := s2.IDs()
	if t1 == t2 {
		t.Fatal("root spans must get distinct random trace ids")
	}
	if len(t1) != 32 {
		t.Fatalf("trace id len = %d, want 32 hex", len(t1))
	}
}

func TestContainerIDStamping(t *testing.T) {
	ResetForTest()
	ctx := WithContainer(context.Background(), "cont-1")
	ctx, span := BeginSpanCtx(ctx, "inject", "c")
	_, child := BeginSpanCtx(ctx, "resolve", "c")
	child.End(nil)
	span.End(nil)

	res := QueryEvents(Query{ContainerID: "cont-1"})
	if res.Total == 0 {
		t.Fatal("events should carry container id")
	}
	for _, rec := range res.Records {
		if rec.ContainerID != "cont-1" {
			t.Fatalf("event %s has container %q", rec.Event, rec.ContainerID)
		}
	}
	if res := QueryEvents(Query{ContainerID: "cont-other"}); res.Total != 0 {
		t.Fatal("other container must not match")
	}
}

func TestNewContainerID(t *testing.T) {
	a, b := NewContainerID(), NewContainerID()
	if a == b || len(a) != 16 {
		t.Fatalf("container ids must be random 16 hex, got %q %q", a, b)
	}
}
