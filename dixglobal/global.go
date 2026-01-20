package dixglobal

import (
	"reflect"

	"github.com/pubgo/dix/v2"
)

var _dix = dix.New(dix.WithValuesNull())

// Example:
//
//	dixglobal.Provide(func() *Config { return &Config{Endpoint: "localhost:..."} }) // Configuration
//	dixglobal.Provide(NewDB)                                                  // Database connection
//	dixglobal.Provide(NewHTTPServer)                                          // Server
//
//	dixglobal.Invoke(func(server *http.Server) { // Application startup
//		server.ListenAndServe()
//	})
//
// For more usage details, see the documentation for the Container type.

// Provide registers an object constructor
func Provide(data any) {
	_dix.Provide(data)
}

// Inject injects objects
//
//	data: <*struct> or <func>
func Inject[T any](data T, opts ...dix.Option) T {
	vp := reflect.ValueOf(data)
	if vp.Kind() == reflect.Struct {
		_ = _dix.Inject(&data, opts...)
	} else {
		_ = _dix.Inject(data, opts...)
	}
	return data
}

func InjectT[T any](opts ...dix.Option) T {
	var data T
	if reflect.TypeOf(data).Kind() != reflect.Struct {
		panic("<T> type kind is not struct")
	}

	_ = _dix.Inject(&data, opts...)
	return data
}
