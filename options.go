package gotdx

import (
	"path/filepath"
	"runtime"
)

const (
	_defaultTCPAddress      = "119.147.212.81:7709"
	_defaultExTCPAddress    = "112.74.214.43:7727"
	_defaultMacTCPAddress   = "121.36.248.138:7709"
	_defaultMacExTCPAddress = "116.205.135.205:7727"
	_defaultRetryTimes      = 3
	_defaultTimeoutSec      = 8
	_defaultPoolSize        = 3
)

// Options 保存客户端创建时使用的主站、扩展站、MAC 站地址与地址池、重试次数、超时等全部配置。
//
// 该结构体通常不直接构造：`New` 会先应用内置默认配置，再按传入顺序应用各 `Option` 覆盖
// 对应字段，后传入的选项优先生效。各字段的零值表示"沿用对应默认值"，字段级并发读写
// 应在构造阶段完成，连接建立后不应再修改。
type Options struct {
	TCPAddress             string   // 主行情服务器单点地址，格式为 `host:port`；空串时回退到主站地址池
	TCPAddressPool         []string // 主行情服务器地址池，连接时按顺序或测速结果依次尝试；空切片使用内置默认池
	ExTCPAddress           string   // 扩展行情服务器单点地址，格式为 `host:port`；空串时回退到扩展站地址池
	ExTCPAddressPool       []string // 扩展行情服务器地址池；空切片使用内置默认池
	MacTCPAddress          string   // MAC 主行情服务器单点地址，格式为 `host:port`；空串时回退到 MAC 主站地址池
	MacTCPAddressPool      []string // MAC 主行情服务器地址池；空切片使用内置默认池
	MacExTCPAddress        string   // MAC 扩展行情服务器单点地址，格式为 `host:port`；空串时回退到 MAC 扩展站地址池
	MacExTCPAddressPool    []string // MAC 扩展行情服务器地址池；空切片使用内置默认池
	MaxRetryTimes          int      // 单次请求写失败的重试次数；构造阶段由默认值 3 填充，连接流程中按该值重试
	TimeoutSec             int      // 连接与读写共用的超时时间，单位秒；构造阶段由默认值 8 秒填充，非正数回退默认
	PoolSize               int      // 连接池大小，即并行可用的 TCP 连接条数；构造阶段由默认值 3 填充，非正数回退默认
	DisableEmptyNodeSwitch bool     // 关闭空数据换节点：空应答直接返回首次结果，不并行探测其他连接；默认 `false` 保持池化换节点行为
	FuturesCalendarPath    string   // 期货交易日历 SQLite 路径；为空时不转换 MAC 期货自然时间
}

// defaultOptions 构造内置默认配置：各站点使用内置主机列表与兜底地址，并设置默认重试次数、超时与日历路径。
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
		PoolSize:            _defaultPoolSize,
		FuturesCalendarPath: defaultFuturesCalendarPath(),
	}
}

// defaultFuturesCalendarPath 返回随 `gotdx` 源码发布的默认期货交易日历路径。
func defaultFuturesCalendarPath() string {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("data", "futures-trade-calendar.db")
	}
	return filepath.Join(filepath.Dir(sourceFile), "data", "futures-trade-calendar.db")
}

