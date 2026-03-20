package dixinternal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"
)

// newDix creates a new Dix container instance
func newDix(opts ...Option) (d *Dix) {
	defer func() {
		if r := recover(); r != nil {
			maybePrintStack()
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("panic: %v", r)
			}
			panic(fmt.Errorf("failed to create new dix container with options: %w", err))
		}
	}()

	options := Options{
		AllowValuesNull:       true,
		ProviderTimeout:       DefaultProviderTimeout,
		SlowProviderThreshold: DefaultSlowProviderThreshold,
	}
	for _, opt := range opts {
		opt(&options)
	}

	if err := options.Validate(); err != nil {
		panic(err)
	}

	container := &Dix{
		option:        options,
		providers:     make(map[outputType][]*providerFn),
		objects:       make(map[outputType]map[group][]value),
		initializer:   make(map[reflect.Value]bool),
		providerStats: make(map[reflect.Value]*providerRuntimeStat),
		recentErrors:  make([]recentErrorRecord, 0, 16),
	}

	// Register the container itself
	container.provide(func() *Dix { return container })

	return container
}

type Dix struct {
	option        Options
	providers     map[outputType][]*providerFn
	objects       map[outputType]map[group][]value
	initializer   map[reflect.Value]bool
	providerStats map[reflect.Value]*providerRuntimeStat
	recentErrors  []recentErrorRecord
}

const maxRecentErrorRecords = 200

type providerRuntimeStat struct {
	FunctionName  string
	OutputType    string
	CallCount     int
	TotalDuration time.Duration
	LastDuration  time.Duration
	LastError     string
	LastRunAt     time.Time
}

type recentErrorRecord struct {
	Operation        string
	ErrorType        string
	Component        string
	Stage            string
	ProviderFunction string
	OutputType       string
	InputType        string
	InputTypes       []string
	Message          string
	RootCause        string
	Hint             string
	TimedOut         bool
	Duration         time.Duration
	Timeout          time.Duration
	Occurred         time.Time
}

type recentErrorContext struct {
	Stage            string
	ErrorType        string
	ProviderFunction string
	OutputType       string
	InputType        string
	InputTypes       []string
	RootCause        string
	Hint             string
	TimedOut         bool
	Duration         time.Duration
	Timeout          time.Duration
}

func (dix *Dix) Option() Options {
	return dix.option
}

// getOutputTypeValues retrieves or creates values for a specific output type
func (dix *Dix) getOutputTypeValues(outTyp outputType, opt Options) (map[group][]value, error) {
	logDITrace("resolve.type.start",
		"type", outTyp.String(),
		"kind", outTyp.Kind().String(),
		"provider_count", len(dix.providers[outTyp]),
	)

	// 1. Validate type kind
	switch outTyp.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func:
		// Valid types
	default:
		logDITrace("resolve.type.unsupported", "type", outTyp.String(), "kind", outTyp.Kind().String())
		return nil, fmt.Errorf("unsupported provider type kind: %s (kind=%s), supported: ptr, interface, func", outTyp, outTyp.Kind())
	}

	// 2. Check if providers exist
	if len(dix.providers[outTyp]) == 0 {
		logDITrace("resolve.type.provider_missing", "type", outTyp.String(), "kind", outTyp.Kind().String())
		logger.Warn("provider not found, please check imports or type definition", "type", outTyp.String(), "kind", outTyp.Kind().String())
	}

	// 3. Initialize object cache for this type if needed
	if dix.objects[outTyp] == nil {
		dix.objects[outTyp] = make(map[group][]value)
	}

	// 4. Iterate over providers and execute them if not already initialized
	for _, provider := range dix.providers[outTyp] {
		providerName := GetFnTraceName(provider.fn)
		if dix.initializer[provider.fn] {
			logDITrace("provider.skip.initialized", "provider", providerName, "output_type", outTyp.String())
			continue
		}

		logDITrace("provider.execute.dispatch",
			"provider", providerName,
			"output_type", outTyp.String(),
			"input_types", strings.Join(providerInputTypeNames(provider.inputList), ", "),
		)

		if err := dix.executeProvider(provider, outTyp, opt); err != nil {
			logDITrace("provider.execute.dispatch_failed", "provider", providerName, "output_type", outTyp.String(), "error", err)
			return nil, err
		}
	}

	logDITrace("resolve.type.done", "type", outTyp.String(), "group_count", len(dix.objects[outTyp]))

	return dix.objects[outTyp], nil
}

