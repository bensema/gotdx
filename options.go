package gotdx

const (
	_defaultTCPAddress   = "119.147.212.81:7709"
	_defaultExTCPAddress = "112.74.214.43:7727"
	_defaultRetryTimes   = 3
	_defaultTimeoutSec   = 8
)

type Options struct {
	TCPAddress       string   // 主行情服务器地址
	TCPAddressPool   []string // 主行情服务器地址池
	ExTCPAddress     string   // 扩展行情服务器地址
	ExTCPAddressPool []string // 扩展行情服务器地址池
	MaxRetryTimes    int      // 重试次数
	TimeoutSec       int      // 连接和读写超时时间，单位秒
}

func defaultOptions() *Options {
	return &Options{
		TCPAddress:    _defaultTCPAddress,
		ExTCPAddress:  _defaultExTCPAddress,
		MaxRetryTimes: _defaultRetryTimes,
		TimeoutSec:    _defaultTimeoutSec,
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

func WithTCPAddressPool(ips ...string) Option {
	return func(o *Options) {
		o.TCPAddressPool = ips
	}
}

func WithExTCPAddress(tcpAddress string) Option {
	return func(o *Options) {
		o.ExTCPAddress = tcpAddress
	}
}

func WithExTCPAddressPool(ips ...string) Option {
	return func(o *Options) {
		o.ExTCPAddressPool = ips
	}
}

func WithTimeoutSec(timeoutSec int) Option {
	return func(o *Options) {
		if timeoutSec > 0 {
			o.TimeoutSec = timeoutSec
		}
	}
}
