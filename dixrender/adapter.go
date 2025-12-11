package dixrender

import (
	"reflect"

	"github.com/pubgo/dix/v2/dixinternal"
)

// dixAdapter adapts dixinternal.Dix to dixrender.DixAccessor interface
type dixAdapter struct {
	dix *dixinternal.Dix
}

// NewDixAdapter creates a new adapter for dixinternal.Dix
func NewDixAdapter(dix *dixinternal.Dix) DixAccessor {
	return &dixAdapter{dix: dix}
}

// providerFnWrapper wraps dixinternal's providerFn to implement dixrender.ProviderFnAccessor
type providerFnWrapper struct {
	providerFn interface{} // *dixinternal.providerFn (unexported type)
}

func (w *providerFnWrapper) GetFn() reflect.Value {
	// Use reflection to access the 'fn' field of providerFn
	val := reflect.ValueOf(w.providerFn).Elem()
	fnField := val.FieldByName("fn")
	if fnField.IsValid() {
		return fnField
	}
	return reflect.Value{}
}

func (w *providerFnWrapper) GetInputTypes() []reflect.Type {
	// Use the public API through helper functions
	// Since GetProviderInputTypes requires *providerFn which is unexported,
	// we'll use reflection to access the inputList field
	val := reflect.ValueOf(w.providerFn).Elem()
	inputListField := val.FieldByName("inputList")
	if inputListField.IsValid() && inputListField.Kind() == reflect.Slice {
		var result []reflect.Type
		for i := 0; i < inputListField.Len(); i++ {
			inputItem := inputListField.Index(i).Elem()
			typField := inputItem.FieldByName("typ")
			if typField.IsValid() {
				result = append(result, typField.Interface().(reflect.Type))
			}
		}
		return result
	}
	return nil
}

func (a *dixAdapter) GetProviders() map[reflect.Type][]ProviderFnAccessor {
	providers := a.dix.GetProviders()
	result := make(map[reflect.Type][]ProviderFnAccessor)
	for k, v := range providers {
		adapters := make([]ProviderFnAccessor, len(v))
		for i, p := range v {
			adapters[i] = &providerFnWrapper{
				providerFn: p,
			}
		}
		result[k] = adapters
	}
	return result
}

func (a *dixAdapter) GetObjects() map[reflect.Type]map[string][]reflect.Value {
	return a.dix.GetObjects()
}

func (a *dixAdapter) GetProviderInputTypes(p ProviderFnAccessor) []reflect.Type {
	if wrapper, ok := p.(*providerFnWrapper); ok {
		return wrapper.GetInputTypes()
	}
	return nil
}

func (a *dixAdapter) GetProvideAllInputTypes(typ reflect.Type) []reflect.Type {
	return dixinternal.GetProvideAllInputTypes(typ)
}

func (a *dixAdapter) GetFnName(fn reflect.Value) string {
	return dixinternal.GetFnName(fn)
}

func (a *dixAdapter) GetProviderFn(p ProviderFnAccessor) reflect.Value {
	if wrapper, ok := p.(*providerFnWrapper); ok {
		return wrapper.GetFn()
	}
	return reflect.Value{}
}