// executeProvider handles the execution of a single provider function
func (dix *Dix) executeProvider(p *providerFn, outTyp outputType, opt Options) error {
	fnName := GetFnName(p.fn)
	traceFnName := GetFnTraceName(p.fn)
	inputTypes := providerInputTypeNames(p.inputList)
	logDITrace("provider.execute.start",
		"provider", traceFnName,
		"output_type", outTyp.String(),
		"input_types", strings.Join(inputTypes, ", "),
	)

	// 1. Prepare inputs
	var inputs []reflect.Value
	for _, in := range p.inputList {
		logDITrace("provider.input.resolve.start",
			"provider", traceFnName,
			"output_type", outTyp.String(),
			"input_type", in.typ.String(),
			"query_kind", dependencyQueryKind(in.isMap, in.isList),
		)

		val, err := dix.getValue(in.typ, opt, in.isMap, in.isList, outTyp)
		if err != nil {
			logDITrace("provider.input.resolve.failed",
				"provider", traceFnName,
				"output_type", outTyp.String(),
				"input_type", in.typ.String(),
				"query_kind", dependencyQueryKind(in.isMap, in.isList),
				"error", err,
			)
			wrappedErr := fmt.Errorf("failed to get input value for provider: %w", err)
			dix.recordRecentErrorWithContext("provider_execute", fnName, wrappedErr, recentErrorContext{
				Stage:            "resolve_input",
				ProviderFunction: fnName,
				OutputType:       outTyp.String(),
				InputType:        in.typ.String(),
				InputTypes:       inputTypes,
				RootCause:        rootCauseMessage(err),
			})
			logger.Error("failed to get input value",
				"error", err,
				"error_type", buildErrorType("provider_execute", "resolve_input", false, wrappedErr.Error()),
				"provider", fnName,
				"output_type", outTyp.String(),
				"type", in.typ.String(),
				"kind", in.typ.Kind().String(),
				"map", in.isMap,
				"list", in.isList,
				"root_cause", rootCauseMessage(err),
				"hint", buildErrorHint("provider_execute", "resolve_input", false),
			)
			return wrappedErr
		}
		logDITrace("provider.input.resolve.found",
			"provider", traceFnName,
			"input_type", in.typ.String(),
			"query_kind", dependencyQueryKind(in.isMap, in.isList),
		)
		inputs = append(inputs, val)
	}

	// 2. Call provider function
	start := time.Now()

	logDITrace("provider.call.start", "provider", traceFnName, "output_type", outTyp.String(), "timeout", opt.ProviderTimeout.String())
	logger.Debug("evaluating provider", "provider", fnName)

	outputs, err, timedOut := p.callWithTimeout(inputs, opt.ProviderTimeout)
	duration := time.Since(start)
	if err != nil {
		logDITrace("provider.call.failed",
			"provider", traceFnName,
			"output_type", outTyp.String(),
			"timed_out", timedOut,
			"duration", duration.String(),
			"error", err,
		)
		wrappedErr := fmt.Errorf("provider call failed for %s: %w", fnName, err)
		dix.recordProviderStat(p, duration, err)
		dix.recordRecentErrorWithContext("provider_execute", fnName, wrappedErr, recentErrorContext{
			Stage:            "call",
			ProviderFunction: fnName,
			OutputType:       outTyp.String(),
			InputTypes:       inputTypes,
			RootCause:        rootCauseMessage(err),
			TimedOut:         timedOut,
			Duration:         duration,
			Timeout:          opt.ProviderTimeout,
		})
		if timedOut {
			logger.Error("provider execution timeout",
				"error_type", buildErrorType("provider_execute", "call", true, wrappedErr.Error()),
				"provider", fnName,
				"output_type", outTyp.String(),
				"input_types", strings.Join(inputTypes, ", "),
				"timeout", opt.ProviderTimeout.String(),
				"duration", duration.String(),
				"hint", buildErrorHint("provider_execute", "call", true),
			)
		}
		return wrappedErr
	}

	// 3. Check for error return
	if p.hasError && len(outputs) > 1 && !outputs[1].IsNil() {
		if err, ok := outputs[1].Interface().(error); ok && err != nil {
			logDITrace("provider.call.return_error", "provider", traceFnName, "output_type", outTyp.String(), "duration", duration.String(), "error", err)
			wrappedErr := fmt.Errorf("provider execution failed: %s: %w", fnName, err)
			dix.recordProviderStat(p, duration, err)
			dix.recordRecentErrorWithContext("provider_execute", fnName, wrappedErr, recentErrorContext{
				Stage:            "return_error",
				ProviderFunction: fnName,
				OutputType:       outTyp.String(),
				InputTypes:       inputTypes,
				RootCause:        rootCauseMessage(err),
				Duration:         duration,
				Timeout:          opt.ProviderTimeout,
			})
			logger.Error("provider returned error",
				"error_type", buildErrorType("provider_execute", "return_error", false, wrappedErr.Error()),
				"provider", fnName,
				"output_type", outTyp.String(),
				"input_types", strings.Join(inputTypes, ", "),
				"duration", duration.String(),
				"error", err,
				"root_cause", rootCauseMessage(err),
				"hint", buildErrorHint("provider_execute", "return_error", false),
			)
			return wrappedErr
		}
	}

	dix.initializer[p.fn] = true
	dix.recordProviderStat(p, duration, nil)
	if opt.SlowProviderThreshold > 0 && duration > opt.SlowProviderThreshold {
		logger.Warn("slow provider execution detected",
			"provider", fnName,
			"duration", duration.String(),
			"threshold", opt.SlowProviderThreshold.String(),
		)
	}
	logger.Debug("provider evaluated successfully", "duration", duration.String(), "provider", fnName)
	logDITrace("provider.call.done", "provider", traceFnName, "output_type", outTyp.String(), "duration", duration.String())

	// 4. Process output values and update cache
	dix.processProviderOutput(outTyp, p, outputs[0])
	logDITrace("provider.output.cached", "provider", traceFnName, "output_type", outTyp.String())
	return nil
}

