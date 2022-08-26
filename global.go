package dix

import (
	"github.com/pubgo/funk/assert"
	"reflect"
)

var _dix = New()

func Sub(fn func(ns string, val reflect.Value)) {
	assert.Must(_dix.sub(fn))
}

// Provider 注册provider
// 	provider必须有返回值, 且返回值只能有一个, 类型为map,any,ptr,slice,func
func Provider(data any) { _dix.Provider(data) }

// Provide 注册provider
// 	同Provider
func Provide(data any) { _dix.Provider(data) }

// Inject 注入对象
// 	data是<*struct>或者<func>
func Inject[T any](data T, opts ...Option) T {
	_ = _dix.Inject(data, opts...)
	return data
}

// SubDix 子域
func SubDix(opts ...Option) *Dix {
	return _dix.dix(opts...)
}

// Graph Dix graph
func Graph() *graph { return _dix.Graph() }

// New Dix new
func New(opts ...Option) *Dix { return newDix(opts...) }
