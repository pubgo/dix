package dixinternal

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"time"
)

// New Dix new
func New(opts ...Option) *Dix {
	return newDix(opts...)
}

// Provide registers a provider function. Panics on error.
// NOTE: Dix container is not thread-safe. Do not call Provide/Inject concurrently on the same container.
func (dix *Dix) Provide(param any) {
	component := describeComponent(param)
	defer func() {
		if r := recover(); r != nil {
			err := normalizeRecoveredError(r)
			logger.Error("provide failed", "component", component, "error", err, "root_cause", rootCauseMessage(err))
			dix.recordRecentErrorWithContext("provide", component, err, recentErrorContext{
				Stage:     "registration",
				RootCause: rootCauseMessage(err),
			})
			panic(err)
		}
	}()
	dix.provide(param)
}

// TryProvide registers a provider function. Returns error instead of panicking.
// NOTE: Dix container is not thread-safe. Do not call Provide/Inject concurrently on the same container.
func (dix *Dix) TryProvide(param any) (err error) {
	component := describeComponent(param)
	defer func() {
		if r := recover(); r != nil {
			err = normalizeRecoveredError(r)
			logger.Warn("try provide failed", "component", component, "error", err, "root_cause", rootCauseMessage(err))
			dix.recordRecentErrorWithContext("try_provide", component, err, recentErrorContext{
				Stage:     "registration",
				RootCause: rootCauseMessage(err),
			})
		}
	}()
	dix.provide(param)
	return nil
}

// Inject injects dependencies into the given parameter. Panics on error.
// NOTE: Dix container is not thread-safe. Do not call Provide/Inject concurrently on the same container.
func (dix *Dix) Inject(param any, opts ...Option) any {
	component := describeComponent(param)
	if dep, ok := dix.isCycle(); ok {
		err := errors.New("circular dependency: " + dep)
		logger.Error("dependency cycle detected", "cycle_path", dep, "component", component)
		dix.recordRecentErrorWithContext("inject", component, err, recentErrorContext{
			Stage:     "cycle_check",
			RootCause: rootCauseMessage(err),
		})
		panic(err)
	}

	if err := dix.inject(param, opts...); err != nil {
		logger.Error("inject failed", "component", component, "error", err, "root_cause", rootCauseMessage(err))
		dix.recordRecentErrorWithContext("inject", component, err, recentErrorContext{
			Stage:     "inject",
			RootCause: rootCauseMessage(err),
		})
		panic(err)
	}
	return param
}

// TryInject injects dependencies into the given parameter. Returns error instead of panicking.
// NOTE: Dix container is not thread-safe. Do not call Provide/Inject concurrently on the same container.
func (dix *Dix) TryInject(param any, opts ...Option) error {
	component := describeComponent(param)
	if dep, ok := dix.isCycle(); ok {
		err := errors.New("circular dependency: " + dep)
		logger.Warn("dependency cycle detected", "cycle_path", dep, "component", component)
		dix.recordRecentErrorWithContext("try_inject", component, err, recentErrorContext{
			Stage:     "cycle_check",
			RootCause: rootCauseMessage(err),
		})
		return err
	}

	err := dix.inject(param, opts...)
	if err != nil {
		logger.Warn("try inject failed", "component", component, "error", err, "root_cause", rootCauseMessage(err))
		dix.recordRecentErrorWithContext("try_inject", component, err, recentErrorContext{
			Stage:     "inject",
			RootCause: rootCauseMessage(err),
		})
	}
	return err
}

// GetProvideAllInputTypes returns all input types for a given type, including struct fields
// This is a public version of getProvideAllInputs that returns types instead of internal structures
func GetProvideAllInputTypes(typ reflect.Type) []reflect.Type {
	inputs := getProvideAllInputs(typ)
	types := make([]reflect.Type, 0, len(inputs))
	for _, input := range inputs {
		types = append(types, input.typ)
	}
	return types
}