// processProviderOutput handles the result of a provider and updates the object cache
func (dix *Dix) processProviderOutput(requestedType outputType, p *providerFn, outputVal reflect.Value) {
	// Parse the output value into groups
	newObjects := handleOutput(requestedType, outputVal)

	// Check for duplicate map keys if applicable
	if p.output.isMap {
		for outT := range newObjects {
			if _, exists := dix.objects[outT]; exists {
				logger.Info("provider value already exists for type", "type", requestedType.String(), "key", outT.String())
			}
		}
	}

	// Merge new objects into the main cache
	for typeKey, groups := range newObjects {
		if dix.objects[typeKey] == nil {
			dix.objects[typeKey] = make(map[group][]value)
		}

		for groupKey, values := range groups {
			dix.objects[typeKey][groupKey] = append(dix.objects[typeKey][groupKey], values...)
		}
	}
}

func (dix *Dix) recordProviderStat(p *providerFn, duration time.Duration, err error) {
	if p == nil {
		return
	}

	stat, ok := dix.providerStats[p.fn]
	if !ok {
		outputType := ""
		if p.output != nil && p.output.typ != nil {
			outputType = p.output.typ.String()
		}
		stat = &providerRuntimeStat{
			FunctionName: GetFnName(p.fn),
			OutputType:   outputType,
		}
		dix.providerStats[p.fn] = stat
	}

	stat.CallCount++
	stat.TotalDuration += duration
	stat.LastDuration = duration
	stat.LastRunAt = time.Now()
	if err != nil {
		stat.LastError = err.Error()
	} else {
		stat.LastError = ""
	}
}

func (dix *Dix) recordRecentErrorWithContext(operation, component string, err error, ctx recentErrorContext) {
	if err == nil {
		return
	}

	root := ctx.RootCause
	if root == "" {
		root = rootCauseMessage(err)
	}

	hint := strings.TrimSpace(ctx.Hint)
	if hint == "" {
		hint = buildErrorHint(operation, ctx.Stage, ctx.TimedOut)
	}

	errorType := strings.TrimSpace(ctx.ErrorType)
	if errorType == "" {
		errorType = buildErrorType(operation, ctx.Stage, ctx.TimedOut, err.Error())
	}

	record := recentErrorRecord{
		Operation:        operation,
		ErrorType:        errorType,
		Component:        component,
		Stage:            ctx.Stage,
		ProviderFunction: ctx.ProviderFunction,
		OutputType:       ctx.OutputType,
		InputType:        ctx.InputType,
		InputTypes:       append([]string{}, ctx.InputTypes...),
		Message:          err.Error(),
		RootCause:        root,
		Hint:             hint,
		TimedOut:         ctx.TimedOut,
		Duration:         ctx.Duration,
		Timeout:          ctx.Timeout,
		Occurred:         time.Now(),
	}

	dix.recentErrors = append(dix.recentErrors, record)
	emitDiagFileErrorRecord(record)
	emitLLMDiagnosticLine(record)

	if len(dix.recentErrors) > maxRecentErrorRecords {
		dix.recentErrors = dix.recentErrors[len(dix.recentErrors)-maxRecentErrorRecords:]
	}
}

func emitLLMDiagnosticLine(record recentErrorRecord) {
	payloadBody := struct {
		Operation          string   `json:"operation"`
		ErrorType          string   `json:"error_type"`
		Stage              string   `json:"stage,omitempty"`
		Component          string   `json:"component,omitempty"`
		ProviderFunction   string   `json:"provider_function,omitempty"`
		OutputType         string   `json:"output_type,omitempty"`
		InputType          string   `json:"input_type,omitempty"`
		InputTypes         []string `json:"input_types,omitempty"`
		Message            string   `json:"message"`
		RootCause          string   `json:"root_cause,omitempty"`
		Hint               string   `json:"hint,omitempty"`
		TimedOut           bool     `json:"timed_out,omitempty"`
		DurationNs         int64    `json:"duration_ns,omitempty"`
		TimeoutNs          int64    `json:"timeout_ns,omitempty"`
		OccurredAtUnixNano int64    `json:"occurred_at_unix_nano"`
	}{
		Operation:          record.Operation,
		ErrorType:          record.ErrorType,
		Stage:              record.Stage,
		Component:          record.Component,
		ProviderFunction:   record.ProviderFunction,
		OutputType:         record.OutputType,
		InputType:          record.InputType,
		InputTypes:         append([]string{}, record.InputTypes...),
		Message:            record.Message,
		RootCause:          record.RootCause,
		Hint:               record.Hint,
		TimedOut:           record.TimedOut,
		DurationNs:         int64(record.Duration),
		TimeoutNs:          int64(record.Timeout),
		OccurredAtUnixNano: record.Occurred.UnixNano(),
	}

	emitDiagFileLLMRecord(payloadBody)

	if !shouldEmitLLMDiagnosticLine() {
		return
	}

	payload, err := json.Marshal(payloadBody)
	if err != nil {
		if logger != nil {
			logger.Warn("failed to marshal llm diagnostic payload", "error", err)
		}
		return
	}

	if _, err := fmt.Fprintf(os.Stderr, "DIX_LLM_DIAG %s\n", payload); err != nil {
		if logger != nil {
			logger.Warn("failed to write llm diagnostic line", "error", err)
		}
	}
}

