package di

import (
	"github.com/pubgo/dix/dix_internal"
)

var _dix = dix_internal.New(dix_internal.WithValuesNull())

// Provide 注册对象构造器
func Provide(data any) {
	_dix.Provide(data)
}

// Inject 注入对象
//
//	data: <*struct>或<func>
func Inject[T any](data T, opts ...dix_internal.Option) T {
	_ = _dix.Inject(data, opts...)
	return data
}

// Graph Dix graph
func Graph() *dix_internal.Graph {
	return _dix.Graph()
}
