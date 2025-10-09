package dix

import (
	_ "embed"
	"reflect"

	"github.com/pubgo/dix/v2/dixinternal"
)

//go:embed .version
var version string

func ReleaseVersion() string { return version }

type (
	Option  = dixinternal.Option
	Options = dixinternal.Options
	Dix     = dixinternal.Dix
	Graph   = dixinternal.Graph
)

var WithValuesNull = dixinternal.WithValuesNull
var New = dixinternal.New

func Inject[T any](di *Dix, data T, opts ...Option) T {
	vp := reflect.ValueOf(data)
	if vp.Kind() == reflect.Struct {
		_ = di.Inject(&data, opts...)
	} else {
		_ = di.Inject(data, opts...)
	}

	return data
}

func InjectT[T any](di *Dix, opts ...Option) T {
	var data T
	if reflect.TypeOf(data).Kind() != reflect.Struct {
		panic("<T> type kind is not struct")
	}

	_ = di.Inject(&data, opts...)
	return data
}

func Provide(di *Dix, data any) { di.Provide(data) }