// GetProviders returns a copy of the providers map for inspection
// This is useful for HTTP visualization endpoints
func (dix *Dix) GetProviders() map[reflect.Type][]*providerFn {
	result := make(map[reflect.Type][]*providerFn)
	for k, v := range dix.providers {
		result[k] = v
	}
	return result
}

// GetObjects returns a copy of the objects map for inspection
// This is useful for HTTP visualization endpoints
func (dix *Dix) GetObjects() map[reflect.Type]map[string][]reflect.Value {
	result := make(map[reflect.Type]map[string][]reflect.Value)
	for k, v := range dix.objects {
		groupMap := make(map[string][]reflect.Value)
		for gk, gv := range v {
			groupMap[gk] = gv
		}
		result[k] = groupMap
	}
	return result
}

// ProviderDetails contains detailed information about a provider
type ProviderDetails struct {
	OutputType   string
	OutputPkg    string
	FunctionName string
	FunctionPkg  string
	FunctionFile string
	FunctionLine int
	InputTypes   []string
	InputPkgs    []string
}

// ProviderRuntimeStats contains provider runtime metrics for diagnostics.
type ProviderRuntimeStats struct {
	FunctionName      string        `json:"function_name"`
	OutputType        string        `json:"output_type"`
	CallCount         int           `json:"call_count"`
	TotalDuration     time.Duration `json:"total_duration"`
	AverageDuration   time.Duration `json:"average_duration"`
	LastDuration      time.Duration `json:"last_duration"`
	LastError         string        `json:"last_error,omitempty"`
	LastRunAtUnixNano int64         `json:"last_run_at_unix_nano"`
}

// RecentError contains a recently captured Inject/TryInject failure event.
type RecentError struct {
	Operation          string        `json:"operation"`
	Component          string        `json:"component"`
	Stage              string        `json:"stage,omitempty"`
	ProviderFunction   string        `json:"provider_function,omitempty"`
	OutputType         string        `json:"output_type,omitempty"`
	InputType          string        `json:"input_type,omitempty"`
	InputTypes         []string      `json:"input_types,omitempty"`
	Message            string        `json:"message"`
	RootCause          string        `json:"root_cause,omitempty"`
	TimedOut           bool          `json:"timed_out,omitempty"`
	Duration           time.Duration `json:"duration,omitempty"`
	Timeout            time.Duration `json:"timeout,omitempty"`
	OccurredAtUnixNano int64         `json:"occurred_at_unix_nano"`
}

// GetProviderDetails returns detailed information about all providers
func (dix *Dix) GetProviderDetails() []ProviderDetails {
	var details []ProviderDetails
	for outputType, providerList := range dix.providers {
		for _, providerFn := range providerList {
			fnName := GetFnName(providerFn.fn)
			fnFile, fnLine := resolveFuncLocation(providerFn.fn)
			var inputTypes []string
			var inputPkgs []string
			seen := make(map[string]bool)
			for _, input := range providerFn.inputList {
				if input.isStruct || input.typ.Kind() == reflect.Struct {
					for _, in := range getProvideAllInputs(input.typ) {
						name := in.typ.String()
						if name == "" || seen[name] {
							continue
						}
						seen[name] = true
						inputTypes = append(inputTypes, name)
						inputPkgs = append(inputPkgs, resolveTypePkgPath(in.typ))
					}
					continue
				}

				name := input.typ.String()
				if name == "" || seen[name] {
					continue
				}
				seen[name] = true
				inputTypes = append(inputTypes, name)
				inputPkgs = append(inputPkgs, resolveTypePkgPath(input.typ))
			}
			details = append(details, ProviderDetails{
				OutputType:   outputType.String(),
				OutputPkg:    resolveTypePkgPath(outputType),
				FunctionName: fnName,
				FunctionPkg:  resolveFuncPkgPath(fnName),
				FunctionFile: fnFile,
				FunctionLine: fnLine,
				InputTypes:   inputTypes,
				InputPkgs:    inputPkgs,
			})
		}
	}
	return details
}

