package gotdx

const (
	_defaultTCPAddress = "119.147.212.81:7709"
	_defaultRetryTimes = 3
)

type Options struct {
	TCPAddress    string // 服务器地址
	MaxRetryTimes int    // 重试次数
}

func defaultOptions() *Options {
	return &Options{
		TCPAddress:    _defaultTCPAddress,
		MaxRetryTimes: _defaultRetryTimes,
	}
}

func applyOptions(opts ...Option) *Options {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	return o
}

type Option func(options *Options)

func WithTCPAddress(tcpAddress string) Option {
	return func(o *Options) {
		o.TCPAddress = tcpAddress
	}
}
