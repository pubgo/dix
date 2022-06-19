package dix

var _dix = New()

// Provider 注册provider
// 	provider必须有返回值, 且返回值只能有一个, 类型为map,any,ptr,slice,func
func Provider(data any) { _dix.Provider(data) }

// Inject 注入对象
// 	data是<*struct>或者<func>
func Inject[T any](data T, opts ...Option) T {
	_ = _dix.Inject(data, opts...)
	return data
}

// Graph dix graph
func Graph() *graph { return _dix.Graph() }

// New dix new
func New(opts ...Option) *dix { return newDix(opts...) }
