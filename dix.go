package dix

import (
	"reflect"

	"github.com/pubgo/dix/dixinternal"
	"github.com/pubgo/funk/assert"
)

const (
	InjectMethodPrefix = dixinternal.InjectMethodPrefix
)

type (
	Option    = dixinternal.Option
	Options   = dixinternal.Options
	Container = dixinternal.Container
	Graph     = dixinternal.Graph
)

// WithValuesNull 配置选项：允许值为null
func WithValuesNull() Option {
	return dixinternal.WithValuesNull()
}

// New 创建新的依赖注入容器
func New(opts ...Option) Container {
	return dixinternal.New(opts...)
}

// Inject 执行依赖注入
//
//	container: 依赖注入容器
//	target: 注入目标 <*struct> 或 <func>
//	opts: 可选配置
func Inject[T any](container Container, target T, opts ...Option) T {
	vp := reflect.ValueOf(target)
	if vp.Kind() == reflect.Struct {
		assert.Must(container.Inject(&target, opts...))
	} else {
		assert.Must(container.Inject(target, opts...))
	}
	return target
}

// Provide 注册依赖提供者
//
//	container: 依赖注入容器
//	provider: 提供者函数
func Provide(container Container, provider any) {
	assert.Must(container.Provide(provider))
}

// Get 获取指定类型的实例（泛型版本）
//
//	container: 依赖注入容器
//	opts: 可选配置
func Get[T any](container Container, opts ...Option) (T, error) {
	return dixinternal.Get[T](container, opts...)
}

// MustGet 获取指定类型的实例，失败时panic（泛型版本）
//
//	container: 依赖注入容器
//	opts: 可选配置
func MustGet[T any](container Container, opts ...Option) T {
	return dixinternal.MustGet[T](container, opts...)
}

// GetGraph 获取依赖关系图
//
//	container: 依赖注入容器
func GetGraph(container Container) *Graph {
	return container.Graph()
}

// 为了向后兼容，保留旧的类型别名
// Deprecated: 使用 Container 替代
type Dix = Container