func providerInputTypeNames(inputs []*providerInputType) []string {
	names := make([]string, 0, len(inputs))
	for _, in := range inputs {
		if in == nil || in.typ == nil {
			continue
		}
		names = append(names, in.typ.String())
	}
	return names
}

func dependencyQueryKind(isMap, isList bool) string {
	switch {
	case isMap && isList:
		return "map_list"
	case isMap:
		return "map"
	case isList:
		return "list"
	default:
		return "single"
	}
}

func parentTypeChain(parents []reflect.Type) string {
	if len(parents) == 0 {
		return ""
	}
	items := make([]string, 0, len(parents))
	for _, p := range parents {
		if p == nil {
			continue
		}
		items = append(items, p.String())
	}
	return strings.Join(items, " -> ")
}

func rootCauseMessage(err error) string {
	if err == nil {
		return ""
	}
	root := err
	for {
		next := errors.Unwrap(root)
		if next == nil {
			break
		}
		root = next
	}
	if root == err {
		return ""
	}
	return root.Error()
}

func buildErrorHint(operation, stage string, timedOut bool) string {
	if timedOut {
		return "provider 初始化超时：可先优化初始化逻辑，或临时调大 WithProviderTimeout；排查时可用 /api/errors 与 /api/runtime-stats 联动定位"
	}

	switch operation {
	case "provide", "try_provide":
		return "检查 provider 签名（必须是函数且返回值数量合法）；若不希望中断启动，请优先用 TryProvide 并继续启动 Web 诊断页"
	case "inject", "try_inject":
		switch stage {
		case "cycle_check":
			return "检测到循环依赖：请拆分相互引用的组件，或改为接口/延迟注入；可用 /api/dependencies 观察依赖环"
		default:
			return "注入失败：先确认依赖是否已注册、类型是否一致；生产路径建议改用 TryInject 避免进程直接退出"
		}
	case "provider_execute":
		switch stage {
		case "resolve_input":
			return "provider 输入依赖未解析：检查对应输入类型是否有 provider、命名空间(map/list)是否匹配、是否遗漏导入"
		case "return_error":
			return "provider 主动返回 error：优先检查外部资源可用性（DB/Redis/HTTP）与配置参数；必要时在 provider 内增加上下文日志"
		default:
			return "provider 执行失败：查看 provider_function、output_type 与 input_types 定位具体构造链路"
		}
	default:
		return "可先查看 /api/errors 获取最近错误详情，并结合 /api/runtime-stats 定位慢/失败 provider"
	}
}

func buildErrorType(operation, stage string, timedOut bool, message string) string {
	msg := strings.ToLower(strings.TrimSpace(message))

	if timedOut {
		return "provider_timeout"
	}

	switch operation {
	case "provide", "try_provide":
		return "provider_registration_invalid"
	case "inject", "try_inject":
		switch stage {
		case "cycle_check":
			return "dependency_cycle"
		case "inject":
			if strings.Contains(msg, "injected function returned error") {
				return "inject_callback_error"
			}
			if strings.Contains(msg, "value not found") || strings.Contains(msg, "failed to get value") {
				return "inject_dependency_missing"
			}
			return "inject_failed"
		default:
			return "inject_failed"
		}
	case "provider_execute":
		switch stage {
		case "resolve_input":
			return "provider_input_unresolved"
		case "return_error":
			return "provider_return_error"
		case "call":
			if strings.Contains(msg, "panic") {
				return "provider_panic"
			}
			return "provider_call_failed"
		default:
			return "provider_call_failed"
		}
	default:
		return "unknown_error"
	}
}

func describeComponent(param any) string {
	if param == nil {
		return "<nil>"
	}
	typ := reflect.TypeOf(param)
	if typ == nil {
		return "<nil>"
	}
	return typ.String()
}

func (dix *Dix) getProviderStack(typ reflect.Type) []string {
	var stacks []string
	for _, n := range dix.providers[typ] {
		stacks = append(stacks, GetFnName(n.fn))
	}
	return stacks
}

