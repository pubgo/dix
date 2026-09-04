package dix

import (
	"context"
	_ "embed"
	"log/slog"
	"reflect"
	"time"

	"github.com/pubgo/dix/v2/dixinternal"
)

//go:embed .version/VERSION
var version string

func Version() string { return version }

type (
	Option  = dixinternal.Option
	Options = dixinternal.Options
	Dix     = dixinternal.Dix
)

func SetLog(log slog.Handler) { dixinternal.SetLog(log) }

func WithValuesNull() Option { return dixinternal.WithValuesNull() }

func WithRejectEmptyCollections() Option { return dixinternal.WithRejectEmptyCollections() }

func WithProviderTimeout(timeout time.Duration) Option {
	return dixinternal.WithProviderTimeout(timeout)
}

func WithSlowProviderThreshold(threshold time.Duration) Option {
	return dixinternal.WithSlowProviderThreshold(threshold)
}

func WithTraceBuffer(n int) Option { return dixinternal.WithTraceBuffer(n) }

func New(opts ...Option) *Dix { return dixinternal.New(opts...) }

func Inject[T any](di *Dix, data T, opts ...Option) T {
	vp := reflect.ValueOf(data)
	if vp.Kind() == reflect.Struct {
		_ = di.Inject(&data, opts...)
	} else {
		_ = di.Inject(data, opts...)
	}

	return data
}

func InjectContext[T any](ctx context.Context, di *Dix, data T, opts ...Option) T {
	vp := reflect.ValueOf(data)
	if vp.Kind() == reflect.Struct {
		_ = di.InjectContext(ctx, &data, opts...)
	} else {
		_ = di.InjectContext(ctx, data, opts...)
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

func InjectTContext[T any](ctx context.Context, di *Dix, opts ...Option) T {
	var data T
	if reflect.TypeOf(data).Kind() != reflect.Struct {
		panic("<T> type kind is not struct")
	}

	_ = di.InjectContext(ctx, &data, opts...)
	return data
}

func Provide(di *Dix, data any) { di.Provide(data) }

func TryProvide(di *Dix, data any) error { return di.TryProvide(data) }

func TryInject(di *Dix, data any, opts ...Option) error {
	return di.TryInject(data, opts...)
}

func TryInjectContext(ctx context.Context, di *Dix, data any, opts ...Option) error {
	return di.TryInjectContext(ctx, data, opts...)
}
