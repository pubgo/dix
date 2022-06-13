package dix

type Option func(opts *Options)
type Options struct {
}

func (o Options) Check() {
}