// getValue retrieves a value for a dependency, handling recursion for structs
func (dix *Dix) getValue(typ reflect.Type, opt Options, isMap, isList bool, parents ...reflect.Type) (reflect.Value, error) {
	logDITrace("resolve.value.start",
		"type", typ.String(),
		"kind", typ.Kind().String(),
		"query_kind", dependencyQueryKind(isMap, isList),
		"parents", parentTypeChain(parents),
	)

	// If it's a struct, we inject into a new instance
	if typ.Kind() == reflect.Struct {
		logDITrace("resolve.value.struct_inject.start", "type", typ.String())
		v := reflect.New(typ).Elem()
		if err := dix.injectStruct(v, opt); err != nil {
			logDITrace("resolve.value.struct_inject.failed", "type", typ.String(), "error", err)
			return reflect.Value{}, err
		}
		logDITrace("resolve.value.struct_inject.done", "type", typ.String())
		return v, nil
	}

	// Otherwise, resolve from providers
	logDITrace("resolve.value.search_provider.start", "type", typ.String(), "query_kind", dependencyQueryKind(isMap, isList))
	valMap, err := dix.getOutputTypeValues(typ, opt)
	if err != nil {
		logDITrace("resolve.value.search_provider.failed", "type", typ.String(), "error", err)
		return reflect.Value{}, err
	}

	// Handle Map injection
	if isMap {
		if !opt.AllowValuesNull && len(valMap) == 0 {
			logDITrace("resolve.value.not_found", "type", typ.String(), "query_kind", dependencyQueryKind(isMap, isList), "reason", "map_empty")
			return reflect.Value{}, fmt.Errorf("value not found for map injection: type=%s options=%v providers=%v parents=%v",
				typ, opt, dix.getProviderStack(typ), parents)
		}
		logDITrace("resolve.value.found", "type", typ.String(), "query_kind", dependencyQueryKind(isMap, isList), "group_count", len(valMap))
		return makeMap(typ, valMap, isList), nil
	}

	// Handle List injection
	if isList {
		if !opt.AllowValuesNull && len(valMap[defaultKey]) == 0 {
			logDITrace("resolve.value.not_found", "type", typ.String(), "query_kind", dependencyQueryKind(isMap, isList), "reason", "list_empty")
			return reflect.Value{}, dix.createNotFoundError(typ, valMap, parents, opt, "list value not found")
		}
		logDITrace("resolve.value.found", "type", typ.String(), "query_kind", dependencyQueryKind(isMap, isList), "value_count", len(valMap[defaultKey]))
		return makeList(typ, valMap[defaultKey]), nil
	}

	// Handle Single Value injection
	valList, ok := valMap[defaultKey]
	if !ok || len(valList) == 0 {
		logDITrace("resolve.value.not_found", "type", typ.String(), "query_kind", dependencyQueryKind(isMap, isList), "reason", "single_empty")
		return reflect.Value{}, dix.createNotFoundError(typ, valMap, parents, opt, "value not found")
	}

	// Use the last provided value
	val := valList[len(valList)-1]
	if val.IsZero() {
		logDITrace("resolve.value.not_found", "type", typ.String(), "query_kind", dependencyQueryKind(isMap, isList), "reason", "single_zero")
		return reflect.Value{}, dix.createNotFoundError(typ, valMap, parents, opt, "value is zero/nil")
	}

	logDITrace("resolve.value.found", "type", typ.String(), "query_kind", dependencyQueryKind(isMap, isList), "value_count", len(valList))

	return val, nil
}

func (dix *Dix) createNotFoundError(typ reflect.Type, valMap map[group][]value, parents []reflect.Type, opt Options, msg string) error {
	return fmt.Errorf("%s: type=%s kind=%s values=%v providers=%v parents=%q options=%v",
		msg,
		typ.String(),
		typ.Kind().String(),
		valMap,
		dix.getProviderStack(typ),
		parents,
		opt,
	)
}

// injectFunc injects dependencies into a function and executes it
func (dix *Dix) injectFunc(fnVal reflect.Value, opt Options) (err error) {
	traceFnName := GetFnTraceName(fnVal)
	logDITrace("inject.func.start", "function", traceFnName)

	defer func() {
		if r := recover(); r != nil {
			maybePrintStack()
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("panic: %v", r)
			}
			logDITrace("inject.func.panic", "function", traceFnName, "error", err)
		}
	}()

	fnType := fnVal.Type()
	if fnType.NumOut() > 1 {
		logDITrace("inject.func.invalid", "function", traceFnName, "reason", "too_many_outputs", "outputs", fnType.NumOut())
		return errors.New("injected function output count must be <= 1")
	}
	if fnType.NumIn() == 0 {
		logDITrace("inject.func.invalid", "function", traceFnName, "reason", "no_inputs")
		return errors.New("injected function input count must be > 0")
	}

	// Check return type if exists
	hasErrorReturn := false
	if fnType.NumOut() == 1 {
		outType := fnType.Out(0)
		if !outType.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			logDITrace("inject.func.invalid", "function", traceFnName, "reason", "non_error_return", "return_type", outType.String())
			return fmt.Errorf("injected function return type must be error, but got %s", outType)
		}
		hasErrorReturn = true
	}

	// Prepare inputs
	var inputs []reflect.Value
	for i := 0; i < fnType.NumIn(); i++ {
		inType := fnType.In(i)
		inputTypeInfo := dix.analyzeInputType(inType)
		logDITrace("inject.func.resolve_input.start",
			"function", traceFnName,
			"index", i,
			"input_type", inputTypeInfo.typ.String(),
			"query_kind", dependencyQueryKind(inputTypeInfo.isMap, inputTypeInfo.isList),
		)

		val, err := dix.getValue(inputTypeInfo.typ, opt, inputTypeInfo.isMap, inputTypeInfo.isList, fnType)
		if err != nil {
			logDITrace("inject.func.resolve_input.failed", "function", traceFnName, "index", i, "input_type", inputTypeInfo.typ.String(), "error", err)
			return err
		}
		logDITrace("inject.func.resolve_input.done", "function", traceFnName, "index", i, "input_type", inputTypeInfo.typ.String())
		inputs = append(inputs, val)
	}

	// Execute
	logDITrace("inject.func.call.start", "function", traceFnName, "input_count", len(inputs))
	results := fnVal.Call(inputs)
	logDITrace("inject.func.call.done", "function", traceFnName, "result_count", len(results))

	// Handle error return
	if hasErrorReturn && len(results) > 0 && !results[0].IsNil() {
		if err, ok := results[0].Interface().(error); ok {
			logDITrace("inject.func.call.return_error", "function", traceFnName, "error", err)
			return fmt.Errorf("injected function returned error: %w", err)
		}
	}
	logDITrace("inject.func.done", "function", traceFnName)
	return nil
}

