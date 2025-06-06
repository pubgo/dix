package dixinternal

import (
	"reflect"
)

// Container 依赖注入容器接口
type Container interface {
	// Provide 注册依赖提供者
	Provide(provider interface{}) error

	// Inject 执行依赖注入
	Inject(target interface{}, opts ...Option) error

	// Get 获取指定类型的实例
	Get(typ reflect.Type, opts ...Option) (interface{}, error)

	// Graph 获取依赖关系图
	Graph() *Graph

	// Option 获取容器配置
	Option() Options
}

// Provider 依赖提供者接口
type Provider interface {
	// Type 返回提供的类型
	Type() reflect.Type

	// Invoke 调用提供者函数
	Invoke(args []reflect.Value) ([]reflect.Value, error)

	// Dependencies 返回依赖的类型列表
	Dependencies() []Dependency

	// IsInitialized 是否已初始化
	IsInitialized() bool

	// SetInitialized 设置初始化状态
	SetInitialized(bool)
}

// Dependency 依赖描述接口
type Dependency interface {
	// Type 依赖类型
	Type() reflect.Type

	// IsMap 是否为Map类型
	IsMap() bool

	// IsList 是否为List类型
	IsList() bool

	// Validate 验证依赖是否有效
	Validate() error
}

// Resolver 依赖解析器接口
type Resolver interface {
	// Resolve 解析依赖
	Resolve(typ reflect.Type, opts Options) (reflect.Value, error)

	// ResolveAll 解析所有依赖
	ResolveAll(deps []Dependency, opts Options) ([]reflect.Value, error)
}

// Injector 注入器接口
type Injector interface {
	// InjectStruct 注入结构体
	InjectStruct(target reflect.Value, opts Options) error

	// InjectFunc 注入函数
	InjectFunc(fn reflect.Value, opts Options) error
}

// CycleDetector 循环依赖检测器接口
type CycleDetector interface {
	// DetectCycle 检测循环依赖
	DetectCycle(providers map[reflect.Type][]Provider) ([]reflect.Type, error)
}

// GraphRenderer 图形渲染器接口
type GraphRenderer interface {
	// RenderProviders 渲染提供者图
	RenderProviders(providers map[reflect.Type][]Provider) string

	// RenderObjects 渲染对象图
	RenderObjects(objects map[reflect.Type]map[string][]reflect.Value) string
}

// Graph 依赖关系图
type Graph struct {
	Objects   string `json:"objects"`
	Providers string `json:"providers"`
}
