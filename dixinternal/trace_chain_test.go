package dixinternal

import (
	"context"
	"testing"

	"github.com/pubgo/dix/v2/dixtrace"
)

func TestTraceSpanChainInjectResolveProvider(t *testing.T) {
	dixtrace.ResetForTest()

	type depA struct{}
	type depB struct{}

	d := New()
	d.Provide(func() *depA { return &depA{} })
	d.Provide(func(a *depA) *depB { return &depB{} })

	if err := d.TryInject(func(*depB) {}); err != nil {
		t.Fatalf("inject failed: %v", err)
	}

	res := dixtrace.QueryEvents(dixtrace.Query{Event: "span.start", Limit: 2000})
	if len(res.Records) == 0 {
		t.Fatalf("expected span.start events, got none")
	}

	var injectSpan dixtrace.Event
	var injectFuncSpan dixtrace.Event
	paramSpanIDs := map[string]bool{}
	var resolveValueWithParamParent bool
	var providerExecuteSpan dixtrace.Event

	for _, e := range res.Records {
		if e.Event != "span.start" {
			continue
		}
		if e.TraceID == "" || e.SpanID == "" {
			t.Fatalf("span.start missing trace/span id: %+v", e)
		}
		if e.Operation == "inject" && injectSpan.SpanID == "" {
			injectSpan = e
		}
		if e.Operation == "provider.execute" && providerExecuteSpan.SpanID == "" {
			providerExecuteSpan = e
		}
		if e.Operation == "inject.func" && injectFuncSpan.SpanID == "" {
			injectFuncSpan = e
		}
		if e.Operation == "inject.param" && e.SpanID != "" {
			paramSpanIDs[e.SpanID] = true
		}
	}

	if injectSpan.SpanID == "" {
		t.Fatalf("expected inject span.start")
	}
	if injectSpan.ParentSpanID != "" {
		t.Fatalf("root inject span should not have parent, got %q", injectSpan.ParentSpanID)
	}
	if providerExecuteSpan.SpanID == "" {
		t.Fatalf("expected provider.execute span.start")
	}
	if providerExecuteSpan.ParentSpanID == "" {
		t.Fatalf("provider.execute span should have parent")
	}
	if providerExecuteSpan.TraceID != injectSpan.TraceID {
		t.Fatalf("expected same trace id for inject/provider span, got inject=%s provider=%s", injectSpan.TraceID, providerExecuteSpan.TraceID)
	}
	if injectFuncSpan.SpanID == "" {
		t.Fatalf("expected inject.func span.start")
	}
	if injectFuncSpan.ParentSpanID != injectSpan.SpanID {
		t.Fatalf("inject.func should be child of inject span, got parent=%q inject=%q", injectFuncSpan.ParentSpanID, injectSpan.SpanID)
	}
	if len(paramSpanIDs) == 0 {
		t.Fatalf("expected inject.param span.start")
	}

	for _, e := range res.Records {
		if e.Event != "span.start" || e.Operation != "resolve.value" {
			continue
		}
		if paramSpanIDs[e.ParentSpanID] {
			resolveValueWithParamParent = true
			break
		}
	}

	if !resolveValueWithParamParent {
		t.Fatalf("expected at least one resolve.value span directly under inject.param span")
	}
}

func TestTraceInjectParamsAreSiblings(t *testing.T) {
	dixtrace.ResetForTest()

	type depA struct{}
	type depB struct{}

	d := New()
	d.Provide(func() *depA { return &depA{} })
	d.Provide(func() *depB { return &depB{} })

	if err := d.TryInject(func(*depA, *depB) {}); err != nil {
		t.Fatalf("inject failed: %v", err)
	}

	res := dixtrace.QueryEvents(dixtrace.Query{Event: "span.start", Limit: 4000})
	if len(res.Records) == 0 {
		t.Fatalf("expected span.start events, got none")
	}

	var injectSpan dixtrace.Event
	var injectFuncSpan dixtrace.Event
	var paramSpans []dixtrace.Event

	for _, e := range res.Records {
		if e.Event != "span.start" {
			continue
		}
		if e.Operation == "inject" && injectSpan.SpanID == "" {
			injectSpan = e
		}
		if e.Operation == "inject.param" {
			paramSpans = append(paramSpans, e)
		}
		if e.Operation == "inject.func" && injectFuncSpan.SpanID == "" {
			injectFuncSpan = e
		}
	}

	if injectSpan.SpanID == "" {
		t.Fatalf("expected inject span")
	}
	if len(paramSpans) < 2 {
		t.Fatalf("expected at least 2 inject.param spans, got %d", len(paramSpans))
	}
	if injectFuncSpan.SpanID == "" {
		t.Fatalf("expected inject.func span")
	}

	checked := 0
	for _, p := range paramSpans {
		if p.TraceID != injectSpan.TraceID {
			continue
		}
		if p.ParentSpanID != injectFuncSpan.SpanID {
			t.Fatalf("inject.param should be child of inject.func span, got parent=%q inject.func=%q", p.ParentSpanID, injectFuncSpan.SpanID)
		}
		checked++
		if checked >= 2 {
			break
		}
	}

	if checked < 2 {
		t.Fatalf("expected to validate 2 inject.param sibling spans under inject.func")
	}
}

