package di

import (
	"github.com/alecthomas/repr"
	"github.com/pubgo/dix"
	"github.com/pubgo/funk/assert"
)

var _dix = dix.New(dix.WithValuesNull())

// Provide 注册对象构造器
func Provide(data any) {
	assert.Must(_dix.Provide(data), repr.String(data))
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
