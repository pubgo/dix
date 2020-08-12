package dix

var defaultDix = New()

func Dix(data interface{}) error { return defaultDix.Dix(data) }
func Graph() string              { return defaultDix.graph() }
