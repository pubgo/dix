package dixinternal

type (
	Option  func(opts *Options)
	Options struct {
		// AllowValuesNull allows result to be nil
		AllowValuesNull bool
	}
)

func (o Options) Merge(opt Options) Options {
	if o.AllowValuesNull {
		opt.AllowValuesNull = o.AllowValuesNull
	}
	return opt
}

func (o Options) Check() {
}

func WithValuesNull() Option {
	return func(opts *Options) {
		opts.AllowValuesNull = true
	}
}
