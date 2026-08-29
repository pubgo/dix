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
		// AllowValuesNull allows result to be nil
		AllowValuesNull bool

		// ProviderTimeout limits the maximum execution time of one provider call.
		// Zero means no timeout.
		ProviderTimeout time.Duration

		// SlowProviderThreshold emits warning log if provider execution is slower than this threshold.
		// Zero means no slow-warning threshold.
		SlowProviderThreshold time.Duration
	}
)

func (o Options) Validate() error {
	if o.ProviderTimeout < 0 {
		return fmt.Errorf("ProviderTimeout must be >= 0, got %s", o.ProviderTimeout)
	}
	if o.SlowProviderThreshold < 0 {
		return fmt.Errorf("SlowProviderThreshold must be >= 0, got %s", o.SlowProviderThreshold)
	}
	return nil
}

func WithValuesNull() Option {
	return func(opts *Options) {
		opts.AllowValuesNull = true
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
