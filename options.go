package gotdx

const (
	_defaultTCPAddress      = "119.147.212.81:7709"
	_defaultExTCPAddress    = "112.74.214.43:7727"
	_defaultMacTCPAddress   = "121.36.248.138:7709"
	_defaultMacExTCPAddress = "116.205.135.205:7727"
	_defaultRetryTimes      = 3
	_defaultTimeoutSec      = 8
)

type Options struct {
	TCPAddress          string   // 主行情服务器地址
	TCPAddressPool      []string // 主行情服务器地址池
	ExTCPAddress        string   // 扩展行情服务器地址
	ExTCPAddressPool    []string // 扩展行情服务器地址池
	MacTCPAddress       string   // MAC 主行情服务器地址
	MacTCPAddressPool   []string // MAC 主行情服务器地址池
	MacExTCPAddress     string   // MAC 扩展行情服务器地址
	MacExTCPAddressPool []string // MAC 扩展行情服务器地址池
	AutoSelectFastest   bool     // 连接前先对地址池做 TCP 测速并优先尝试低延迟节点
	MaxRetryTimes       int      // 重试次数
	TimeoutSec          int      // 连接和读写超时时间，单位秒
}

func defaultOptions() *Options {
	mainAddress, mainPool := defaultAddressAndPool(MainHostAddresses(), _defaultTCPAddress)
	exAddress, exPool := defaultAddressAndPool(ExHostAddresses(), _defaultExTCPAddress)
	macAddress, macPool := defaultAddressAndPool(MACHostAddresses(), _defaultMacTCPAddress)
	macExAddress, macExPool := defaultAddressAndPool(MACExHostAddresses(), _defaultMacExTCPAddress)

	return &Options{
		TCPAddress:          mainAddress,
		TCPAddressPool:      mainPool,
		ExTCPAddress:        exAddress,
		ExTCPAddressPool:    exPool,
		MacTCPAddress:       macAddress,
		MacTCPAddressPool:   macPool,
		MacExTCPAddress:     macExAddress,
		MacExTCPAddressPool: macExPool,
		MaxRetryTimes:       _defaultRetryTimes,
		TimeoutSec:          _defaultTimeoutSec,
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

func WithMacTCPAddress(tcpAddress string) Option {
	return func(o *Options) {
		o.MacTCPAddress = tcpAddress
	}
}

func WithMacTCPAddressPool(ips ...string) Option {
	return func(o *Options) {
		o.MacTCPAddressPool = ips
	}
}

func WithMacExTCPAddress(tcpAddress string) Option {
	return func(o *Options) {
		o.MacExTCPAddress = tcpAddress
	}
}

func WithMacExTCPAddressPool(ips ...string) Option {
	return func(o *Options) {
		o.MacExTCPAddressPool = ips
	}
}

func WithTimeoutSec(timeoutSec int) Option {
	return func(o *Options) {
		if timeoutSec > 0 {
			o.TimeoutSec = timeoutSec
		}
	}
}

// WithAutoSelectFastest enables TCP probe sorting before connection attempts.
func WithAutoSelectFastest(enabled bool) Option {
	return func(o *Options) {
		o.AutoSelectFastest = enabled
	}
}
