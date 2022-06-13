package dix

type Option func(opts *Options)
type Options struct {
	// 允许结果为nil
	allowNil bool
}

func (o Options) Check() {
}

func AllowNil() Option {
	return func(opts *Options) {
		opts.allowNil = true
	}
}
