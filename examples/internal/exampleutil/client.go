// Package exampleutil 提供 `examples` 目录下各示例共用的客户端构造、内置主机地址与小工具函数，
// 屏蔽主站、扩展站、MAC 主站、MAC 扩展站之间的地址配置差异，使示例代码聚焦于协议调用本身。
package exampleutil

import "github.com/bensema/gotdx"

// NewMainClient 使用内置主行情服务器地址构造主站客户端。
//
// 本函数无业务入参；内置主站地址列表的首条作为主地址，其余地址进入候选地址池，拨号与读写超时
// 沿用库内默认值。构造过程不拨号、不返回错误，连接在首次请求时按需建立；返回的客户端不可由零值
// 直接使用，用完应调用 `Disconnect` 释放连接，拨号、登录握手与协议错误均由后续查询方法返回。
func NewMainClient() *gotdx.Client {
	primary, pool := splitHosts(gotdx.MainHostAddresses())
	return gotdx.New(
		gotdx.WithTCPAddress(primary),
		gotdx.WithTCPAddressPool(pool...),
	)
}

// NewMainClientForHost 使用指定主站地址构造主站客户端。
//
// 参数 `host` 为目标主站 `host:port` 地址；地址池保留内置主站候选地址，拨号与读写超时固定为 6 秒。
// 构造过程不拨号、不返回错误，连接在首次请求时按需建立；返回的客户端用完应调用 `Disconnect`
// 释放连接，拨号与协议错误由后续查询方法返回。本构造器适用于示例单独指定探测到的可用主机的场景。
func NewMainClientForHost(host string) *gotdx.Client {
	return gotdx.New(
		gotdx.WithTCPAddress(host),
		gotdx.WithTimeoutSec(6),
	)
}

// NewExClient 使用内置扩展行情服务器地址构造扩展站客户端。
//
// 本函数无业务入参；内置扩展站地址列表的首条作为主地址，其余地址进入候选地址池，拨号与读写超时
// 固定为 6 秒。构造过程不拨号、不返回错误，连接在首次请求时按需建立；返回的客户端用完应调用
// `Disconnect` 释放连接，拨号、登录握手与协议错误由后续查询方法返回。
func NewExClient() *gotdx.Client {
	primary, pool := splitHosts(gotdx.ExHostAddresses())
	return gotdx.New(
		gotdx.WithExTCPAddress(primary),
		gotdx.WithExTCPAddressPool(pool...),
		gotdx.WithTimeoutSec(6),
	)
}

// NewExClientForHost 使用指定扩展站地址构造扩展站客户端。
//
// 参数 `host` 为目标扩展站 `host:port` 地址；拨号与读写超时固定为 6 秒。构造过程不拨号、不返回
// 错误，连接在首次请求时按需建立；返回的客户端用完应调用 `Disconnect` 释放连接，拨号与协议错误
// 由后续查询方法返回。
func NewExClientForHost(host string) *gotdx.Client {
	return gotdx.New(
		gotdx.WithExTCPAddress(host),
		gotdx.WithTimeoutSec(6),
	)
}

// NewExSingleConnectionClient 构造单连接模式的扩展站客户端：连接池大小为 1 并关闭空数据
// 换节点，适用于单并发、追求窄尾延迟的调用方；断连自愈（心跳判死、后台补建）仍然保留。
//
// 本函数无业务入参；内置扩展站地址列表首条作为主地址，其余进入候选池，拨号与读写超时
// 固定为 6 秒。构造过程不拨号、不返回错误，连接在首次请求时按需建立；返回的客户端用完
// 应调用 `Disconnect` 释放连接，拨号、登录握手与协议错误由后续查询方法返回。
func NewExSingleConnectionClient() *gotdx.Client {
	primary, pool := splitHosts(gotdx.ExHostAddresses())
	return gotdx.NewEx(
		gotdx.WithExTCPAddress(primary),
		gotdx.WithExTCPAddressPool(pool...),
		gotdx.WithTimeoutSec(6),
		gotdx.WithSingleConnectionMode(),
	)
}

