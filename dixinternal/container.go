package dixinternal

import (
	"fmt"
	"reflect"
)

// ContainerImpl 容器实现
type ContainerImpl struct {
	options       Options
	resolver      *ResolverImpl
	injector      *InjectorImpl
	cycleDetector *CycleDetectorImpl
}

// NewContainer 创建新的容器实例
func NewContainer(opts ...Option) Container {
	// 应用配置选项
	options := Options{AllowValuesNull: true}
	for _, opt := range opts {
		opt(&options)
	}
	options.Check()

	// 创建组件
	resolver := NewResolver()
	injector := NewInjector(resolver)
	cycleDetector := NewCycleDetector()

	container := &ContainerImpl{
		options:       options,
		resolver:      resolver,
		injector:      injector,
		cycleDetector: cycleDetector,
	}

	// 注册容器自身
	containerProvider, _ := NewFuncProvider(reflect.ValueOf(func() Container { return container }))
	resolver.AddProvider(containerProvider)

	return container
}

// Provide 注册依赖提供者
func (c *ContainerImpl) Provide(provider interface{}) error {
	if provider == nil {
		return NewValidationError("provider cannot be nil")
	}

	providerValue := reflect.ValueOf(provider)
	if !providerValue.IsValid() || providerValue.IsZero() {
		return NewValidationError("provider must be valid and non-nil")
	}

	// 创建函数提供者
	funcProvider, err := NewFuncProvider(providerValue)
	if err != nil {
		return WrapError(err, ErrorTypeProvider, "failed to create provider")
	}

	// 添加到解析器
	c.resolver.AddProvider(funcProvider)

	// 检查循环依赖
	providers := c.getAllProviders()
	if err := c.cycleDetector.ValidateNoCycles(providers); err != nil {
		// 如果发现循环依赖，移除刚添加的提供者
		c.removeProvider(funcProvider)
		return WrapError(err, ErrorTypeCyclicDep, "provider registration failed due to cycle detection")
	}

	return nil
}

// Inject 执行依赖注入
func (c *ContainerImpl) Inject(target interface{}, opts ...Option) (err error) {
	defer func() {
		if r := recover(); r != nil {
			var panicErr error
			if e, ok := r.(error); ok {
				panicErr = e
			} else {
				panicErr = fmt.Errorf("panic: %v", r)
			}
			err = WrapError(panicErr, ErrorTypeInjection, "dependency injection failed with panic").
				WithDetail("panic_value", r)
		}
	}()

	// 合并选项
	mergedOpts := c.options
	for _, opt := range opts {
		opt(&mergedOpts)
	}

	// 检查循环依赖
	providers := c.getAllProviders()
	if err := c.cycleDetector.ValidateNoCycles(providers); err != nil {
		return WrapError(err, ErrorTypeCyclicDep, "injection failed due to cycle detection")
	}

	// 执行注入
	return c.injector.InjectTarget(target, mergedOpts)
}

// Graph 获取依赖关系图
func (c *ContainerImpl) Graph() *Graph {
	renderer := NewDotRenderer()

	return &Graph{
		Providers: renderer.RenderProviders(c.resolver.providers),
		Objects:   renderer.RenderObjects(c.resolver.objects),
	}
}

// Option 获取容器配置
func (c *ContainerImpl) Option() Options {
	return c.options
}

// getAllProviders 获取所有提供者
func (c *ContainerImpl) getAllProviders() map[reflect.Type][]Provider {
	result := make(map[reflect.Type][]Provider)

	for typ, providers := range c.resolver.providers {
		result[typ] = make([]Provider, len(providers))
		copy(result[typ], providers)
	}

	return result
}

// removeProvider 移除提供者（用于回滚）
func (c *ContainerImpl) removeProvider(provider Provider) {
	// 移除该 provider 在所有其能提供的类型下的注册
	providedTypes := provider.ProvidedTypes()

	for _, typ := range providedTypes {
		providers := c.resolver.providers[typ]

		for i, p := range providers {
			if p == provider {
				// 移除该提供者
				c.resolver.providers[typ] = append(providers[:i], providers[i+1:]...)
				break
			}
		}

		// 如果该类型没有提供者了，删除整个条目
		if len(c.resolver.providers[typ]) == 0 {
			delete(c.resolver.providers, typ)
		}
	}

	// 向后兼容：如果没有 ProvidedTypes，使用传统的 PrimaryType() 方法
	if len(providedTypes) == 0 {
		typ := provider.PrimaryType()
		providers := c.resolver.providers[typ]

		for i, p := range providers {
			if p == provider {
				// 移除该提供者
				c.resolver.providers[typ] = append(providers[:i], providers[i+1:]...)
				break
			}
		}

		// 如果该类型没有提供者了，删除整个条目
		if len(c.resolver.providers[typ]) == 0 {
			delete(c.resolver.providers, typ)
		}
	}
}
