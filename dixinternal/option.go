package dixinternal

import (
	"fmt"
	"time"
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

func (o Options) Merge(opt Options) Options {
	if o.AllowValuesNull {
		opt.AllowValuesNull = o.AllowValuesNull
	}
	if o.ProviderTimeout > 0 {
		opt.ProviderTimeout = o.ProviderTimeout
	}
	if o.SlowProviderThreshold > 0 {
		opt.SlowProviderThreshold = o.SlowProviderThreshold
	}
	return opt
}

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