func TestTraceResolveNotNestedUnderInjectParam(t *testing.T) {
	dixtrace.ResetForTest()

	type depA struct{}
	type depB struct{}

	d := New()
	d.Provide(func() *depA { return &depA{} })
	d.Provide(func() *depB { return &depB{} })

	if err := d.TryInject(func(*depA, *depB) {}); err != nil {
		t.Fatalf("inject failed: %v", err)
	}

	res := dixtrace.QueryEvents(dixtrace.Query{Event: "span.start", Limit: 4000})
	if len(res.Records) == 0 {
		t.Fatalf("expected span.start events, got none")
	}

	var injectSpan dixtrace.Event
	var injectFuncSpan dixtrace.Event
	paramSpanIDs := map[string]bool{}
	resolveUnderParam := 0

	for _, e := range res.Records {
		if e.Event != "span.start" {
			continue
		}
		if e.Operation == "inject" && injectSpan.SpanID == "" {
			injectSpan = e
		}
		if e.Operation == "inject.param" && e.SpanID != "" {
			paramSpanIDs[e.SpanID] = true
		}
		if e.Operation == "inject.func" && injectFuncSpan.SpanID == "" {
			injectFuncSpan = e
		}
	}

	if injectSpan.SpanID == "" {
		t.Fatalf("expected inject span")
	}
	if injectFuncSpan.SpanID == "" {
		t.Fatalf("expected inject.func span")
	}

	for _, e := range res.Records {
		if e.Event != "span.start" || e.Operation != "resolve.value" {
			continue
		}
		if paramSpanIDs[e.ParentSpanID] {
			resolveUnderParam++
		}
		if e.ParentSpanID == injectFuncSpan.SpanID {
			t.Fatalf("resolve.value should be nested under inject.param, got inject.func parent %s", e.ParentSpanID)
		}
	}

	if resolveUnderParam == 0 {
		t.Fatalf("expected resolve.value spans directly under inject.param span")
	}
}

func TestTryInjectContextUsesGivenParentSpan(t *testing.T) {
	dixtrace.ResetForTest()

	type depA struct{}

	d := New()
	d.Provide(func() *depA { return &depA{} })

	ctx, root := dixtrace.BeginSpanCtx(context.Background(), "http.request", "GET /demo")
	if err := d.TryInjectContext(ctx, func(*depA) {}); err != nil {
		t.Fatalf("inject failed: %v", err)
	}
	root.End(nil)

	res := dixtrace.QueryEvents(dixtrace.Query{Event: "span.start", Limit: 2000})
	if len(res.Records) == 0 {
		t.Fatalf("expected span.start events, got none")
	}

	var rootSpan, injectSpan dixtrace.Event
	for _, e := range res.Records {
		if e.Event != "span.start" {
			continue
		}
		if e.Operation == "http.request" && rootSpan.SpanID == "" {
			rootSpan = e
		}
		if e.Operation == "inject" && injectSpan.SpanID == "" {
			injectSpan = e
		}
	}

	if rootSpan.SpanID == "" || injectSpan.SpanID == "" {
		t.Fatalf("missing root/inject spans: root=%q inject=%q", rootSpan.SpanID, injectSpan.SpanID)
	}
	if injectSpan.ParentSpanID != rootSpan.SpanID {
		t.Fatalf("inject span should be child of provided context span, parent=%q root=%q", injectSpan.ParentSpanID, rootSpan.SpanID)
	}
	if injectSpan.TraceID != rootSpan.TraceID {
		t.Fatalf("inject should keep trace id from provided context, inject=%q root=%q", injectSpan.TraceID, rootSpan.TraceID)
	}
}

