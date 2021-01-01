package dix

var defaultDix = New()

// Init ...
func Init(opts ...Option) error { return defaultDix.Init(opts...) }

// Dix ...
func Dix(data ...interface{}) error { return defaultDix.Dix(data...) }

// Graph dix graph
func Graph() string { return defaultDix.graph() }

// Json dix json graph
func Json() map[string]interface{} { return defaultDix.json() }
