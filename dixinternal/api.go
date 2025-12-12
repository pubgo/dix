package dixinternal

import (
	"errors"
	"reflect"
)

// New Dix new
func New(opts ...Option) *Dix {
	return newDix(opts...)
}

func (dix *Dix) Provide(param any) {
	dix.provide(param)
}

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
	FunctionName string
	InputTypes   []string
}

// GetProviderDetails returns detailed information about all providers
func (dix *Dix) GetProviderDetails() []ProviderDetails {
	var details []ProviderDetails
	for outputType, providerList := range dix.providers {
		for _, providerFn := range providerList {
			fnName := GetFnName(providerFn.fn)
			var inputTypes []string
			for _, input := range providerFn.inputList {
				inputTypes = append(inputTypes, input.typ.String())
			}
			details = append(details, ProviderDetails{
				OutputType:   outputType.String(),
				FunctionName: fnName,
				InputTypes:   inputTypes,
			})
		}
	}
	return details
}