func TestInjectFunctionHasDedicatedSpan(t *testing.T) {
	dixtrace.ResetForTest()

	type depA struct{}

	d := New()
	d.Provide(func() *depA { return &depA{} })

	if err := d.TryInject(func(*depA) {}); err != nil {
		t.Fatalf("inject failed: %v", err)
	}

	res := dixtrace.QueryEvents(dixtrace.Query{Event: "span.start", Limit: 2000})
	if len(res.Records) == 0 {
		t.Fatalf("expected span.start events, got none")
	}

	var injectSpan, injectFuncSpan dixtrace.Event
	for _, e := range res.Records {
		if e.Event != "span.start" {
			continue
		}
		if e.Operation == "inject" && injectSpan.SpanID == "" {
			injectSpan = e
		}
		if e.Operation == "inject.func" && injectFuncSpan.SpanID == "" {
			injectFuncSpan = e
		}
	}

	if injectSpan.SpanID == "" {
		t.Fatalf("expected inject span")
	}
	if injectFuncSpan.SpanID == "" {
		t.Fatalf("expected inject.func span")
	}
	if injectFuncSpan.ParentSpanID != injectSpan.SpanID {
		t.Fatalf("inject.func should be child of inject, parent=%q inject=%q", injectFuncSpan.ParentSpanID, injectSpan.SpanID)
	}
}

func TestStructInputMarkedAsAggregate(t *testing.T) {
	dixtrace.ResetForTest()

	type aggInput struct{}

	d := New()

	if err := d.TryInject(func(aggInput) {}); err != nil {
		t.Fatalf("inject failed: %v", err)
	}

	res := dixtrace.QueryEvents(dixtrace.Query{Event: "span.start", Operation: "inject.param", Limit: 2000})
	if len(res.Records) == 0 {
		t.Fatalf("expected inject.param span.start events")
	}

	found := false
	for _, e := range res.Records {
		if e.Event != "span.start" || e.Operation != "inject.param" {
			continue
		}
		if e.Attrs == nil {
			continue
		}
		if v, ok := e.Attrs["aggregate_input"].(bool); ok && v {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected at least one inject.param span with aggregate_input=true for struct input")
	}
}

func TestProviderResolveNestedUnderProviderInput(t *testing.T) {
	dixtrace.ResetForTest()

	type depA struct{}
	type depB struct{}

	d := New()
	d.Provide(func() *depA { return &depA{} })
	d.Provide(func(a *depA) *depB { return &depB{} })

	if err := d.TryInject(func(*depB) {}); err != nil {
		t.Fatalf("inject failed: %v", err)
	}

	res := dixtrace.QueryEvents(dixtrace.Query{Event: "span.start", Limit: 4000})
	if len(res.Records) == 0 {
		t.Fatalf("expected span.start events, got none")
	}

	providerInputSpanIDs := map[string]bool{}
	providerExecuteSpanIDs := map[string]bool{}
	resolveUnderProviderInput := 0

	for _, e := range res.Records {
		if e.Event != "span.start" {
			continue
		}
		if e.Operation == "provider.input" && e.SpanID != "" {
			providerInputSpanIDs[e.SpanID] = true
		}
		if e.Operation == "provider.execute" && e.SpanID != "" {
			providerExecuteSpanIDs[e.SpanID] = true
		}
	}

	if len(providerInputSpanIDs) == 0 {
		t.Fatalf("expected provider.input spans")
	}

	for _, e := range res.Records {
		if e.Event != "span.start" || e.Operation != "resolve.value" {
			continue
		}
		if providerInputSpanIDs[e.ParentSpanID] {
			resolveUnderProviderInput++
		}
		if providerExecuteSpanIDs[e.ParentSpanID] {
			t.Fatalf("resolve.value for provider input should not be directly under provider.execute")
		}
	}

	if resolveUnderProviderInput == 0 {
		t.Fatalf("expected resolve.value spans directly under provider.input")
	}
}

func TestStructInputDoesNotCreateResolveValueSpan(t *testing.T) {
	dixtrace.ResetForTest()

	type aggInput struct{}

	d := New()

	if err := d.TryInject(func(aggInput) {}); err != nil {
		t.Fatalf("inject failed: %v", err)
	}

	res := dixtrace.QueryEvents(dixtrace.Query{Event: "span.start", Limit: 2000})
	if len(res.Records) == 0 {
		t.Fatalf("expected span.start events")
	}

	foundResolveValueForStruct := false
	foundResolveStruct := false
	for _, e := range res.Records {
		if e.Event != "span.start" {
			continue
		}
		if e.Operation == "resolve.value" && e.Component == "dixinternal.aggInput" {
			foundResolveValueForStruct = true
		}
		if e.Operation == "inject.param" && e.Component == "dixinternal.aggInput" {
			if e.Attrs != nil {
				if v, ok := e.Attrs["aggregate_input"].(bool); ok && v {
					foundResolveStruct = true
				}
			}
		}
	}

	if foundResolveValueForStruct {
		t.Fatalf("struct aggregate input should not create resolve.value span")
	}
	if !foundResolveStruct {
		t.Fatalf("expected aggregate_input marker on inject.param for struct input")
	}
}
