package di

import "github.com/pubgo/dix"

var _dix = dix.New()

// Provide 注册对象构造器
func Provide(data any) { _dix.Provide(data) }

// Inject 注入对象
// 	data是<*struct>或者<func>
func Inject[T any](data T, opts ...dix.Option) T {
	_ = _dix.Inject(data, opts...)
	return data
}

// Graph Dix graph
func Graph() *dix.Graph { return _dix.Graph() }