// NewUnifiedClient 构造同时配置主站与扩展站两套候选地址的统一客户端。
//
// 本函数无业务入参；主站、扩展站各自取内置地址列表的首条为主地址、其余为候选池，拨号与读写超时
// 固定为 6 秒。构造过程不拨号、不返回错误，连接在首次请求对应站点时按需建立；返回的客户端可分别
// 调用主站与扩展站接口，用完应调用 `Disconnect` 级联释放两条连接，拨号与协议错误由后续查询方法
// 返回。
func NewUnifiedClient() *gotdx.Client {
	mainPrimary, mainPool := splitHosts(gotdx.MainHostAddresses())
	exPrimary, exPool := splitHosts(gotdx.ExHostAddresses())
	return gotdx.New(
		gotdx.WithTCPAddress(mainPrimary),
		gotdx.WithTCPAddressPool(mainPool...),
		gotdx.WithExTCPAddress(exPrimary),
		gotdx.WithExTCPAddressPool(exPool...),
		gotdx.WithTimeoutSec(6),
	)
}

// NewMACClient 使用内置 MAC 主站地址构造 MAC 客户端。
//
// 本函数无业务入参；内置 MAC 主站地址列表的首条作为主地址，其余地址进入候选地址池，拨号与读写
// 超时固定为 6 秒。构造过程不拨号、不返回错误，连接在首次请求时按需建立；返回的客户端用完应调用
// `Disconnect` 释放连接，拨号、登录握手与协议错误由后续查询方法返回。
func NewMACClient() *gotdx.Client {
	primary, pool := splitHosts(gotdx.MACHostAddresses())
	return gotdx.NewMAC(
		gotdx.WithMacTCPAddress(primary),
		gotdx.WithMacTCPAddressPool(pool...),
		gotdx.WithTimeoutSec(6),
	)
}

// NewMACExClient 使用内置 MAC 扩展站地址构造 MAC 扩展客户端。
//
// 本函数无业务入参；内置 MAC 扩展站地址列表的首条作为主地址，其余地址进入候选地址池，拨号与读写
// 超时固定为 6 秒。构造过程不拨号、不返回错误，连接在首次请求时按需建立；返回的客户端用完应调用
// `Disconnect` 释放连接，拨号、登录握手与协议错误由后续查询方法返回。
func NewMACExClient() *gotdx.Client {
	primary, pool := splitHosts(gotdx.MACExHostAddresses())
	return gotdx.NewMACEx(
		gotdx.WithMacExTCPAddress(primary),
		gotdx.WithMacExTCPAddressPool(pool...),
		gotdx.WithTimeoutSec(6),
	)
}

// MainHosts 返回内置主行情服务器的 `host:port` 地址列表副本。
//
// 本函数无入参；返回切片为独立副本，调用方增删元素不会影响内置地址列表；内置列表为空时返回空切片。
func MainHosts() []string {
	return gotdx.MainHostAddresses()
}

// ExHosts 返回内置扩展行情服务器的 `host:port` 地址列表副本。
//
// 本函数无入参；返回切片为独立副本，调用方增删元素不会影响内置地址列表；内置列表为空时返回空切片。
func ExHosts() []string {
	return gotdx.ExHostAddresses()
}

// MACHosts 返回内置 MAC 主行情服务器的 `host:port` 地址列表副本。
//
// 本函数无入参；返回切片为独立副本，调用方增删元素不会影响内置地址列表；内置列表为空时返回空切片。
func MACHosts() []string {
	return gotdx.MACHostAddresses()
}

// MACExHosts 返回内置 MAC 扩展行情服务器的 `host:port` 地址列表副本。
//
// 本函数无入参；返回切片为独立副本，调用方增删元素不会影响内置地址列表；内置列表为空时返回空切片。
func MACExHosts() []string {
	return gotdx.MACExHostAddresses()
}

// splitHosts 把地址列表拆成主地址与候选地址池。
//
// 参数 `hosts` 为 `host:port` 地址序列；空列表返回空主地址与 `nil` 候选池；非空列表首条作为主地址，
// 其余元素复制为候选池切片副本，避免调用方改动影响内置地址列表。
func splitHosts(hosts []string) (string, []string) {
	if len(hosts) == 0 {
		return "", nil
	}
	return hosts[0], append([]string(nil), hosts[1:]...)
}

// Min 返回两个整数中较小的一个，供示例在截取有限条数时使用。
//
// 参数 `a`、`b` 为待比较的两个整数；二者相等时返回 `a`；返回值即参与比较的入参之一，不做类型
// 转换或额外边界处理。
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// PreviewHex 截取原始十六进制片段用于日志预览。
//
// 参数 `value` 为待预览的原始字符串；参数 `limit` 为允许展示的最大字符数，`value` 长度不超过
// `limit` 时原样返回，否则返回前 `limit` 个字符；`limit` 为 0 时返回空串。
func PreviewHex(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
