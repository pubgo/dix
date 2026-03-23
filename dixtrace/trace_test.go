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

func TestDetachedSpanDoesNotBecomeParentOfNestedSpan(t *testing.T) {
	ResetForTest()

	ctx, root := BeginSpanCtx(context.Background(), "inject", "comp")
	err := RunDetachedSpanCtx(ctx, "inject.param", "arg0", func() error {
		_, inner := BeginSpanCtx(ctx, "resolve.value", "dep")
		inner.End(nil)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	root.End(nil)

	result := QueryEvents(Query{Limit: 2000})
	if result.Total == 0 {
		t.Fatal("expected trace events")
	}

	var injectStart, paramStart, resolveStart Event
	for _, rec := range result.Records {
		if rec.Event != "span.start" {
			continue
		}
		switch rec.Operation {
		case "inject":
			injectStart = rec
		case "inject.param":
			paramStart = rec
		case "resolve.value":
			resolveStart = rec
		}
	}

	if injectStart.SpanID == "" || paramStart.SpanID == "" || resolveStart.SpanID == "" {
		t.Fatalf("missing expected span starts: inject=%q param=%q resolve=%q", injectStart.SpanID, paramStart.SpanID, resolveStart.SpanID)
	}
	if paramStart.ParentSpanID != injectStart.SpanID {
		t.Fatalf("expected inject.param parent=%s, got %s", injectStart.SpanID, paramStart.ParentSpanID)
	}
	if resolveStart.ParentSpanID != injectStart.SpanID {
		t.Fatalf("expected resolve.value parent=%s, got %s", injectStart.SpanID, resolveStart.ParentSpanID)
	}
	if resolveStart.ParentSpanID == paramStart.SpanID {
		t.Fatalf("resolve.value must not be nested under inject.param")
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
