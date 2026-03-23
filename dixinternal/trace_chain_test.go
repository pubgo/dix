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
	var cycleCheckSpan dixtrace.Event
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
		if e.Operation == "inject.cycle_check" && cycleCheckSpan.SpanID == "" {
			cycleCheckSpan = e
		}
		if e.Operation == "provider.execute" && providerExecuteSpan.SpanID == "" {
			providerExecuteSpan = e
		}
		if e.Operation == "inject.param" && e.SpanID != "" {
			paramSpanIDs[e.SpanID] = true
		}
	}

	if injectSpan.SpanID == "" {
		t.Fatalf("expected inject span.start")
	}
	if cycleCheckSpan.SpanID == "" {
		t.Fatalf("expected inject.cycle_check span.start")
	}
	if injectSpan.ParentSpanID != cycleCheckSpan.SpanID {
		t.Fatalf("inject span should be child of cycle_check, parent=%q cycle=%q", injectSpan.ParentSpanID, cycleCheckSpan.SpanID)
	}
	if injectSpan.TraceID != cycleCheckSpan.TraceID {
		t.Fatalf("inject should keep trace id from cycle_check, inject=%q cycle=%q", injectSpan.TraceID, cycleCheckSpan.TraceID)
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
	}

	if injectSpan.SpanID == "" {
		t.Fatalf("expected inject span")
	}
	if len(paramSpans) < 2 {
		t.Fatalf("expected at least 2 inject.param spans, got %d", len(paramSpans))
	}
	checked := 0
	for _, p := range paramSpans {
		if p.TraceID != injectSpan.TraceID {
			continue
		}
		if p.ParentSpanID != injectSpan.SpanID {
			t.Fatalf("inject.param should be child of inject span, got parent=%q inject=%q", p.ParentSpanID, injectSpan.SpanID)
		}
		checked++
		if checked >= 2 {
			break
		}
	}

	if checked < 2 {
		t.Fatalf("expected to validate 2 inject.param sibling spans under inject")
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
	}

	if injectSpan.SpanID == "" {
		t.Fatalf("expected inject span")
	}
	for _, e := range res.Records {
		if e.Event != "span.start" || e.Operation != "resolve.value" {
			continue
		}
		if paramSpanIDs[e.ParentSpanID] {
			resolveUnderParam++
		}
		if e.ParentSpanID == injectSpan.SpanID {
			t.Fatalf("resolve.value should be nested under inject.param, got inject parent %s", e.ParentSpanID)
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

	var rootSpan, cycleSpan, injectSpan dixtrace.Event
	for _, e := range res.Records {
		if e.Event != "span.start" {
			continue
		}
		if e.Operation == "http.request" && rootSpan.SpanID == "" {
			rootSpan = e
		}
		if e.Operation == "inject.cycle_check" && cycleSpan.SpanID == "" {
			cycleSpan = e
		}
		if e.Operation == "inject" && injectSpan.SpanID == "" {
			injectSpan = e
		}
	}

	if rootSpan.SpanID == "" || cycleSpan.SpanID == "" || injectSpan.SpanID == "" {
		t.Fatalf("missing root/cycle/inject spans: root=%q cycle=%q inject=%q", rootSpan.SpanID, cycleSpan.SpanID, injectSpan.SpanID)
	}
	if cycleSpan.ParentSpanID != rootSpan.SpanID {
		t.Fatalf("cycle_check span should be child of provided context span, parent=%q root=%q", cycleSpan.ParentSpanID, rootSpan.SpanID)
	}
	if injectSpan.ParentSpanID != cycleSpan.SpanID {
		t.Fatalf("inject span should be child of cycle_check span, parent=%q cycle=%q", injectSpan.ParentSpanID, cycleSpan.SpanID)
	}
	if injectSpan.TraceID != rootSpan.TraceID {
		t.Fatalf("inject should keep trace id from provided context, inject=%q root=%q", injectSpan.TraceID, rootSpan.TraceID)
	}
	if cycleSpan.TraceID != rootSpan.TraceID {
		t.Fatalf("cycle_check should keep trace id from provided context, cycle=%q root=%q", cycleSpan.TraceID, rootSpan.TraceID)
	}
}

func TestTryInjectContextInjectNestedUnderCycleCheck(t *testing.T) {
	dixtrace.ResetForTest()

	type depA struct{}

	d := New()
	d.Provide(func() *depA { return &depA{} })

	if err := d.TryInjectContext(context.Background(), func(*depA) {}); err != nil {
		t.Fatalf("inject failed: %v", err)
	}

	res := dixtrace.QueryEvents(dixtrace.Query{Event: "span.start", Limit: 2000})
	if len(res.Records) == 0 {
		t.Fatalf("expected span.start events, got none")
	}

	var cycleSpan, injectSpan dixtrace.Event
	for _, e := range res.Records {
		if e.Event != "span.start" {
			continue
		}
		switch e.Operation {
		case "inject.cycle_check":
			if cycleSpan.SpanID == "" {
				cycleSpan = e
			}
		case "inject":
			if injectSpan.SpanID == "" {
				injectSpan = e
			}
		}
	}

	if cycleSpan.SpanID == "" || injectSpan.SpanID == "" {
		t.Fatalf("missing cycle/inject spans: cycle=%q inject=%q", cycleSpan.SpanID, injectSpan.SpanID)
	}
	if injectSpan.ParentSpanID != cycleSpan.SpanID {
		t.Fatalf("inject span should be child of cycle_check span, parent=%q cycle=%q", injectSpan.ParentSpanID, cycleSpan.SpanID)
	}
	if injectSpan.TraceID != cycleSpan.TraceID {
		t.Fatalf("inject should keep trace id from cycle_check, inject=%q cycle=%q", injectSpan.TraceID, cycleSpan.TraceID)
	}
}

func TestInjectFunctionDoesNotCreateDedicatedSpan(t *testing.T) {
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

	var injectSpan dixtrace.Event
	injectFuncCount := 0
	for _, e := range res.Records {
		if e.Event != "span.start" {
			continue
		}
		if e.Operation == "inject" && injectSpan.SpanID == "" {
			injectSpan = e
		}
		if e.Operation == "inject.func" {
			injectFuncCount++
		}
	}

	if injectSpan.SpanID == "" {
		t.Fatalf("expected inject span")
	}
	if injectFuncCount != 0 {
		t.Fatalf("expected no inject.func span, got %d", injectFuncCount)
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

func TestInjectParamKeepsDeclaredMapAndSliceInputType(t *testing.T) {
	dixtrace.ResetForTest()

	type dep struct{}

	d := New()
	d.Provide(func() *dep { return &dep{} })

	if err := d.TryInject(func(map[string]*dep, []*dep) {}); err != nil {
		t.Fatalf("inject failed: %v", err)
	}

	res := dixtrace.QueryEvents(dixtrace.Query{Event: "span.start", Operation: "inject.param", Limit: 2000})
	if len(res.Records) == 0 {
		t.Fatalf("expected inject.param span.start events")
	}

	var seenMap, seenList bool
	for _, e := range res.Records {
		if e.Event != "span.start" || e.Operation != "inject.param" {
			continue
		}
		if e.Attrs == nil {
			continue
		}

		inputType, _ := e.Attrs["input_type"].(string)
		queryKind, _ := e.Attrs["query_kind"].(string)
		resolvedType, _ := e.Attrs["resolved_input_type"].(string)

		switch {
		case inputType == "map[string]*dixinternal.dep":
			seenMap = true
			if e.Component != inputType {
				t.Fatalf("map input component should keep declared type, got %q want %q", e.Component, inputType)
			}
			if queryKind != "map" {
				t.Fatalf("map input query_kind should be map, got %q", queryKind)
			}
			if resolvedType != "*dixinternal.dep" {
				t.Fatalf("map input resolved_input_type should be element type, got %q", resolvedType)
			}
		case inputType == "[]*dixinternal.dep":
			seenList = true
			if e.Component != inputType {
				t.Fatalf("slice input component should keep declared type, got %q want %q", e.Component, inputType)
			}
			if queryKind != "list" {
				t.Fatalf("slice input query_kind should be list, got %q", queryKind)
			}
			if resolvedType != "*dixinternal.dep" {
				t.Fatalf("slice input resolved_input_type should be element type, got %q", resolvedType)
			}
		}
	}

	if !seenMap || !seenList {
		t.Fatalf("expected map and slice inject.param spans, got map=%v list=%v", seenMap, seenList)
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

func TestResolveProviderSpanContainsProviderFunctionSignature(t *testing.T) {
	dixtrace.ResetForTest()

	type depA struct{}
	type depB struct{}

	d := New()
	d.Provide(func() *depA { return &depA{} })
	d.Provide(func(*depA) *depB { return &depB{} })

	if err := d.TryInject(func(*depB) {}); err != nil {
		t.Fatalf("inject failed: %v", err)
	}

	res := dixtrace.QueryEvents(dixtrace.Query{Event: "span.start", Operation: "resolve.provider", Limit: 4000})
	if len(res.Records) == 0 {
		t.Fatalf("expected resolve.provider span.start events")
	}

	found := false
	for _, e := range res.Records {
		if e.Event != "span.start" || e.Operation != "resolve.provider" {
			continue
		}
		if e.Component != "*dixinternal.depB" {
			continue
		}
		if e.Attrs == nil {
			t.Fatalf("resolve.provider attrs missing")
		}
		providerFn, _ := e.Attrs["provider_function"].(string)
		if providerFn == "" {
			t.Fatalf("expected provider_function in resolve.provider attrs for depB")
		}
		found = true
		break
	}

	if !found {
		t.Fatalf("expected resolve.provider span for *dixinternal.depB")
	}
}

func TestResolveProviderSpanContainsProviderFunctionListForMultiProviders(t *testing.T) {
	dixtrace.ResetForTest()

	type multiProviderSvc interface{}
	type implA struct{}
	type implB struct{}

	d := New()
	d.Provide(func() multiProviderSvc { return implA{} })
	d.Provide(func() multiProviderSvc { return implB{} })

	if err := d.TryInject(func([]multiProviderSvc) {}); err != nil {
		t.Fatalf("inject failed: %v", err)
	}

	res := dixtrace.QueryEvents(dixtrace.Query{Event: "span.start", Operation: "resolve.provider", Limit: 4000})
	if len(res.Records) == 0 {
		t.Fatalf("expected resolve.provider span.start events")
	}

	found := false
	for _, e := range res.Records {
		if e.Event != "span.start" || e.Operation != "resolve.provider" {
			continue
		}
		if e.Component != "dixinternal.multiProviderSvc" {
			continue
		}
		if e.Attrs == nil {
			t.Fatalf("resolve.provider attrs missing")
		}

		providerCount, ok := e.Attrs["provider_count"].(int)
		if !ok || providerCount != 2 {
			t.Fatalf("expected provider_count=2, got %#v", e.Attrs["provider_count"])
		}

		list, ok := e.Attrs["provider_functions"].([]string)
		if !ok {
			t.Fatalf("expected provider_functions []string, got %#v", e.Attrs["provider_functions"])
		}
		if len(list) != 2 {
			t.Fatalf("expected provider_functions len=2, got %d (%v)", len(list), list)
		}

		preview, _ := e.Attrs["provider_function"].(string)
		if preview == "" {
			t.Fatalf("expected provider_function preview for multi providers")
		}

		found = true
		break
	}

	if !found {
		t.Fatalf("expected resolve.provider span for dixinternal.multiProviderSvc")
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
