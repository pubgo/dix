package dixinternal

import (
	"errors"
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
	dix.provide(param)
}

// TryProvide registers a provider function. Returns error instead of panicking.
// NOTE: Dix container is not thread-safe. Do not call Provide/Inject concurrently on the same container.
func (dix *Dix) TryProvide(param any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = errors.New(r.(string))
			}
		}
	}()
	dix.provide(param)
	return nil
}

// Inject injects dependencies into the given parameter. Panics on error.
// NOTE: Dix container is not thread-safe. Do not call Provide/Inject concurrently on the same container.
func (dix *Dix) Inject(param any, opts ...Option) any {
	if dep, ok := dix.isCycle(); ok {
		logger.Error("dependency cycle detected", "cycle_path", dep, "component", reflect.TypeOf(param).String())
		panic(errors.New("circular dependency: " + dep))
	}

	if err := dix.inject(param, opts...); err != nil {
		panic(err)
	}
	return param
}

// TryInject injects dependencies into the given parameter. Returns error instead of panicking.
// NOTE: Dix container is not thread-safe. Do not call Provide/Inject concurrently on the same container.
func (dix *Dix) TryInject(param any, opts ...Option) error {
	if dep, ok := dix.isCycle(); ok {
		logger.Warn("dependency cycle detected", "cycle_path", dep, "component", reflect.TypeOf(param).String())
		return errors.New("circular dependency: " + dep)
	}

	return dix.inject(param, opts...)
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
