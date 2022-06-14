package dix

var _dix = New()

// Register 注册provider和invoke
// 	provider必须有返回值, 且返回值只能有一个, 类型为map,interface,ptr,slice,func
func Register(data interface{}) { _dix.Register(data) }

// Provider 同Register
func Provider(data interface{}) { _dix.Provider(data) }

// Inject 注入对象
// 	data是<*struct>或者<func>
func Inject(data interface{}, opts ...Option) interface{} { return _dix.Inject(data, opts...) }

// Graph dix graph
func Graph() *graph { return _dix.Graph() }

// New dix new
func New(opts ...Option) *dix { return newDix(opts...) }
