package dixinternal

import (
	"fmt"
	"time"
)

const (
	// DefaultProviderTimeout is the default max execution time for a single provider call.
	// Set WithProviderTimeout(0) to disable timeout control.
	DefaultProviderTimeout = 15 * time.Second

	// DefaultSlowProviderThreshold is the default warning threshold for provider execution latency.
	// Set WithSlowProviderThreshold(0) to disable slow-provider warnings.
	DefaultSlowProviderThreshold = 2 * time.Second
)

type (
	Option  func(opts *Options)
	Options struct {
		// AllowValuesNull tolerates missing map/list dependencies by resolving
		// them as empty collections. Missing single-value dependencies always
		// fail regardless of this flag.
		AllowValuesNull bool

		// ProviderTimeout limits the maximum execution time of one provider call.
		// Zero means no timeout.
		// A timed-out call cannot be aborted: the provider is marked as failed and
		// will not be re-executed by later Inject/TryInject calls.
		ProviderTimeout time.Duration

		// SlowProviderThreshold emits warning log if provider execution is slower than this threshold.
		// Zero means no slow-warning threshold.
		SlowProviderThreshold time.Duration

		// TraceBuffer 为容器分配私有 trace 内存缓冲(条数)。
		// 0 表示不私有化:span 事件进入全局内存 sink(默认)。
		TraceBuffer int
	}
)

func (o Options) Validate() error {
	if o.ProviderTimeout < 0 {
		return fmt.Errorf("ProviderTimeout must be >= 0, got %s", o.ProviderTimeout)
	}
	if o.SlowProviderThreshold < 0 {
		return fmt.Errorf("SlowProviderThreshold must be >= 0, got %s", o.SlowProviderThreshold)
	}
	if o.TraceBuffer < 0 {
		return fmt.Errorf("TraceBuffer must be >= 0, got %d", o.TraceBuffer)
	}
	return nil
}

func WithValuesNull() Option {
	return func(opts *Options) {
		opts.AllowValuesNull = true
	}
}

// WithRejectEmptyCollections rejects injections of missing map/list dependencies
// instead of resolving them as empty collections. It is the counterpart of the
// default AllowValuesNull=true behavior.
func WithRejectEmptyCollections() Option {
	return func(opts *Options) {
		opts.AllowValuesNull = false
	}
}

func WithProviderTimeout(timeout time.Duration) Option {
	return func(opts *Options) {
		opts.ProviderTimeout = timeout
	}
}

func WithSlowProviderThreshold(threshold time.Duration) Option {
	return func(opts *Options) {
		opts.SlowProviderThreshold = threshold
	}
}

// WithTraceBuffer 为容器分配私有 trace 内存缓冲(条数,容量不足 FIFO 驱逐)。
// 默认 0:span 事件进入全局内存 sink。
func WithTraceBuffer(n int) Option {
	return func(opts *Options) {
		opts.TraceBuffer = n
	}
}
