package dixtrace

import (
	"context"
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
	if rootStart.TraceID != "t-1" {
		t.Fatalf("expected root trace id t-1, got %s", rootStart.TraceID)
	}
	if childStart.TraceID != rootStart.TraceID {
		t.Fatalf("child should keep parent trace id, child=%s root=%s", childStart.TraceID, rootStart.TraceID)
	}
	if anotherRootStart.TraceID != "t-2" {
		t.Fatalf("expected next root trace id t-2, got %s", anotherRootStart.TraceID)
	}
}