// analyzeInputType analyzes the input type and returns its metadata
// This is a wrapper around parseInputType for backward compatibility
func (dix *Dix) analyzeInputType(inType reflect.Type) *providerInputType {
	inputs := parseInputType(inType)
	if len(inputs) > 0 {
		return inputs[0]
	}
	return &providerInputType{typ: inType}
}

// injectStruct injects dependencies into struct fields
func (dix *Dix) injectStruct(structVal reflect.Value, opt Options) error {
	structType := structVal.Type()
	logDITrace("inject.struct.start", "struct_type", structType.String(), "field_count", structType.NumField())
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldVal := structVal.Field(i)

		// Skip unexported fields or fields we can't set.
		if !fieldVal.CanSet() {
			logDITrace("inject.struct.field.skip", "struct_type", structType.String(), "field", field.Name, "reason", "cannot_set")
			continue
		}

		var val reflect.Value
		var err error

		switch field.Type.Kind() {
		case reflect.Struct:
			logDITrace("inject.struct.field.resolve.start", "struct_type", structType.String(), "field", field.Name, "field_type", field.Type.String(), "query_kind", "struct")
			// Recursively inject into nested structs
			if err := dix.injectStruct(fieldVal, opt); err != nil {
				logDITrace("inject.struct.field.resolve.failed", "struct_type", structType.String(), "field", field.Name, "error", err)
				return err
			}
			logDITrace("inject.struct.field.resolve.done", "struct_type", structType.String(), "field", field.Name, "field_type", field.Type.String(), "query_kind", "struct")
			continue // Done for this field
		case reflect.Interface, reflect.Pointer, reflect.Func:
			logDITrace("inject.struct.field.resolve.start", "struct_type", structType.String(), "field", field.Name, "field_type", field.Type.String(), "query_kind", "single")
			val, err = dix.getValue(field.Type, opt, false, false, structType)
		case reflect.Map:
			elemType := field.Type.Elem()
			isList := elemType.Kind() == reflect.Slice
			if isList {
				elemType = elemType.Elem()
			}
			logDITrace("inject.struct.field.resolve.start", "struct_type", structType.String(), "field", field.Name, "field_type", field.Type.String(), "query_kind", dependencyQueryKind(true, isList), "lookup_type", elemType.String())
			val, err = dix.getValue(elemType, opt, true, isList, structType)
		case reflect.Slice:
			logDITrace("inject.struct.field.resolve.start", "struct_type", structType.String(), "field", field.Name, "field_type", field.Type.String(), "query_kind", "list", "lookup_type", field.Type.Elem().String())
			val, err = dix.getValue(field.Type.Elem(), opt, false, true, structType)
		default:
			// We do not inject into basic types, so we just continue.
			logDITrace("inject.struct.field.skip", "struct_type", structType.String(), "field", field.Name, "field_type", field.Type.String(), "reason", "unsupported_kind")
			logger.Debug("skipping basic type injection", "field", field.Name)
			continue
		}

		if err != nil {
			logDITrace("inject.struct.field.resolve.failed", "struct_type", structType.String(), "field", field.Name, "field_type", field.Type.String(), "error", err)
			return fmt.Errorf("failed to get value for field %s: %w", field.Name, err)
		}

		if fieldVal.CanSet() {
			fieldVal.Set(val)
			logDITrace("inject.struct.field.resolve.done", "struct_type", structType.String(), "field", field.Name, "field_type", field.Type.String())
		}
	}
	logDITrace("inject.struct.done", "struct_type", structType.String())
	return nil
}

