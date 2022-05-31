package dix

var _dix = New()

// Register 注册provider和invoke
// 	invoke没有返回值
// 	provider必须有返回值, 且返回值只能有一个, 类型为map,interface,ptr
func Register(data interface{}) { _dix.register(data) }

// Inject 注入对象
// 	data是指针类型
func Inject(data interface{}) { _dix.inject(data) }

// Invoke 懒执行注册的invoke
// 	执行所有预先注册的invoke
func Invoke() { _dix.invoke() }

// Graph dix graph
func Graph() string { return _dix.graph() }

// New dix new
func New(opts ...Option) *dix { return newDix(opts...) }
