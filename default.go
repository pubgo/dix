package dix

var _dix = New()

// Register 注册provider和invoke
func Register(data interface{}) { _dix.register(data) }

// Inject 注入对象
func Inject(data interface{}) { _dix.inject(data) }

// Invoke 懒执行注册的invoke
func Invoke() { _dix.invoke() }

// Graph dix graph
//func Graph() string { return _dix.Graph() }

// Json dix json graph
//func Json() map[string]interface{} { return _dix.json() }

// New dix new
func New(opts ...Option) *dix { return newDix(opts...) }
