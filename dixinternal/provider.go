package dixinternal

import (
	"reflect"
	"time"

	"github.com/pubgo/funk/stack"
)

// FuncProvider 函数提供者实现
type FuncProvider struct {
	fn           reflect.Value
	outputType   reflect.Type
	dependencies []Dependency
	initialized  bool
	isMap        bool
	isList       bool
}

// NewFuncProvider 创建函数提供者
func NewFuncProvider(fn reflect.Value) (*FuncProvider, error) {
	if fn.Kind() != reflect.Func {
		return nil, NewValidationError("provider must be a function").
			WithDetail("actual_kind", fn.Kind().String())
	}

	fnType := fn.Type()
	if fnType.NumOut() == 0 {
		return nil, NewValidationError("provider function must have at least one return value")
	}

	if fnType.IsVariadic() {
		return nil, NewValidationError("provider function cannot have variadic parameters")
	}

	// 解析输出类型
	outputType := fnType.Out(0)
	isMap := false
	isList := false

	switch outputType.Kind() {
	case reflect.Map:
		isMap = true
		outputType = outputType.Elem()
		if outputType.Kind() == reflect.Slice {
			isList = true
			outputType = outputType.Elem()
		}
	case reflect.Slice:
		isList = true
		outputType = outputType.Elem()
	case reflect.Ptr, reflect.Interface, reflect.Func: // 支持的单类型
	case reflect.Struct: // 结构体类型，需要特殊处理
	default:
		return nil, NewValidationError("unsupported provider output type").
			WithDetail("type", outputType.String()).
			WithDetail("kind", outputType.Kind().String())
	}

	// 解析依赖
	dependencies, err := parseDependencies(fnType)
	if err != nil {
		return nil, WrapError(err, ErrorTypeProvider, "failed to parse dependencies")
	}

	return &FuncProvider{
		fn:           fn,
		outputType:   outputType,
		dependencies: dependencies,
		initialized:  false,
		isMap:        isMap,
		isList:       isList,
	}, nil
}

func (p *FuncProvider) Type() reflect.Type {
	return p.outputType
}

func (p *FuncProvider) Invoke(args []reflect.Value) ([]reflect.Value, error) {
	if len(args) != len(p.dependencies) {
		return nil, NewInvocationError("argument count mismatch").
			WithDetail("expected", len(p.dependencies)).
			WithDetail("actual", len(args))
	}

	defer func() {
		if r := recover(); r != nil {
			// 记录调用栈信息
			fnStack := stack.CallerWithFunc(p.fn)
			logger.Error().
				Str("provider", fnStack.String()).
				Interface("panic", r).
				Msg("provider function panicked")
		}
	}()

	start := time.Now()
	results := p.fn.Call(args)

	// 记录调用信息
	fnStack := stack.CallerWithFunc(p.fn)
	logger.Debug().
		Str("cost", time.Since(start).String()).
		Str("provider", fnStack.String()).
		Msgf("invoked provider %s", fnStack.Name)

	return results, nil
}

func (p *FuncProvider) Dependencies() []Dependency {
	return p.dependencies
}

func (p *FuncProvider) IsInitialized() bool {
	return p.initialized
}

func (p *FuncProvider) SetInitialized(initialized bool) {
	p.initialized = initialized
}

func (p *FuncProvider) IsMap() bool {
	return p.isMap
}

func (p *FuncProvider) IsList() bool {
	return p.isList
}

// DependencyImpl 依赖实现
type DependencyImpl struct {
	typ    reflect.Type
	isMap  bool
	isList bool
}

func NewDependency(typ reflect.Type, isMap, isList bool) *DependencyImpl {
	return &DependencyImpl{
		typ:    typ,
		isMap:  isMap,
		isList: isList,
	}
}

func (d *DependencyImpl) Type() reflect.Type {
	return d.typ
}

func (d *DependencyImpl) IsMap() bool {
	return d.isMap
}

func (d *DependencyImpl) IsList() bool {
	return d.isList
}

func (d *DependencyImpl) Validate() error {
	if d.isMap && !isSupportedType(d.typ.Kind()) {
		return NewValidationError("unsupported map value type").
			WithDetail("type", d.typ.String()).
			WithDetail("kind", d.typ.Kind().String())
	}

	if d.isList && !isSupportedType(d.typ.Kind()) {
		return NewValidationError("unsupported list element type").
			WithDetail("type", d.typ.String()).
			WithDetail("kind", d.typ.Kind().String())
	}

	if !isSupportedType(d.typ.Kind()) {
		return NewValidationError("unsupported dependency type").
			WithDetail("type", d.typ.String()).
			WithDetail("kind", d.typ.Kind().String())
	}

	return nil
}

// parseDependencies 解析函数的依赖
func parseDependencies(fnType reflect.Type) ([]Dependency, error) {
	var dependencies []Dependency

	for i := 0; i < fnType.NumIn(); i++ {
		paramType := fnType.In(i)

		switch paramType.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Func, reflect.Struct:
			dep := NewDependency(paramType, false, false)
			if err := dep.Validate(); err != nil {
				return nil, err
			}
			dependencies = append(dependencies, dep)

		case reflect.Map:
			elemType := paramType.Elem()
			isList := elemType.Kind() == reflect.Slice
			if isList {
				elemType = elemType.Elem()
			}
			dep := NewDependency(elemType, true, isList)
			if err := dep.Validate(); err != nil {
				return nil, err
			}
			dependencies = append(dependencies, dep)

		case reflect.Slice:
			elemType := paramType.Elem()
			dep := NewDependency(elemType, false, true)
			if err := dep.Validate(); err != nil {
				return nil, err
			}
			dependencies = append(dependencies, dep)

		default:
			return nil, NewValidationError("unsupported parameter type").
				WithDetail("type", paramType.String()).
				WithDetail("kind", paramType.Kind().String()).
				WithDetail("parameter_index", i)
		}
	}

	return dependencies, nil
}

// isSupportedType 检查是否为支持的类型
func isSupportedType(kind reflect.Kind) bool {
	switch kind {
	case reflect.Interface, reflect.Ptr, reflect.Func, reflect.Struct:
		return true
	default:
		return false
	}
}
