package dix

var _dix = New()

// Register 注册provider和invoke
// 	invoke没有返回值
// 	provider必须有返回值, 且返回值只能有一个, 类型为map,interface,ptr
func Register(data interface{}) { _dix.Register(data) }
func Provider(data interface{}) { _dix.Register(data) }

// Inject 注入对象
// 	data是指针类型
func Inject(data interface{}) interface{} { return _dix.Inject(data) }

// Graph dix graph
func Graph() *graph { return _dix.Graph() }

// New dix new
func New(opts ...Option) *dix { return newDix(opts...) }