// inject is the entry point for dependency injection
// NOTE: This method is NOT thread-safe by itself. Use Inject() or TryInject() which handle locking.
func (dix *Dix) inject(param any, opts ...Option) (err error) {
	paramType := "<nil>"
	if typ := reflect.TypeOf(param); typ != nil {
		paramType = typ.String()
	}
	logDITrace("inject.start", "component", describeComponent(param), "param_type", paramType)

	defer func() {
		if r := recover(); r != nil {
			maybePrintStack()
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("panic: %v", r)
			}
			logDITrace("inject.panic", "component", describeComponent(param), "param_type", paramType, "error", err)
			logger.Error("injection failed with panic", "error", err, "param", fmt.Sprintf("%+v", param))
		}
	}()

	if param == nil {
		logDITrace("inject.invalid", "reason", "nil_param")
		return errors.New("nil injection parameter")
	}

	// Merge options
	opt := dix.option
	for _, o := range opts {
		o(&opt)
	}
	opt = dix.option.Merge(opt)

	val := reflect.ValueOf(param)
	if !val.IsValid() || val.IsNil() {
		logDITrace("inject.invalid", "component", describeComponent(param), "reason", "invalid_or_nil_reflect_value")
		return fmt.Errorf("param must be valid and non-nil, but got %v", param)
	}

	// Handle Function Injection
	if val.Kind() == reflect.Func {
		logDITrace("inject.route", "component", describeComponent(param), "route", "function")
		return dix.injectFunc(val, opt)
	}

	// Handle Struct Pointer Injection
	if val.Kind() != reflect.Ptr {
		logDITrace("inject.invalid", "component", describeComponent(param), "reason", "not_pointer", "kind", val.Kind().String())
		return fmt.Errorf("param must be a pointer, but got %T", param)
	}

	// Inject into methods marked with prefix
	logDITrace("inject.methods.scan", "component", describeComponent(param), "method_count", val.NumMethod(), "prefix", InjectMethodPrefix)
	for i := 0; i < val.NumMethod(); i++ {
		method := val.Type().Method(i)
		if strings.HasPrefix(method.Name, InjectMethodPrefix) {
			logDITrace("inject.method.start", "component", describeComponent(param), "method", method.Name)
			if err := dix.injectFunc(val.Method(i), opt); err != nil {
				logDITrace("inject.method.failed", "component", describeComponent(param), "method", method.Name, "error", err)
				return err
			}
			logDITrace("inject.method.done", "component", describeComponent(param), "method", method.Name)
		}
	}

	// Unwrap pointer to get struct
	elem := val
	for elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}

	if elem.Kind() != reflect.Struct {
		logDITrace("inject.invalid", "component", describeComponent(param), "reason", "not_struct_pointer", "kind", elem.Kind().String())
		return fmt.Errorf("param must point to a struct, but got %T", param)
	}

	logDITrace("inject.route", "component", describeComponent(param), "route", "struct", "struct_type", elem.Type().String())
	return dix.injectStruct(elem, opt)
}

// handleProvide registers a provider function for a specific output type
func (dix *Dix) handleProvide(fnVal reflect.Value, outType reflect.Type, inputs []*providerInputType) error {
	traceFnName := GetFnTraceName(fnVal)
	logDITrace("provide.register.start",
		"provider", traceFnName,
		"declared_output_type", outType.String(),
		"declared_output_kind", outType.Kind().String(),
		"input_types", strings.Join(providerInputTypeNames(inputs), ", "),
	)

	// Check for error return value
	hasError := false
	if fnVal.Type().NumOut() == 2 {
		errorType := fnVal.Type().Out(1)
		if errorType.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			hasError = true
			logDITrace("provide.register.return_error.enabled", "provider", traceFnName, "error_type", errorType.String())
		} else {
			logDITrace("provide.register.failed", "provider", traceFnName, "declared_output_type", outType.String(), "reason", "second_return_not_error", "second_return", errorType.String())
			return fmt.Errorf("second return value must be error type, but got %s for fn %v", errorType, fnVal)
		}
	}

	provider := &providerFn{fn: fnVal, inputList: inputs, hasError: hasError}

	// Register based on output kind
	switch outType.Kind() {
	case reflect.Slice:
		provider.output = &providerOutputType{isList: true, typ: outType.Elem()}
		dix.providers[provider.output.typ] = append(dix.providers[provider.output.typ], provider)
		logDITrace("provide.register.output.done",
			"provider", traceFnName,
			"declared_output_type", outType.String(),
			"registered_output_type", provider.output.typ.String(),
			"query_kind", dependencyQueryKind(false, true),
		)

	case reflect.Map:
		elemType := outType.Elem()
		isList := false
		if elemType.Kind() == reflect.Slice {
			isList = true
			elemType = elemType.Elem()
		}
		provider.output = &providerOutputType{isMap: true, isList: isList, typ: elemType}
		dix.providers[provider.output.typ] = append(dix.providers[provider.output.typ], provider)
		logDITrace("provide.register.output.done",
			"provider", traceFnName,
			"declared_output_type", outType.String(),
			"registered_output_type", provider.output.typ.String(),
			"query_kind", dependencyQueryKind(true, isList),
		)

	case reflect.Ptr, reflect.Interface, reflect.Func:
		provider.output = &providerOutputType{typ: outType}
		dix.providers[provider.output.typ] = append(dix.providers[provider.output.typ], provider)
		logDITrace("provide.register.output.done",
			"provider", traceFnName,
			"declared_output_type", outType.String(),
			"registered_output_type", provider.output.typ.String(),
			"query_kind", dependencyQueryKind(false, false),
		)

	case reflect.Struct:
		// Recursively register exported fields of the struct
		for i := 0; i < outType.NumField(); i++ {
			field := outType.Field(i)
			if !field.IsExported() {
				logDITrace("provide.register.struct_field.skip", "provider", traceFnName, "declared_output_type", outType.String(), "field", field.Name, "field_type", field.Type.String(), "reason", "unexported")
				continue
			}
			if !isSupportedType(field.Type) {
				logDITrace("provide.register.struct_field.skip", "provider", traceFnName, "declared_output_type", outType.String(), "field", field.Name, "field_type", field.Type.String(), "reason", "unsupported_kind")
				continue
			}

			logDITrace("provide.register.struct_field.start", "provider", traceFnName, "declared_output_type", outType.String(), "field", field.Name, "field_type", field.Type.String())

			// Recursive call
			if err := dix.handleProvide(fnVal, field.Type, inputs); err != nil {
				logDITrace("provide.register.struct_field.failed", "provider", traceFnName, "declared_output_type", outType.String(), "field", field.Name, "field_type", field.Type.String(), "error", err)
				return err
			}
			logDITrace("provide.register.struct_field.done", "provider", traceFnName, "declared_output_type", outType.String(), "field", field.Name, "field_type", field.Type.String())
		}
		logDITrace("provide.register.output.done", "provider", traceFnName, "declared_output_type", outType.String(), "registered_output_type", outType.String(), "query_kind", "struct")

	default:
		logDITrace("provide.register.output.unsupported", "provider", traceFnName, "declared_output_type", outType.String(), "declared_output_kind", outType.Kind().String())
		logger.Warn("unsupported output type", "type", outType.String(), "kind", outType.Kind().String(), "provider", fnVal)
	}
	logDITrace("provide.register.done", "provider", traceFnName, "declared_output_type", outType.String())
	return nil
}

