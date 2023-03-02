package dix_inter

type Option func(opts *Options)
type Options struct {
	// 允许结果为nil
	AllowValuesNull bool
}

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