// GetProviderRuntimeStats returns runtime stats sorted by total duration (descending).
// This is helpful for startup latency diagnosis.
func (dix *Dix) GetProviderRuntimeStats() []ProviderRuntimeStats {
	stats := make([]ProviderRuntimeStats, 0, len(dix.providers))
	seen := make(map[reflect.Value]bool)

	for _, providerList := range dix.providers {
		for _, p := range providerList {
			if p == nil || seen[p.fn] {
				continue
			}
			seen[p.fn] = true

			outputType := ""
			if p.output != nil && p.output.typ != nil {
				outputType = p.output.typ.String()
			}

			item := ProviderRuntimeStats{
				FunctionName: GetFnName(p.fn),
				OutputType:   outputType,
			}

			if s, ok := dix.providerStats[p.fn]; ok && s != nil {
				avg := time.Duration(0)
				if s.CallCount > 0 {
					avg = s.TotalDuration / time.Duration(s.CallCount)
				}
				item.FunctionName = s.FunctionName
				if s.OutputType != "" {
					item.OutputType = s.OutputType
				}
				item.CallCount = s.CallCount
				item.TotalDuration = s.TotalDuration
				item.AverageDuration = avg
				item.LastDuration = s.LastDuration
				item.LastError = s.LastError
				item.LastRunAtUnixNano = s.LastRunAt.UnixNano()
			}

			stats = append(stats, item)
		}
	}

	sort.Slice(stats, func(i, j int) bool {
		if stats[i].TotalDuration == stats[j].TotalDuration {
			return stats[i].FunctionName < stats[j].FunctionName
		}
		return stats[i].TotalDuration > stats[j].TotalDuration
	})

	return stats
}

// GetRecentErrors returns recent Inject/TryInject errors in reverse-chronological order.
// limit <= 0 means return all currently retained errors.
func (dix *Dix) GetRecentErrors(limit int) []RecentError {
	total := len(dix.recentErrors)
	if total == 0 {
		return []RecentError{}
	}

	if limit <= 0 || limit > total {
		limit = total
	}

	result := make([]RecentError, 0, limit)
	for i := total - 1; i >= 0 && len(result) < limit; i-- {
		r := dix.recentErrors[i]
		result = append(result, RecentError{
			Operation:          r.Operation,
			Component:          r.Component,
			Stage:              r.Stage,
			ProviderFunction:   r.ProviderFunction,
			OutputType:         r.OutputType,
			InputType:          r.InputType,
			InputTypes:         append([]string{}, r.InputTypes...),
			Message:            r.Message,
			RootCause:          r.RootCause,
			TimedOut:           r.TimedOut,
			Duration:           r.Duration,
			Timeout:            r.Timeout,
			OccurredAtUnixNano: r.Occurred.UnixNano(),
		})
	}

	return result
}

func normalizeRecoveredError(r any) error {
	if r == nil {
		return errors.New("unknown panic")
	}
	if e, ok := r.(error); ok {
		return e
	}
	if s, ok := r.(string); ok {
		return errors.New(s)
	}
	return fmt.Errorf("panic: %v", r)
}

func resolveTypePkgPath(typ reflect.Type) string {
	if typ == nil {
		return ""
	}
	for typ.Kind() == reflect.Ptr || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array {
		typ = typ.Elem()
		if typ == nil {
			return ""
		}
	}
	if typ.Kind() == reflect.Map {
		return resolveTypePkgPath(typ.Elem())
	}
	return typ.PkgPath()
}

func resolveFuncPkgPath(fnName string) string {
	name := strings.TrimSpace(fnName)
	if name == "" {
		return ""
	}
	if idx := strings.LastIndex(name, "."); idx > 0 {
		return name[:idx]
	}
	return ""
}

func resolveFuncLocation(fn reflect.Value) (string, int) {
	if !fn.IsValid() || fn.IsZero() {
		return "", 0
	}

	pc := fn.Pointer()
	f := runtime.FuncForPC(pc)
	if f == nil {
		return "", 0
	}

	file, line := f.FileLine(pc)
	return file, line
}