// getProvideInput is a wrapper for parseInputType used during provider registration
func (dix *Dix) getProvideInput(typ reflect.Type) []*providerInputType {
	return parseInputType(typ)
}

// parseInputType parses a reflect.Type and returns the corresponding providerInputType(s)
// This is the unified implementation for input type analysis
func parseInputType(typ reflect.Type) []*providerInputType {
	var input []*providerInputType
	switch typ.Kind() {
	case reflect.Interface, reflect.Ptr, reflect.Func:
		input = append(input, &providerInputType{typ: typ})
	case reflect.Struct:
		input = append(input, &providerInputType{typ: typ, isStruct: true})
	case reflect.Map:
		elemType := typ.Elem()
		isList := elemType.Kind() == reflect.Slice
		if isList {
			elemType = elemType.Elem()
		}
		input = append(input, &providerInputType{typ: elemType, isMap: true, isList: isList})
	case reflect.Slice:
		input = append(input, &providerInputType{typ: typ.Elem(), isList: true})
	default:
		logger.Warn("unsupported input type", "type", typ.String(), "kind", typ.Kind().String())
	}
	return input
}

// provide registers a constructor function
// NOTE: This method is NOT thread-safe by itself. Use Provide() or TryProvide() which handle locking.
func (dix *Dix) provide(param any) {
	component := describeComponent(param)
	logDITrace("provide.start", "component", component)

	defer func() {
		if r := recover(); r != nil {
			maybePrintStack()
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("panic: %v", r)
			}
			logDITrace("provide.panic", "component", component, "error", err)
			panic(fmt.Errorf("failed to provide param (%v): %w", param, err))
		}
	}()

	if param == nil {
		logDITrace("provide.invalid", "component", component, "reason", "nil_param")
		panic("param cannot be nil")
	}

	fnVal := reflect.ValueOf(param)
	if !fnVal.IsValid() || fnVal.IsZero() {
		logDITrace("provide.invalid", "component", component, "reason", "invalid_param")
		panic("param must be valid")
	}
	if fnVal.Kind() != reflect.Func {
		logDITrace("provide.invalid", "component", component, "reason", "not_function", "kind", fnVal.Kind().String())
		panic("param must be a function")
	}

	typ := fnVal.Type()
	traceFnName := GetFnTraceName(fnVal)
	logDITrace("provide.signature",
		"component", component,
		"provider", traceFnName,
		"input_count", typ.NumIn(),
		"output_count", typ.NumOut(),
		"variadic", typ.IsVariadic(),
	)
	if typ.IsVariadic() {
		logDITrace("provide.invalid", "component", component, "provider", traceFnName, "reason", "variadic_not_supported")
		panic("variadic functions are not supported")
	}
	if typ.NumOut() == 0 {
		logDITrace("provide.invalid", "component", component, "provider", traceFnName, "reason", "no_output")
		panic("provider function must return at least one value")
	}
	if typ.NumOut() > 2 {
		logDITrace("provide.invalid", "component", component, "provider", traceFnName, "reason", "too_many_outputs", "output_count", typ.NumOut())
		panic("provider function cannot return more than two values")
	}

	var inputs []*providerInputType
	for i := 0; i < typ.NumIn(); i++ {
		inType := typ.In(i)
		logDITrace("provide.input.analyze.start", "provider", traceFnName, "index", i, "input_type", inType.String())
		parsedInputs := dix.getProvideInput(inType)
		if len(parsedInputs) == 0 {
			logDITrace("provide.input.analyze.unsupported", "provider", traceFnName, "index", i, "input_type", inType.String())
		}
		for _, input := range parsedInputs {
			if input == nil || input.typ == nil {
				continue
			}
			logDITrace("provide.input.analyze.done",
				"provider", traceFnName,
				"index", i,
				"input_type", inType.String(),
				"resolved_type", input.typ.String(),
				"query_kind", dependencyQueryKind(input.isMap, input.isList),
				"is_struct", input.isStruct,
			)
		}
		inputs = append(inputs, parsedInputs...)
	}

	if err := dix.handleProvide(fnVal, typ.Out(0), inputs); err != nil {
		logDITrace("provide.register.failed", "provider", traceFnName, "declared_output_type", typ.Out(0).String(), "error", err)
		panic(err)
	}

	logDITrace("provide.done", "component", component, "provider", traceFnName)
}