// applyOptions 先填充默认配置，再按传入顺序依次执行 `Option` 覆盖对应字段；后传入的选项优先生效。
func applyOptions(opts ...Option) *Options {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Option 是用于修改 `Options` 的函数式选项，返回闭包在合并阶段按顺序应用到配置上。
type Option func(options *Options)

// WithTCPAddress 设置主行情服务器地址，格式为 `host:port`。
func WithTCPAddress(tcpAddress string) Option {
	return func(o *Options) {
		o.TCPAddress = tcpAddress
	}
}

// WithTCPAddressPool 设置主行情服务器地址池，连接时按顺序或测速结果依次尝试。
//
// 参数 `ips` 为 `host:port` 地址序列，整体替换原地址池；不传参数（空序列）时清空地址池。
func WithTCPAddressPool(ips ...string) Option {
	return func(o *Options) {
		o.TCPAddressPool = ips
	}
}

// WithExTCPAddress 设置扩展行情服务器地址，格式为 `host:port`。
func WithExTCPAddress(tcpAddress string) Option {
	return func(o *Options) {
		o.ExTCPAddress = tcpAddress
	}
}

// WithExTCPAddressPool 设置扩展行情服务器地址池，连接时按顺序或测速结果依次尝试。
//
// 参数 `ips` 为 `host:port` 地址序列，整体替换原地址池；不传参数时清空地址池。
func WithExTCPAddressPool(ips ...string) Option {
	return func(o *Options) {
		o.ExTCPAddressPool = ips
	}
}

// WithMacTCPAddress 设置 MAC 主行情服务器地址，格式为 `host:port`。
func WithMacTCPAddress(tcpAddress string) Option {
	return func(o *Options) {
		o.MacTCPAddress = tcpAddress
	}
}

// WithMacTCPAddressPool 设置 MAC 主行情服务器地址池，连接时按顺序或测速结果依次尝试。
//
// 参数 `ips` 为 `host:port` 地址序列，整体替换原地址池；不传参数时清空地址池。
func WithMacTCPAddressPool(ips ...string) Option {
	return func(o *Options) {
		o.MacTCPAddressPool = ips
	}
}

// WithMacExTCPAddress 设置 MAC 扩展行情服务器地址，格式为 `host:port`。
func WithMacExTCPAddress(tcpAddress string) Option {
	return func(o *Options) {
		o.MacExTCPAddress = tcpAddress
	}
}

// WithMacExTCPAddressPool 设置 MAC 扩展行情服务器地址池，连接时按顺序或测速结果依次尝试。
//
// 参数 `ips` 为 `host:port` 地址序列，整体替换原地址池；不传参数时清空地址池。
func WithMacExTCPAddressPool(ips ...string) Option {
	return func(o *Options) {
		o.MacExTCPAddressPool = ips
	}
}

// WithTimeoutSec 设置连接与读写超时时间，单位秒；传入非正数时保持默认超时。
func WithTimeoutSec(timeoutSec int) Option {
	return func(o *Options) {
		if timeoutSec > 0 {
			o.TimeoutSec = timeoutSec
		}
	}
}

// WithFuturesCalendarPath 设置 MAC 期货 K 线自然时间转换使用的 `SQLite` 交易日历路径。
//
// 参数 `path` 为日历文件路径；传入空串时关闭期货自然时间转换，保留服务端原始交易时间。
func WithFuturesCalendarPath(path string) Option {
	return func(o *Options) {
		o.FuturesCalendarPath = path
	}
}

// WithPoolSize 设置连接池大小，即并行可用的 TCP 连接条数。
//
// 参数 `poolSize` 为目标连接条数；传入非正数时保持默认池大小。池内连接各自独立握手，
// 请求按轮询分散到各连接，连接之间并行、单连接内部串行。
func WithPoolSize(poolSize int) Option {
	return func(o *Options) {
		if poolSize > 0 {
			o.PoolSize = poolSize
		}
	}
}

// WithSingleConnectionMode 开启显式单连接模式：把连接池大小设为 1，并关闭空数据换节点。
//
// 适用于单并发、追求窄尾延迟的调用方：全程复用一条长连接，获得接近老库固定单连接的稳定
// 延迟分布；断连自愈（心跳判死、后台补建、重平衡）仍然保留。默认关闭，不传该选项时保持
// 三连接池与空数据并行换节点行为。需要只关闭换节点、保留多连接池时改用
// `WithEmptyNodeSwitchDisabled(true)`。
func WithSingleConnectionMode() Option {
	return func(o *Options) {
		o.PoolSize = 1
		o.DisableEmptyNodeSwitch = true
	}
}

// WithEmptyNodeSwitchDisabled 设置是否关闭空数据换节点。
//
// 参数 `disabled` 为 `true` 时，空数据应答直接返回首次结果，不再并行探测池内其他连接、
// 不累计节点空数据健康度；为 `false`（默认）时保持原有并行换节点行为。该选项与
// `WithPoolSize` 相互独立，可单独关闭换节点而不改变池大小；单连接组合使用时优先选择
// `WithSingleConnectionMode`。
func WithEmptyNodeSwitchDisabled(disabled bool) Option {
	return func(o *Options) {
		o.DisableEmptyNodeSwitch = disabled
	}
}
