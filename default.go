package dix

var _dix = New()

// Provider ...
func Provider(data ...interface{}) error { return _dix.Provider(data...) }

// ProviderNs ...
func ProviderNs(name string, data interface{}) error { return _dix.ProviderNs(name, data) }

// Inject 注入对象
// ns: namespace
func Inject(data interface{}, ns ...string) error { return _dix.Inject(data, ns...) }

// Graph dix graph
func Graph() string { return _dix.graph() }

// Json dix json graph
func Json() map[string]interface{} { return _dix.json() }

// New dix new
func New(opts ...Option) *dix { return newDix(opts...) }
