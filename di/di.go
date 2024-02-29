package di

import (
	"github.com/pubgo/dix"
)

var _dix = dix.New(dix.WithValuesNull())

// Provide 注册对象构造器
func Provide(data any) {
	_dix.Provide(data)
}

// SetValue 设置对象
func SetValue(data any, types ...any) {
	_dix.SetValue(data, types...)
}

// Inject 注入对象
//
//	data: <*struct>或<func>
func Inject[T any](data T, opts ...dix.Option) T {
	_ = _dix.Inject(data, opts...)
	return data
}

// Graph Dix graph
func Graph() *dix.Graph {
	return _dix.Graph()
}
