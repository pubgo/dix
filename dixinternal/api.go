package dixinternal

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
