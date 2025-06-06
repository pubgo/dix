package dixinternal

import (
	"reflect"
)

// New 创建新的依赖注入容器
func New(opts ...Option) Container {
	return NewContainer(opts...)
}

// Provide 注册依赖提供者（便捷函数）
func Provide(container Container, provider interface{}) error {
	return container.Provide(provider)
}

// Inject 执行依赖注入（便捷函数）
func Inject(container Container, target interface{}, opts ...Option) error {
	return container.Inject(target, opts...)
}

// Get 获取指定类型的实例（便捷函数）
func Get[T any](container Container, opts ...Option) (T, error) {
	var zero T
	typ := reflect.TypeOf((*T)(nil)).Elem()

	result, err := container.Get(typ, opts...)
	if err != nil {
		return zero, err
	}

	if result == nil {
		return zero, NewNotFoundError(typ)
	}

	if typed, ok := result.(T); ok {
		return typed, nil
	}

	return zero, NewValidationError("type assertion failed").
		WithDetail("expected_type", typ.String()).
		WithDetail("actual_type", reflect.TypeOf(result).String())
}

// MustGet 获取指定类型的实例，如果失败则panic（便捷函数）
func MustGet[T any](container Container, opts ...Option) T {
	result, err := Get[T](container, opts...)
	if err != nil {
		panic(err)
	}
	return result
}
