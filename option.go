package dix

type Option func(opts *Options)
type Options struct {
	// 允许结果为nil
	AllowNil bool
}

func (o Options) Merge(opt Options) Options {
	if o.AllowNil {
		opt.AllowNil = o.AllowNil
	}
	return opt
}

func (o Options) Check() {
}

func AllowNil() Option {
	return func(opts *Options) {
		opts.AllowNil = true
	}
}
