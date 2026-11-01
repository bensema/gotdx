package gotdx

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/bensema/gotdx/proto"
	"github.com/bensema/gotdx/types"
)

type clientMode uint8

const (
	clientModeMain clientMode = iota
	clientModeEx
	clientModeMacMain
	clientModeMacEx
)

// New 创建连接通达信主行情服务器的客户端。
//
// 参数 `opts` 为按顺序覆盖内置默认配置的函数式选项，后传入的选项优先生效；不传选项时使用内置
// 主站地址池、默认超时与重试次数。返回的 `Client` 零值不可用，须由 `New` 系列构造；其方法可
// 并发调用，同一连接上的请求由客户端内部串行化。
func New(opts ...Option) *Client {
	return newClientWithOptions(applyOptions(opts...), clientModeMain)
}

// NewEx 创建连接通达信扩展行情服务器（港股、美股、期货等）的客户端。
//
// 参数 `opts` 为按顺序覆盖内置默认配置的函数式选项，后传入的选项优先生效；不传选项时使用内置
// 扩展站地址池与默认超时。返回的 `Client` 零值不可用，方法可并发调用。
func NewEx(opts ...Option) *Client {
	return newClientWithOptions(applyOptions(opts...), clientModeEx)
}

// NewMAC 创建连接 MAC 行情主站的客户端。
//
// 参数 `opts` 为按顺序覆盖内置默认配置的函数式选项，后传入的选项优先生效；不传选项时使用内置
// MAC 主站地址池与默认超时。返回的 `Client` 零值不可用，方法可并发调用。
func NewMAC(opts ...Option) *Client {
	return newClientWithOptions(applyOptions(opts...), clientModeMacMain)
}

// NewMACEx 创建连接 MAC 扩展行情服务器的客户端。
//
// 参数 `opts` 为按顺序覆盖内置默认配置的函数式选项，后传入的选项优先生效；不传选项时使用内置
// MAC 扩展站地址池与默认超时。返回的 `Client` 零值不可用，方法可并发调用。
func NewMACEx(opts ...Option) *Client {
	return newClientWithOptions(applyOptions(opts...), clientModeMacEx)
}

// Client 是通达信行情协议客户端，封装 TCP 连接池、握手、协议收发以及主站、扩展站、MAC 站的高层接口。
//
// Client 须由 `New`、`NewEx`、`NewMAC`、`NewMACEx` 构造，零值不可用；各方法可并发调用，请求
// 按轮询分散到池内各连接：连接之间并行、单连接上的请求由该连接内部串行化，调用方无需自行加锁。
type Client struct {
	opt                 *Options      // 深拷贝后的连接配置
	pool                []*pooledConn // 连接池：仅存放空闲连接，借出的连接移出池、归还时放回
	mu                  sync.Mutex    // 池管理锁：保护借还、补建、关闭与池状态
	closed              bool          // 池已关闭标记，关闭后拒绝借出
	inUse               int           // 借出中的连接计数，供补建判断容量时与池内数量合计
	lastAddr            string        // 池内最近一条连接的实际地址，供池空时诊断兜底
	lastDataAddr        string        // 最近一次历史查询命中数据的服务器地址，空数据重试优先使用
	next                int           // 轮询游标：借出连接时按此下标取池内连接并自增
	done                chan struct{} // 补建循环停止信号，`Disconnect` 时关闭
	refill              sync.Once     // 补建循环只启动一次
	mode                clientMode
	main                *Client
	ex                  *Client
	decimalPoints       map[uint8]map[string]int8
	decimalPointsLoaded map[uint8]bool
	lastReachable       []string                 // 最近一次探活确认可达的地址列表，按测得耗时升序，供重平衡与补建收敛候选
	addrLatency         map[string]time.Duration // 各地址最近一次建连测得的拨号+握手总耗时，供候选排序与后台预比较；池锁保护
	dialSamples         []time.Duration          // 最近若干次成功建连耗时样本（环形窗口），供快路径超时取中位数自适应；池锁保护
	health              map[string]*nodeHealth   // 各地址连续空数据健康度统计，定向借出对降权节点跳过；池锁保护
	heartbeatLatency    map[string]time.Duration // 各地址最近一次心跳往返耗时，供慢连接判定与池内分布比较；池锁保护
	slow                map[string]bool          // 心跳识别出的慢连接标记，触发缩短重平衡周期与放宽切换阈值；池锁保护
	lastActivityAt      time.Time                // 最近一次业务借出连接的时间，供心跳周期按活动度拉长；池锁保护
	rebalanceMu         sync.Mutex               // 重平衡串行锁：避免定时 tick 与全空后台修复并发重建连接
	rebalanceTick       int                      // 重平衡执行计数，每 5 次触发一次全量探活刷新可达集
}

// pooledConn 为池内一条 TCP 连接：自带串行锁，保证该连接上的请求严格
// 先写后读、帧边界不被并发破坏。
type pooledConn struct {
	conn       net.Conn
	mu         sync.Mutex // 该连接串行锁
	addr       string     // 实际连通的 `host:port` 地址
	hello      any        // 建连时模式握手的应答，仅建池返回首个应答时使用
	lastUsedAt time.Time  // 最近一次被业务请求借出的时间，供心跳跳过近期活跃连接；池锁保护
}

// fastPathConnRes 为建池快路径单条并发建连结果：连接、握手应答、错误与本次拨号+握手
// 总耗时；耗时用于候选排序与后台预比较，不参与公开 API。
type fastPathConnRes struct {
	pc      *pooledConn
	reply   any
	err     error
	elapsed time.Duration
}

// CurrentAddress 返回当前池内首条连接实际连通的服务器 `host:port` 地址。
//
// 本方法无业务入参；客户端指针或内部选项为空时返回空字符串；已建立连接池时返回首条连接的
// 实际地址；池内无空闲连接（全部借出待归还）时返回最近一次入池连接的实际地址，未拨号时
// 返回配置的单点地址。
func (client *Client) CurrentAddress() string {
	if client == nil || client.opt == nil {
		return ""
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.pool) > 0 {
		return client.pool[0].addr
	}
	if client.lastAddr != "" {
		return client.lastAddr
	}
	return client.opt.TCPAddress
}

// ProbeHosts 并发探测客户端已配置的全部地址，返回按可达性优先、同可达时延迟从低到高排序的
// 探测结果；每条结果含地址、可达标志与拨号往返耗时，不可达条目同时携带失败原因。
//
// 本方法无业务入参；客户端指针或内部选项为空时返回 `nil`；探测使用当前配置的连接超时。
func (client *Client) ProbeHosts() []HostProbeResult {
	if client == nil || client.opt == nil {
		return nil
	}
	return ProbeAddresses(client.addresses(), client.timeout())
}

// FastestHost 返回已配置地址中延迟最低且可达的一条探测结果。
//
// 本方法无业务入参；全部地址不可达，或客户端指针、内部选项为空时返回零值 `HostProbeResult` 与
// `ErrNoReachableHosts` 错误；探测使用当前配置的连接超时。
func (client *Client) FastestHost() (HostProbeResult, error) {
	if client == nil || client.opt == nil {
		return HostProbeResult{}, ErrNoReachableHosts
	}
	return FastestAddress(client.addresses(), client.timeout())
}

// newClientWithOptions 基于选项与站点模式构造客户端。
//
// 参数 `opt` 为待深拷贝的配置，空指针时回退到默认配置；参数 `mode` 标识主站、扩展站、MAC 主站或
// MAC 扩展站，不同模式把对应站点的单点地址与地址池切换为后续拨号使用的默认目标。
func newClientWithOptions(opt *Options, mode clientMode) *Client {
	client := &Client{
		opt:  cloneOptions(opt),
		mode: mode,
	}
	// 连接池大小非正数时回退默认，避免零值构造出不可用池。
	if client.opt.PoolSize <= 0 {
		client.opt.PoolSize = _defaultPoolSize
	}

	// 扩展站 / MAC 站点模式：用对应站点的地址与地址池覆盖主站连接配置。
	if mode == clientModeEx {
		client.opt.TCPAddress = client.opt.ExTCPAddress
		client.opt.TCPAddressPool = append([]string(nil), client.opt.ExTCPAddressPool...)
	}
	if mode == clientModeMacMain {
		client.opt.TCPAddress = client.opt.MacTCPAddress
		client.opt.TCPAddressPool = append([]string(nil), client.opt.MacTCPAddressPool...)
	}
	if mode == clientModeMacEx {
		client.opt.TCPAddress = client.opt.MacExTCPAddress
		client.opt.TCPAddressPool = append([]string(nil), client.opt.MacExTCPAddressPool...)
	}

	return client
}

// cloneOptions 深拷贝 `Options`。
//
// 参数 `opt` 为空时返回默认配置；结构体值复制后，四个地址池单独分配切片，避免不同客户端的连接
// 配置共享底层切片。
func cloneOptions(opt *Options) *Options {
	if opt == nil {
		return defaultOptions()
	}
	clone := *opt
	clone.TCPAddressPool = append([]string(nil), opt.TCPAddressPool...)
	clone.ExTCPAddressPool = append([]string(nil), opt.ExTCPAddressPool...)
	clone.MacTCPAddressPool = append([]string(nil), opt.MacTCPAddressPool...)
	clone.MacExTCPAddressPool = append([]string(nil), opt.MacExTCPAddressPool...)
	return &clone
}

// addresses 汇总单点地址与地址池，返回 `host:port` 地址序列；单点地址排在前面优先尝试，地址池
// 按配置顺序跟随其后。
func (client *Client) addresses() []string {
	addresses := make([]string, 0, 1+len(client.opt.TCPAddressPool))
	if client.opt.TCPAddress != "" {
		addresses = append(addresses, client.opt.TCPAddress)
	}
	addresses = append(addresses, client.opt.TCPAddressPool...)
	return addresses
}

// timeout 返回本次连接与读写共用的超时时长。
//
// 客户端、选项缺失，或 `Options.TimeoutSec` 为非正数时回退到内置默认超时秒数。
func (client *Client) timeout() time.Duration {
	if client == nil || client.opt == nil {
		return time.Duration(_defaultTimeoutSec) * time.Second
	}
	timeout := time.Duration(client.opt.TimeoutSec) * time.Second
	if timeout <= 0 {
		return time.Duration(_defaultTimeoutSec) * time.Second
	}
	return timeout
}

// buildPoolConn 拨号指定地址并在该连接上完成当前模式握手，返回可入池连接与其握手应答。
//
// 参数 `address` 为 `host:port` 形式地址；拨号或握手失败时关闭连接并返回错误。主站与扩展站
// 各自完成对应握手协议，MAC 站拨号即视为连接成功、应答为 `nil`。
// buildPoolConn 按默认超时建连：拨号后开启 TCP 保活并执行当前模式握手，返回
// 已握手的连接与握手应答。建池慢路径、补建与切换使用；快路径走
// `buildPoolConnWithTimeout`（短拨号超时排除偶发慢样本）。
func (client *Client) buildPoolConn(address string) (*pooledConn, any, error) {
	return client.buildPoolConnWithTimeout(address, client.timeout())
}

// buildPoolConnWithTimeout 在指定拨号超时内建连：拨号后开启 TCP 保活并执行当前
// 模式握手，返回已握手的连接与握手应答。
func (client *Client) buildPoolConnWithTimeout(address string, dialTimeout time.Duration) (*pooledConn, any, error) {
	conn, err := net.DialTimeout("tcp", address, dialTimeout)
	if err != nil {
		return nil, nil, err
	}
	// 开启 TCP 保活：行情长连接可能长时间无请求，避免中间设备（NAT/防火墙）
	// 或服务端静默回收空闲连接导致下次请求失败。
	if tc, ok := conn.(*net.TCPConn); ok {
		_ = tc.SetKeepAlive(true)
		_ = tc.SetKeepAlivePeriod(15 * time.Second)
	}
	pc := &pooledConn{conn: conn, addr: address}
	// 握手按连接串行锁执行（此时池未并发，锁仅为形态一致）。
	pc.mu.Lock()
	reply, err := client.handshakeOn(pc)
	pc.mu.Unlock()
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	return pc, reply, nil
}

// handshakeOn 在指定连接上执行当前模式对应的握手协议，返回解析后的握手应答。
//
// 主站模式发送 `Hello1` 请求并解析应答，扩展站模式发送 `ExLogin` 请求并解析应答；
// MAC 主站与 MAC 扩展站无握手协议，直接返回 `nil`。握手失败时返回对应错误。
func (client *Client) handshakeOn(pc *pooledConn) (any, error) {
	switch client.mode {
	case clientModeEx:
		obj := proto.NewExLogin()
		header, payload, err := client.exchange(pc.conn, obj)
		if err != nil {
			return nil, err
		}
		if err := obj.ParseResponse(header, payload); err != nil {
			return nil, err
		}
		return obj.Response(), nil
	case clientModeMacMain, clientModeMacEx:
		return nil, nil
	default:
		obj := proto.NewHello1()
		header, payload, err := client.exchange(pc.conn, obj)
		if err != nil {
			return nil, err
		}
		if err := obj.ParseResponse(header, payload); err != nil {
			return nil, err
		}
		return obj.Response(), nil
	}
}

// removeAddress 从地址切片中剔除指定地址，返回新切片；被剔除地址可能为
// 重复项，全部剔除后剩余切片可为空。
func removeAddress(addresses []string, target string) []string {
	out := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		if addr != target {
			out = append(out, addr)
		}
	}
	return out
}

// acquire 从池内借出一条连接，调用方随后须持该连接串行锁执行请求。
//
// 借出即从池中移出：在途连接不再参与心跳巡检与重平衡切换，避免巡检或切换误关正在
// 使用的连接导致在途请求中断。池已关闭或池内暂无连接（未连接或连接全部被借出/丢弃
// 待补建）时返回"无可用连接"错误。
//
// 定向优先：存在"最近成功响应的服务器"（`lastDataAddr`）时，优先借出同地址连接，
// 让请求稳定落在已验证可用的节点上（模拟老库固定单节点的稳定性）；定向节点连续空
// 数据被降权，或池内无该地址（已被重平衡切换或丢弃）时回退轮询；轮询同样优先跳过
// 降权节点，全部降权时回退原轮询兜底。
func (client *Client) acquire() (*pooledConn, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closed || len(client.pool) == 0 {
		return nil, errors.New("no available connection")
	}
	// 定向节点连续空数据达到降权阈值时跳过定向、回退轮询，避免继续绑定数据不全的节点。
	if client.lastDataAddr != "" && !client.nodeHealthDegradedLocked(client.lastDataAddr) {
		for i, pc := range client.pool {
			if pc.addr == client.lastDataAddr {
				client.pool = append(client.pool[:i], client.pool[i+1:]...)
				client.inUse++
				client.markActivityLocked(pc)
				return pc, nil
			}
		}
	}
	// 轮询同样优先跳过降权节点；池内存在健康连接时只在健康连接间轮询，全部降权时
	// 回退原轮询，避免健康连接被抽空后无连接可借。
	candidates := client.pool
	healthy := make([]*pooledConn, 0, len(client.pool))
	for _, pc := range client.pool {
		if !client.nodeHealthDegradedLocked(pc.addr) {
			healthy = append(healthy, pc)
		}
	}
	if len(healthy) > 0 {
		candidates = healthy
	}
	idx := client.next % len(candidates)
	client.next++
	pc := candidates[idx]
	for i, p := range client.pool {
		if p == pc {
			client.pool = append(client.pool[:i], client.pool[i+1:]...)
			break
		}
	}
	client.inUse++
	client.markActivityLocked(pc)
	return pc, nil
}

// markActivityLocked 记录本次业务借出时间到客户端活动时间与连接自身时间戳；调用方须已
// 持有池锁。近期活跃连接本轮心跳直接跳过，业务收发本身已证明连接存活，避免重复心跳流量。
func (client *Client) markActivityLocked(pc *pooledConn) {
	now := time.Now()
	client.lastActivityAt = now
	if pc != nil {
		pc.lastUsedAt = now
	}
}

// poolAppendLocked 将一条连接追加进池内并记录其实际地址（供池空时诊断兜底）；
// 调用方须已持有池锁。
func (client *Client) poolAppendLocked(pc *pooledConn) {
	client.pool = append(client.pool, pc)
	client.lastAddr = pc.addr
}

// recordAddrLatencyLocked 记录指定地址最近一次建连的拨号+握手总耗时；空地址或非正
// 耗时忽略。调用方须已持有池锁，映射按需惰性初始化；同时把耗时沉淀进自适应样本窗口。
func (client *Client) recordAddrLatencyLocked(addr string, elapsed time.Duration) {
	if addr == "" || elapsed <= 0 {
		return
	}
	if client.addrLatency == nil {
		client.addrLatency = make(map[string]time.Duration)
	}
	client.addrLatency[addr] = elapsed
	client.recordDialSampleLocked(elapsed)
}

// recordDialSampleLocked 把一次成功建连耗时追加到环形样本窗口，超出窗口长度时丢弃最旧
// 样本；非正耗时忽略。调用方须已持有池锁。
func (client *Client) recordDialSampleLocked(elapsed time.Duration) {
	if elapsed <= 0 {
		return
	}
	client.dialSamples = append(client.dialSamples, elapsed)
	if len(client.dialSamples) > dialSampleLimit {
		client.dialSamples = client.dialSamples[len(client.dialSamples)-dialSampleLimit:]
	}
}

// adaptiveFastDialTimeoutLocked 按历史建连耗时中位数计算快路径拨号超时。
//
// 方案决策：样本不足 `dialSampleMinCount` 时直接返回默认 `fastDialTimeout`，首次建池行为
// 与现状一致；样本充足时取样本中位数并乘以余量系数，再钳制在 `fastDialMinTimeout` 与
// `fastDialMaxTimeout` 之间——慢区域节点历史耗时偏高时自动放宽，避免被固定短超时误杀进
// 慢路径；快网络下中位数很低，结果被下限钳制回 200ms，不放松对偶发慢样本的排除。调用方
// 须已持有池锁。
func (client *Client) adaptiveFastDialTimeoutLocked() time.Duration {
	if len(client.dialSamples) < dialSampleMinCount {
		return fastDialTimeout
	}
	samples := make([]time.Duration, len(client.dialSamples))
	copy(samples, client.dialSamples)
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	mid := len(samples) / 2
	median := samples[mid]
	if len(samples)%2 == 0 {
		median = (samples[mid-1] + samples[mid]) / 2
	}
	candidate := time.Duration(dialTimeoutHeadroom * float64(median))
	if candidate < fastDialMinTimeout {
		candidate = fastDialMinTimeout
	}
	if candidate > fastDialMaxTimeout {
		candidate = fastDialMaxTimeout
	}
	return candidate
}

// recordHeartbeatLatencyLocked 记录指定地址最近一次心跳往返耗时；空地址或非正耗时
// 忽略。调用方须已持有池锁，映射按需惰性初始化。
func (client *Client) recordHeartbeatLatencyLocked(addr string, elapsed time.Duration) {
	if addr == "" || elapsed <= 0 {
		return
	}
	if client.heartbeatLatency == nil {
		client.heartbeatLatency = make(map[string]time.Duration)
	}
	client.heartbeatLatency[addr] = elapsed
}

// medianLatencyLocked 返回指定地址集合在耗时映射中的中位数；无有效样本返回 `0`，调用方
// 据此保持"样本不足不判定慢节点"的保守行为。调用方须已持有池锁。
func medianLatencyLocked(lat map[string]time.Duration, addrs []string) time.Duration {
	samples := make([]time.Duration, 0, len(addrs))
	for _, addr := range addrs {
		if d := lat[addr]; d > 0 {
			samples = append(samples, d)
		}
	}
	if len(samples) == 0 {
		return 0
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	mid := len(samples) / 2
	if len(samples)%2 == 1 {
		return samples[mid]
	}
	return (samples[mid-1] + samples[mid]) / 2
}

// hasSlowNode 报告当前是否存在心跳标记的慢连接，供补建循环动态调整重平衡周期；无慢
// 连接时返回 `false`，周期维持默认 1 分钟。
func (client *Client) hasSlowNode() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	for _, ok := range client.slow {
		if ok {
			return true
		}
	}
	return false
}

// isIdle 报告客户端是否处于低流量空闲状态：最近一次业务借出超过空闲阈值即视为空闲，
// 心跳周期拉长；从未发生业务借出时按活跃处理，保持默认心跳频率保证存活探测。
func (client *Client) isIdle() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.lastActivityAt.IsZero() {
		return false
	}
	return time.Since(client.lastActivityAt) > idleActivityThreshold
}

// refreshReachableLocked 把本次测得耗时的候选并入可达列表并按耗时升序重排：历史可达
// 地址保留，新测得样本覆盖旧耗时；未测到耗时的历史地址排到已知耗时样本之后。调用方须
// 已持有池锁。排序只影响内部候选落点，不改变公开 API 与默认行为。
func (client *Client) refreshReachableLocked(measured map[string]time.Duration) {
	if len(measured) == 0 {
		return
	}
	merged := make(map[string]time.Duration, len(measured)+len(client.lastReachable))
	for _, addr := range client.lastReachable {
		merged[addr] = client.addrLatency[addr]
	}
	for addr, elapsed := range measured {
		merged[addr] = elapsed
	}
	addrs := make([]string, 0, len(merged))
	for addr := range merged {
		addrs = append(addrs, addr)
	}
	sort.Slice(addrs, func(i, j int) bool {
		di, dj := merged[addrs[i]], merged[addrs[j]]
		// 未知耗时（零值）视为最慢，排在已知样本之后；同耗时按地址字典序保证顺序稳定。
		if di == 0 {
			di = time.Hour
		}
		if dj == 0 {
			dj = time.Hour
		}
		if di != dj {
			return di < dj
		}
		return addrs[i] < addrs[j]
	})
	client.lastReachable = addrs
}

// release 将借出的连接归还池内，供后续请求复用；池已关闭或池容量已满（含借出中的
// 连接）时关闭该连接释放资源，由补建循环维持池容量。
func (client *Client) release(pc *pooledConn) {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.inUse--
	if client.closed || len(client.pool)+client.inUse >= client.opt.PoolSize {
		_ = pc.conn.Close()
		return
	}
	client.poolAppendLocked(pc)
}

// discard 关闭并移出池内指定连接；关闭错误忽略。运行期断连由此路径清理，
// 后续由补建循环补齐池容量；借出中的连接（不在池内）同样关闭并回收借出计数。
func (client *Client) discard(pc *pooledConn) {
	client.mu.Lock()
	defer client.mu.Unlock()
	_ = pc.conn.Close()
	found := false
	for i, p := range client.pool {
		if p == pc {
			client.pool = append(client.pool[:i], client.pool[i+1:]...)
			found = true
			break
		}
	}
	if !found && client.inUse > 0 {
		// 借出中的连接被销毁：回收借出计数，容量缺口由补建循环补齐。
		client.inUse--
	}
}

// refillLoop 按固定间隔触发补建；心跳周期随业务活动度在 30s/60s 之间切换，重平衡周期
// 随慢连接状态在 1min/15s 之间切换；收到停止信号（`Disconnect` 关闭 `done`）时退出。
func (client *Client) refillLoop() {
	refillTicker := time.NewTicker(time.Second)
	heartbeatInterval := heartbeatIntervalActive
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	rebalanceInterval := defaultRebalanceInterval
	rebalanceTicker := time.NewTicker(rebalanceInterval)
	defer refillTicker.Stop()
	defer heartbeatTicker.Stop()
	defer rebalanceTicker.Stop()
	// 停止通道在 `Disconnect` 持锁关闭并置空，锁内取一次快照供本循环使用，
	// 避免与 `done` 字段的置空形成数据竞争。
	client.mu.Lock()
	done := client.done
	client.mu.Unlock()
	// 启动即并发补满池余量（建池同步只建 1 条），让高并发场景无需等待每秒补建；
	// 单条失败不阻塞，余量由后续补建循环继续补齐。
	client.refillPoolBulk()
	client.refillPool()
	for {
		select {
		case <-done:
			return
		case <-refillTicker.C:
			client.refillPool()
		case <-heartbeatTicker.C:
			client.heartbeatPool()
			// 心跳后按活动度调整下一轮周期：空闲长驻拉长到 60s，恢复业务访问回到 30s。
			wantHeartbeat := heartbeatIntervalActive
			if client.isIdle() {
				wantHeartbeat = heartbeatIntervalIdle
			}
			if wantHeartbeat != heartbeatInterval {
				heartbeatTicker.Stop()
				heartbeatTicker = time.NewTicker(wantHeartbeat)
				heartbeatInterval = wantHeartbeat
			}
		case <-rebalanceTicker.C:
			client.rebalancePool(false)
			// 重平衡后按慢连接状态调整下一轮周期：有慢节点缩短到 15 秒，恢复健康回到 1 分钟。
			wantInterval := defaultRebalanceInterval
			if client.hasSlowNode() {
				wantInterval = slowRebalanceInterval
			}
			if wantInterval != rebalanceInterval {
				rebalanceTicker.Stop()
				rebalanceTicker = time.NewTicker(wantInterval)
				rebalanceInterval = wantInterval
			}
		}
	}
}

// heartbeatTimeout 为单次应用层心跳的读写超时：僵死连接检测要快，避免占用
// 连接与补建窗口太久，正常连接一次心跳仅一个往返耗时。
const heartbeatTimeout = 2 * time.Second

// 心跳周期按活动度分档：活跃状态保持改动前的 30 秒；低流量长驻超过空闲阈值后拉长到
// 60 秒，减少后台心跳流量与锁争用。近期被业务请求用过的连接本轮直接跳过心跳，业务
// 收发本身已证明连接存活。
const (
	heartbeatIntervalActive = 30 * time.Second
	heartbeatIntervalIdle   = 60 * time.Second
	idleActivityThreshold   = 2 * time.Minute
	heartbeatFreshWindow    = 2 * time.Minute
)

// 重平衡阈值与周期的保守默认值：健康网络下与改动前行为完全一致，仅在池内出现慢节点时
// 放宽切换阈值、缩短重平衡周期，避免慢连接长时间驻留。
const (
	// defaultRebalanceInterval 为健康状态下的重平衡周期，保持改动前的 1 分钟。
	defaultRebalanceInterval = time.Minute
	// slowRebalanceInterval 为识别出慢节点后的缩短周期，让候选测速与迁出更快发生。
	slowRebalanceInterval = 15 * time.Second
	// rebalanceSwitchRatio 为默认切换阈值系数：新节点耗时低于当前节点 30% 以上才切换，
	// 防止网络抖动造成连接频繁漂移。
	rebalanceSwitchRatio = 0.7
	// slowNodeSwitchRatio 为当前节点已被判为慢节点时放宽后的切换阈值系数：新节点略优
	// （低 10%）即迁出，不再等待 30% 的显著优势。
	slowNodeSwitchRatio = 0.9
	// slowNodeRatio 为慢节点判定倍数：当前节点建连耗时达到池内中位数的该倍数即视为
	// 慢节点；样本不足（仅当前节点一条记录）时不判定，保持保守。
	slowNodeRatio = 1.5
	// slowHeartbeatRatio 为心跳慢连接判定倍数：单条心跳耗时达到池内心跳中位数的该倍数
	// 即标记为慢连接；样本不足时不判定。
	slowHeartbeatRatio = 2.0
	// heartbeatSlowMinElapsed 为心跳判慢的绝对下限，避免几十毫秒的正常波动被放大成慢连接。
	heartbeatSlowMinElapsed = 100 * time.Millisecond
)

// fastDialTimeout 为建池快路径单次拨号超时的默认值：行情服务器正常拨号在几十毫秒量级，
// 样本不足（首次建池）时使用该值；样本充足后由 `adaptiveFastDialTimeoutLocked` 按历史
// 建连耗时中位数自适应，全部候选超时失败转慢路径流水线探活兜底。
const fastDialTimeout = 500 * time.Millisecond

// 自适应快路径超时的样本窗口、上下限与余量系数：弱网/慢区域节点的历史耗时偏高时放宽
// 超时，避免被固定 500ms 误杀进慢路径；正常快网络下收紧到下限，保持快路径排除偶发
// 慢样本的能力。
const (
	fastDialMinTimeout  = 200 * time.Millisecond
	fastDialMaxTimeout  = 2 * time.Second
	dialSampleLimit     = 16
	dialSampleMinCount  = 3
	dialTimeoutHeadroom = 1.5
)

// heartbeatPool 对池内空闲连接执行应用层心跳：主站/扩展站发送握手协议请求并以
// 短超时读回完整响应帧，能完整收发即视为连接存活；失败即丢弃该连接，由补建循环补齐。
// MAC 模式无握手协议跳过（由 TCP keepalive 兜底）。连接正被业务请求占用时跳过
// （`TryLock`），避免抢占在途请求。
//
// 方案决策：心跳除判死外顺带识别"慢连接"——样本充足时，单条心跳耗时明显劣于池内心跳
// 中位数即打慢标记并后台触发一次候选测速；慢连接仍然存活不丢弃，迁出决策交给重平衡，
// 避免单次网络抖动直接杀连接。健康节点本轮心跳后清除其慢标记。
func (client *Client) heartbeatPool() {
	client.mu.Lock()
	type heartbeatSnapshot struct {
		pc     *pooledConn
		usedAt time.Time
	}
	snaps := make([]heartbeatSnapshot, 0, len(client.pool))
	poolAddrs := make([]string, 0, len(client.pool))
	for _, pc := range client.pool {
		snaps = append(snaps, heartbeatSnapshot{pc: pc, usedAt: pc.lastUsedAt})
		poolAddrs = append(poolAddrs, pc.addr)
	}
	client.mu.Unlock()

	slowDetected := false
	for _, snap := range snaps {
		pc := snap.pc
		// 心跳只在连接空闲时进行：`TryLock` 失败表示该连接正在执行业务请求。
		if !pc.mu.TryLock() {
			continue
		}
		// 近期被业务请求使用过的连接跳过本轮心跳：业务收发本身已证明连接存活，避免
		// 对活跃连接重复发送握手帧、争抢连接锁；时间戳在池锁内快照，避免与借出路径竞争。
		if !snap.usedAt.IsZero() && time.Since(snap.usedAt) < heartbeatFreshWindow {
			pc.mu.Unlock()
			continue
		}
		started := time.Now()
		err := client.heartbeatConn(pc)
		elapsed := time.Since(started)
		pc.mu.Unlock()
		if err != nil {
			client.discard(pc)
			continue
		}

		client.mu.Lock()
		client.recordHeartbeatLatencyLocked(pc.addr, elapsed)
		median := medianLatencyLocked(client.heartbeatLatency, poolAddrs)
		// 样本不足（中位数为 0）时不判慢，保持改动前只判死的保守行为。
		if median > 0 && elapsed >= heartbeatSlowMinElapsed && elapsed >= time.Duration(slowHeartbeatRatio*float64(median)) {
			if client.slow == nil {
				client.slow = make(map[string]bool)
			}
			client.slow[pc.addr] = true
			slowDetected = true
		} else if client.slow != nil {
			delete(client.slow, pc.addr)
		}
		client.mu.Unlock()
	}
	if slowDetected {
		// 慢连接已识别：后台触发一次候选测速与重平衡，不必等待下一个定时周期。
		go client.rebalancePool(false)
	}
}

// heartbeatConn 在单条连接上执行应用层心跳：发送当前模式握手请求并以 `heartbeatTimeout`
// 超时读回完整响应帧，能完整收发即返回 `nil`，不校验响应内容（验证的是连接存活而非
// 应答正确性）。MAC 模式无握手协议，直接返回 `nil`（由 TCP keepalive 兜底）。
func (client *Client) heartbeatConn(pc *pooledConn) error {
	var obj proto.RequestBuilder
	switch client.mode {
	case clientModeEx:
		obj = proto.NewExLogin()
	case clientModeMacMain, clientModeMacEx:
		return nil
	default:
		obj = proto.NewHello1()
	}
	sendData, err := obj.BuildRequest()
	if err != nil {
		return err
	}
	// 心跳使用独立短超时：僵死连接在 2 秒内即可判定，不必等到业务超时。
	_ = pc.conn.SetDeadline(time.Now().Add(heartbeatTimeout))
	defer func() { _ = pc.conn.SetDeadline(time.Time{}) }()

	if _, err := pc.conn.Write(sendData); err != nil {
		return err
	}
	headerBytes := make([]byte, proto.MessageHeaderBytes)
	if _, err := io.ReadFull(pc.conn, headerBytes); err != nil {
		return err
	}
	headerBuf := bytes.NewReader(headerBytes)
	var header proto.RespHeader
	if err := binary.Read(headerBuf, binary.LittleEndian, &header); err != nil {
		return err
	}
	// 负载长度超出协议上限视为非法数据，防止畸形长度触发过大的内存分配。
	if header.ZipSize > proto.MessageMaxBytes {
		return types.ErrBadData
	}
	msgData := make([]byte, header.ZipSize)
	if _, err := io.ReadFull(pc.conn, msgData); err != nil {
		return err
	}
	// 压缩帧解压后丢弃内容：心跳只验证连接能完整收发，不关心应答语义。
	if header.ZipSize != header.UnZipSize {
		r, err := zlib.NewReader(bytes.NewReader(msgData))
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(io.Discard, r)
		_ = r.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

// buildRefillConnection 为补建流程构建一条连接：稳态优先复用首选地址（保持节点亲和、
// 避免每轮补建重新测速）；首选拨号或握手失败时回退到其他可达候选，避免池内多条连接
// 全部绑定同一故障域。
//
// 方案决策：首选地址可用时不发起任何额外探活，行为与现状一致；只有首选失败才收敛候选
// （最近可达列表优先并剔除首选地址，不足时回退配置地址池），流水线探测首个可用连接直接
// 复用。回退建成后记录测得耗时并刷新可达候选排序；首选与候选全部失败返回错误，由调用方
// 等待下一轮补建循环重试。
func (client *Client) buildRefillConnection(preferred string) (*pooledConn, any, error) {
	if preferred != "" {
		started := time.Now()
		pc, reply, err := client.buildPoolConn(preferred)
		if err == nil {
			client.mu.Lock()
			client.recordAddrLatencyLocked(preferred, time.Since(started))
			client.mu.Unlock()
			return pc, reply, nil
		}
	}

	// 首选不可达：回退候选池其他地址，分散故障域；候选优先取最近可达记录，空则回退配置
	// 地址池，并剔除首选地址避免再次打到同一节点。
	client.mu.Lock()
	candidates := append([]string(nil), client.lastReachable...)
	client.mu.Unlock()
	candidates = removeAddress(candidates, preferred)
	if len(candidates) == 0 {
		candidates = removeAddress(client.addresses(), preferred)
	}
	probeTimeout := min(client.timeout(), slowProbeTimeout)
	started := time.Now()
	pc, reply := client.probeFirstReachable(candidates, probeTimeout)
	if pc == nil {
		return nil, nil, errors.New("no reachable fallback candidate")
	}
	elapsed := time.Since(started)
	client.mu.Lock()
	client.recordAddrLatencyLocked(pc.addr, elapsed)
	client.refreshReachableLocked(map[string]time.Duration{pc.addr: elapsed})
	client.mu.Unlock()
	return pc, reply, nil
}

// refillPoolBulk 启动时并发补齐池余量：稳态与池内首条连接同地址并发拨号
// `PoolSize-当前条数` 条；首选地址失败时由 `buildRefillConnection` 回退其他候选，
// 只入成功者；全部失败不重试，由补建循环后续补齐。
func (client *Client) refillPoolBulk() {
	client.mu.Lock()
	if client.closed || len(client.pool)+client.inUse >= client.opt.PoolSize {
		client.mu.Unlock()
		return
	}
	known := ""
	if len(client.pool) > 0 {
		known = client.pool[0].addr
	}
	need := client.opt.PoolSize - len(client.pool) - client.inUse
	client.mu.Unlock()
	if known == "" || need <= 0 {
		return
	}

	// 网络操作为主，在锁外并发拨号；首选失败自动回退其他候选，只收集成功连接。
	resultsCh := make(chan *pooledConn, need)
	var wg sync.WaitGroup
	for i := 0; i < need; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pc, _, err := client.buildRefillConnection(known)
			if err != nil {
				return
			}
			resultsCh <- pc
		}()
	}
	wg.Wait()
	close(resultsCh)

	client.mu.Lock()
	if client.closed {
		client.mu.Unlock()
		for pc := range resultsCh {
			_ = pc.conn.Close()
		}
		return
	}
	for pc := range resultsCh {
		if len(client.pool)+client.inUse < client.opt.PoolSize {
			client.poolAppendLocked(pc)
		} else {
			// 池已满（补建循环同时入池）时关闭多余连接。
			_ = pc.conn.Close()
		}
	}
	client.mu.Unlock()
}

// rebalancePool 对候选地址重新测速一次，把连接池切换到当前最快的服务器。
//
// 候选收敛为"池内地址 + 最近一次可达地址"，每 5 次（或后台修复强制）触发一次全量探活
// 刷新可达集；并发探活并按可达性、延迟排序，健康节点优先，降权节点兜底。池内连接已
// 全部指向目标地址时不作调整；否则并发预建 `PoolSize` 条指向目标地址的连接，全部失败
// 保持现状，至少建成一条则先入池再关闭旧连接（先建后删，切换期间不出现空窗），未建
// 满的余量由补建循环继续补齐。
//
// 参数 `forceFullProbe` 为 `true` 时跳过候选收敛、直接全量探活配置地址池，用于池内节点
// 连续空数据后的后台修复，尽快迁出数据不全节点；定时 tick 调用传 `false`。重平衡由
// `rebalanceMu` 串行化，定时 tick 与后台修复并发到达时后者直接跳过，避免并发重建连接。
func (client *Client) rebalancePool(forceFullProbe bool) {
	if !client.rebalanceMu.TryLock() {
		return
	}
	defer client.rebalanceMu.Unlock()

	// 探活与拨号均为网络操作，在锁外执行，避免阻塞其他调用方借还连接。
	client.mu.Lock()
	if client.closed {
		client.mu.Unlock()
		return
	}
	client.rebalanceTick++
	fullProbe := forceFullProbe || client.rebalanceTick%5 == 0
	poolAddrs := make([]string, 0, len(client.pool))
	for _, pc := range client.pool {
		duplicated := false
		for _, a := range poolAddrs {
			if a == pc.addr {
				duplicated = true
				break
			}
		}
		if !duplicated {
			poolAddrs = append(poolAddrs, pc.addr)
		}
	}
	// 快照已降权节点：测速排序时健康节点优先，降权节点仅在全部可达节点都被降权时兜底。
	degraded := make(map[string]bool, len(client.health))
	for addr, h := range client.health {
		if h.emptyStreak >= healthDowngradeThreshold {
			degraded[addr] = true
		}
	}
	// 快照心跳标记的慢连接，供切换阈值放宽判定使用。
	slowMarked := make(map[string]bool, len(client.slow))
	for addr, ok := range client.slow {
		slowMarked[addr] = ok
	}
	prevReachable := append([]string(nil), client.lastReachable...)
	client.mu.Unlock()

	// 构建候选集：默认收敛为池内地址与历史可达地址；每 5 次执行一次全量
	// 探活（覆盖配置地址池），以便发现网络变化后新增的可达服务器。
	var candidates []string
	if !fullProbe {
		seen := make(map[string]struct{}, len(poolAddrs)+len(prevReachable))
		candidates = make([]string, 0, len(poolAddrs)+len(prevReachable))
		for _, a := range poolAddrs {
			if _, ok := seen[a]; ok {
				continue
			}
			candidates = append(candidates, a)
			seen[a] = struct{}{}
		}
		for _, a := range prevReachable {
			if _, ok := seen[a]; ok {
				continue
			}
			candidates = append(candidates, a)
			seen[a] = struct{}{}
		}
	} else {
		candidates = client.addresses()
	}
	if len(candidates) == 0 {
		return
	}

	// 测速探活与建池兜底同用短超时：可达节点毫秒级拨号即回，慢样本与失效节点
	// 快速判负，全灭时超时连接驻留时间短，不拥塞同进程其他入口。
	probeTimeout := min(client.timeout(), slowProbeTimeout)
	verified := client.probeReachable(candidates, probeTimeout)
	var fastest string
	var fastestLat time.Duration
	// 健康节点优先选出最快者；全部可达节点均被降权时再回退到任意最快可达节点兜底。
	healthyFastest, healthyLat := "", time.Duration(0)
	latByAddr := make(map[string]time.Duration, len(verified))
	reachable := make([]string, 0, len(verified))
	for _, r := range verified {
		if !r.reachable {
			continue
		}
		latByAddr[r.address] = r.latency
		reachable = append(reachable, r.address)
		if degraded[r.address] {
			continue
		}
		if healthyFastest == "" || r.latency < healthyLat {
			healthyFastest, healthyLat = r.address, r.latency
		}
	}
	fastest, fastestLat = healthyFastest, healthyLat
	if fastest == "" {
		for _, r := range verified {
			if r.reachable {
				fastest, fastestLat = r.address, r.latency
				break
			}
		}
	}
	client.mu.Lock()
	if len(reachable) > 0 {
		client.lastReachable = reachable
		// 同步刷新各地址测得耗时，供快路径后台预比较与补建候选排序使用。
		for _, r := range verified {
			if r.reachable {
				client.recordAddrLatencyLocked(r.address, r.latency)
			}
		}
	}
	curAddr := ""
	if len(client.pool) > 0 {
		curAddr = client.pool[0].addr
	}
	client.mu.Unlock()
	if fastest == "" {
		// 收敛候选全部不可达：保持现有连接不动。
		return
	}

	// 动态切换阈值：当前节点已被心跳标记为慢连接，或其建连耗时达到池内中位数的慢节点
	// 倍数时，放宽到低 10% 即迁出；其余情况维持默认低 30% 的保守阈值，避免网络抖动
	// 造成 `fastest` 频繁漂移、每分钟重建连接（建连风暴）。样本不足（中位数为 0）时不按
	// 分布判慢，保持改动前行为；当前节点已被健康度降权时同样跳过延迟阈值直接迁出。
	switchRatio := rebalanceSwitchRatio
	client.mu.Lock()
	median := medianLatencyLocked(client.addrLatency, poolAddrs)
	curSlow := slowMarked[curAddr]
	client.mu.Unlock()
	if median > 0 {
		if curLat, ok := latByAddr[curAddr]; ok && curLat > 0 && curLat >= time.Duration(slowNodeRatio*float64(median)) {
			curSlow = true
		}
	}
	if curSlow {
		switchRatio = slowNodeSwitchRatio
	}
	if curAddr != "" && !degraded[curAddr] {
		if curLat, ok := latByAddr[curAddr]; ok && curLat > 0 && fastestLat >= time.Duration(float64(curLat)*switchRatio) {
			// 新节点未达到当前阈值：保持现有连接不动，请求继续稳定走当前节点。
			return
		}
	}

	client.mu.Lock()
	if client.closed {
		client.mu.Unlock()
		return
	}
	// 池内已全部指向最快地址时无需切换。
	allFastest := len(client.pool) > 0
	for _, pc := range client.pool {
		if pc.addr != fastest {
			allFastest = false
			break
		}
	}
	if allFastest {
		client.mu.Unlock()
		return
	}
	client.mu.Unlock()

	// 并发预建指向最快地址的连接，单条失败不影响其余，整体耗时由最慢一次拨号决定。
	poolSize := client.opt.PoolSize
	var (
		newMu    sync.Mutex
		newConns = make([]*pooledConn, 0, poolSize)
		wg       sync.WaitGroup
	)
	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pc, _, err := client.buildPoolConn(fastest)
			if err != nil {
				return
			}
			newMu.Lock()
			newConns = append(newConns, pc)
			newMu.Unlock()
		}()
	}
	wg.Wait()
	if len(newConns) == 0 {
		// 最快地址全部拨号失败（瞬时不可达），保持现有连接不动。
		return
	}

	client.mu.Lock()
	if client.closed {
		client.mu.Unlock()
		for _, pc := range newConns {
			_ = pc.conn.Close()
		}
		return
	}
	// 先建后删：新连接入池后再关闭旧连接，切换期间池始终可借出。
	old := client.pool
	client.pool = newConns
	if len(newConns) > 0 {
		client.lastAddr = newConns[0].addr
	}
	// 切换后清空旧地址慢标记：新池地址尚未经历心跳判慢，避免旧标记残留继续放宽阈值。
	client.slow = make(map[string]bool)
	client.mu.Unlock()
	for _, pc := range old {
		_ = pc.conn.Close()
	}
}

// refillPool 补建一条缺失连接：先快速判断是否需要补建，需要时在锁外拨号与
// 握手（网络操作不阻塞借还），成功后加锁入池；池已关闭或容量已满（池内空闲
// 连接与借出中的连接合计达到 `PoolSize`）时关闭多余连接。
func (client *Client) refillPool() {
	client.mu.Lock()
	if client.closed || len(client.pool)+client.inUse >= client.opt.PoolSize {
		client.mu.Unlock()
		return
	}
	// 池内还有可用连接时复用其地址补建，避免每轮重新测速。
	known := ""
	if len(client.pool) > 0 {
		known = client.pool[0].addr
	}
	// 锁内拷贝最近一次可达记录，供池全空时收敛候选使用。
	lastReachable := append([]string(nil), client.lastReachable...)
	client.mu.Unlock()

	// 测速与拨号均为网络操作，在锁外执行，避免阻塞其他调用方借还连接。
	var address string
	if known != "" {
		address = known
	} else {
		// 池完全为空（初始建池失败或全部断连）：优先用最近一次可达地址收敛候选，
		// 避免每轮补建都全量探活；无可达记录时回退配置地址池。流水线探测首个可用
		// 连接并直接复用（零额外建连），兜底探活用更短超时让全灭场景快速判负。
		probeTimeout := min(client.timeout(), slowProbeTimeout)
		candidates := client.addresses()
		if len(lastReachable) > 0 {
			candidates = lastReachable
		}
		pc, _ := client.probeFirstReachable(candidates, probeTimeout)
		if pc == nil {
			return
		}
		client.mu.Lock()
		if !client.closed && len(client.pool)+client.inUse < client.opt.PoolSize {
			client.poolAppendLocked(pc)
		} else {
			_ = pc.conn.Close()
		}
		client.mu.Unlock()
		return
	}
	// 首选地址稳态复用；失败时回退其他可达候选，避免反复绑定同一故障域。首选与回退
	// 候选本轮均不可达时不重试，等待下一轮补建循环。
	pc, _, err := client.buildRefillConnection(address)
	if err != nil {
		return
	}
	client.mu.Lock()
	if !client.closed && len(client.pool)+client.inUse < client.opt.PoolSize {
		client.poolAppendLocked(pc)
	} else {
		_ = pc.conn.Close()
	}
	client.mu.Unlock()
}

// exchange 在指定连接上完成一次请求-响应收发：编码请求、带超时写出发送、按帧读取响应头与负载，
// 压缩帧按需以 `zlib` 解压。
//
// 参数 `conn` 为执行收发的已连接 TCP 连接；参数 `builder` 负责协议请求的字节编码；返回的
// `respHeader` 为解码后的响应头，`payload` 为解压后的明文负载。请求编码失败、写重试超限、帧头或
// 负载读取失败、负载长度超过 `MessageMaxBytes`、解压失败等任一环节出错都会返回错误；连接字节流
// 已无法保证帧边界，调用方（`execute`）须丢弃该连接，避免残留字节破坏后续帧。
func (client *Client) exchange(conn net.Conn, builder proto.RequestBuilder) (respHeader *proto.RespHeader, payload []byte, err error) {
	sendData, err := builder.BuildRequest()
	if err != nil {
		return nil, nil, err
	}

	// 为本次收发设置整体截止时间，结束后清空超时以免影响后续操作。
	_ = conn.SetDeadline(time.Now().Add(client.timeout()))
	defer func() {
		_ = conn.SetDeadline(time.Time{})
	}()

	retryTimes := 0

	for {
		n, err := conn.Write(sendData)
		// 发送不完整时按 `MaxRetryTimes` 重试；超过重试次数或写入出错则放弃本次请求。
		if n < len(sendData) {
			retryTimes++
			if retryTimes <= client.opt.MaxRetryTimes {
				log.Printf("第%d次重试\n", retryTimes)
			} else {
				return nil, nil, err
			}
		} else {
			if err != nil {
				return nil, nil, err
			}
			break
		}
	}

	// 按固定帧长读取响应头，网络不足一帧时 `io.ReadFull` 会等待至凑满或报错。
	headerBytes := make([]byte, proto.MessageHeaderBytes)
	_, err = io.ReadFull(conn, headerBytes)
	if err != nil {
		return nil, nil, err
	}

	// 响应头解码。
	headerBuf := bytes.NewReader(headerBytes)
	var header proto.RespHeader
	if err := binary.Read(headerBuf, binary.LittleEndian, &header); err != nil {
		return nil, nil, err
	}

	// 负载长度超出协议上限视为非法数据，防止畸形长度触发过大的内存分配。
	if header.ZipSize > proto.MessageMaxBytes {
		log.Printf("msgData has bytes(%d) beyond max %d\n", header.ZipSize, proto.MessageMaxBytes)
		return nil, nil, types.ErrBadData
	}

	msgData := make([]byte, header.ZipSize)
	_, err = io.ReadFull(conn, msgData)
	if err != nil {
		return nil, nil, err
	}

	// 压缩大小与解压大小不一致表示负载经过 `zlib` 压缩，否则直接返回明文负载。
	if header.ZipSize != header.UnZipSize {
		var out bytes.Buffer
		b := bytes.NewReader(msgData)
		r, err := zlib.NewReader(b)
		if err != nil {
			return nil, nil, err
		}
		defer r.Close()
		if _, err := io.Copy(&out, r); err != nil {
			return nil, nil, err
		}
		return &header, out.Bytes(), nil
	}

	return &header, msgData, nil
}

// Connect 连接通达信主行情服务器并完成 `Hello1` 握手。
//
// 本方法无业务入参；按候选地址顺序逐个拨号，握手成功后返回主站握手应答 `*proto.Hello1Reply`；全部
// 候选地址拨号或握手失败时返回最后一次错误，应答为 `nil`。扩展站客户端会复用内嵌主站连接完成本次
// 握手。
func (client *Client) Connect() (*proto.Hello1Reply, error) {
	// 扩展站客户端复用内嵌主站连接完成主站握手。
	if client.mode == clientModeEx {
		client.mu.Lock()
		if client.main == nil {
			client.main = newClientWithOptions(client.opt, clientModeMain)
		}
		main := client.main
		client.mu.Unlock()
		return main.Connect()
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closed {
		return nil, errors.New("client closed")
	}
	// 幂等：已有池直接复用，避免重复拨号。
	if len(client.pool) > 0 {
		return nil, nil
	}
	reply, err := client.buildPoolLocked()
	if err != nil {
		return nil, err
	}
	r, _ := reply.(*proto.Hello1Reply)
	return r, nil
}

// buildPoolLocked 在调用方已持池锁的前提下按候选顺序逐级建池并启动补建循环。
//
// 探活升级为"拨号 + 握手验证"：先并发拨号筛选可达地址（探活超时限制 5 秒），再对可达
// 地址并发执行模式握手，握手成功才判定为可用并按总耗时（拨号 + 握手）排序。建池从最快
// 可用地址开始，每个候选并发拨号 `PoolSize` 条，建成至少一条即停止（先建成即降级可用，
// 未建满的余量由补建循环补齐）；当前候选全部失败时按排序逐级回退下一候选。全部候选均
// 失败返回最后一次错误与 `nil`。返回首个成功连接的握手应答；候选为空时返回 `nil` 与错误。
func (client *Client) buildPoolLocked() (any, error) {
	addresses := client.addresses()
	if len(addresses) == 0 {
		return nil, errors.New("no tcp address configured")
	}

	// 快路径：随机抽取并并行拨号 `limit` 个候选（默认 3），收到第一个成功即返回——
	// 波动网络下无需等待首地址超时即可命中可达服务器，随机化避免固定前几个候选
	// 同时落在同一波动窗口时整体判负（原库为串行逐候选拨号，分布窄正因固定单地址；
	// 新库以候选冗余对冲波动，正常网络下各候选毫秒级拨号，随机不损失速度）。
	// 只同步建 1 条返回，余量由补建循环异步补齐；其余候选的并发结果由后台收尾
	// goroutine 收集：记录各候选拨号+握手总耗时并排序，与当前落点预比较，显著更优时
	// 提前切换到最快候选，不必等到 1 分钟重平衡才完成首次收敛。
	limit := 3
	if len(addresses) < limit {
		limit = len(addresses)
	}
	// 快路径拨号超时按历史样本中位数自适应：首次样本不足用默认 500ms，后续慢区域节点
	// 自动放宽、快网络被下限钳制；调用方已持池锁，可直接读取样本窗口。
	fastTimeout := client.adaptiveFastDialTimeoutLocked()
	resultsCh := make(chan fastPathConnRes, limit)
	for _, i := range rand.Perm(len(addresses))[:limit] {
		addr := addresses[i]
		go func() {
			// 快路径拨号用自适应短超时：行情服务器正常拨号在几十毫秒量级，偶发 1 秒级
			// 慢响应是服务器间歇波动，直接排除而非踩中样本（原库固定连稳定服务器，
			// 分布窄正源于此）；全部超时失败转慢路径探活兜底。
			started := time.Now()
			pc, reply, err := client.buildPoolConnWithTimeout(addr, fastTimeout)
			resultsCh <- fastPathConnRes{pc: pc, reply: reply, err: err, elapsed: time.Since(started)}
		}()
	}
	consumed := 0
	for {
		r := <-resultsCh
		consumed++
		if r.err != nil {
			if consumed == limit {
				break // 并行快建全部失败，转慢路径探活回退
			}
			continue
		}
		// 收到第一个成功即返回；剩余并发结果由后台收尾 goroutine 做耗时排序与预比较。
		if remaining := limit - consumed; remaining > 0 {
			go client.collectFastPathResults(resultsCh, remaining, r)
		}
		client.recordAddrLatencyLocked(r.pc.addr, r.elapsed)
		client.poolAppendLocked(r.pc)
		client.startRefillLoop()
		return r.reply, nil
	}

	// 慢路径：并行快建全部失败时流水线探测首个可用连接——拨号成功即握手、首个
	// 握手成功者直接复用（零额外建连），兜底探活用更短超时让全灭场景快速判负并
	// 缩短超时连接驻留，避免大池全灭时并发超时连接拥塞同进程其他入口建池。
	probeTimeout := min(client.timeout(), slowProbeTimeout)
	first, reply := client.probeFirstReachable(addresses, probeTimeout)
	if first == nil {
		return nil, fmt.Errorf("no connection to any candidate")
	}
	client.lastReachable = []string{first.addr}
	client.poolAppendLocked(first)
	client.startRefillLoop()
	return reply, nil
}

// startRefillLoop 启动补建循环（仅一次）。调用方已持池锁，直接启动避免二次加锁；
// 停止信号通道先建好，保证循环可按 `Disconnect` 退出。
func (client *Client) startRefillLoop() {
	if client.done == nil {
		client.done = make(chan struct{})
	}
	client.refill.Do(func() { go client.refillLoop() })
}

// collectFastPathResults 收尾建池快路径剩余并发建连结果：收集成功者、记录拨号+握手
// 总耗时并按耗时排序沉淀可达候选，再与当前落点后台预比较。
//
// 方案决策：同步返回仍由首个成功者决定，不增加 `Connect` 返回延迟；耗时比较与可能的
// 切换全部放后台，避免等待慢候选超时拖慢建连；比较只使用快路径已测得的样本，不额外
// 发起一轮全量探活。切换阈值与重平衡一致（新候选延迟低 30% 以上才动），避免网络抖动
// 导致开局频繁漂移；候选排序只影响内部落点，不改变公开 API 与默认行为。
//
// 参数 `resultsCh` 为快路径结果通道，参数 `remaining` 为尚待收取的结果条数，参数
// `winner` 为已同步入池的首个成功者。未显著更优时保持现状，把余量连接按容量补入池中；
// 显著更优时先建后删切换到最快候选，旧连接与其他地址余量连接关闭回收。
func (client *Client) collectFastPathResults(resultsCh <-chan fastPathConnRes, remaining int, winner fastPathConnRes) {
	collected := make([]fastPathConnRes, 0, remaining)
	for j := 0; j < remaining; j++ {
		rr := <-resultsCh
		if rr.err == nil {
			collected = append(collected, rr)
		}
	}

	client.mu.Lock()
	if client.closed {
		client.mu.Unlock()
		for _, rr := range collected {
			_ = rr.pc.conn.Close()
		}
		return
	}
	// 汇总首个成功者与余量成功者的测得耗时，刷新耗时记录与按耗时升序的可达候选列表。
	measured := map[string]time.Duration{}
	if winner.err == nil {
		measured[winner.pc.addr] = winner.elapsed
	}
	for _, rr := range collected {
		measured[rr.pc.addr] = rr.elapsed
		client.recordAddrLatencyLocked(rr.pc.addr, rr.elapsed)
	}
	client.refreshReachableLocked(measured)
	curAddr := ""
	if len(client.pool) > 0 {
		curAddr = client.pool[0].addr
	}
	curLat := client.addrLatency[curAddr]
	fastest, fastestLat := "", time.Duration(0)
	for addr, elapsed := range measured {
		if fastest == "" || elapsed < fastestLat {
			fastest, fastestLat = addr, elapsed
		}
	}
	client.mu.Unlock()

	// 后台预比较：最快候选与当前落点为同一地址，或新候选未显著更优（未低 30%）时保持
	// 现状；否则提前切换，消除开局抽签后等待 1 分钟重平衡才迁移慢节点的窗口。
	shouldSwitch := fastest != "" && fastest != curAddr && curLat > 0 && fastestLat < curLat*7/10
	if !shouldSwitch {
		client.mu.Lock()
		for _, rr := range collected {
			if !client.closed && len(client.pool)+client.inUse < client.opt.PoolSize {
				client.poolAppendLocked(rr.pc)
			} else {
				_ = rr.pc.conn.Close()
			}
		}
		client.mu.Unlock()
		return
	}

	// 先建后删：并发预建指向最快候选的连接；全部失败保持现状并回收已建成连接，余量
	// 连接按现状补入池中。
	poolSize := client.opt.PoolSize
	var (
		newMu    sync.Mutex
		newConns = make([]*pooledConn, 0, poolSize)
		wg       sync.WaitGroup
	)
	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pc, _, err := client.buildPoolConn(fastest)
			if err != nil {
				return
			}
			newMu.Lock()
			newConns = append(newConns, pc)
			newMu.Unlock()
		}()
	}
	wg.Wait()

	client.mu.Lock()
	if client.closed || len(newConns) == 0 {
		client.mu.Unlock()
		for _, pc := range newConns {
			_ = pc.conn.Close()
		}
		client.mu.Lock()
		for _, rr := range collected {
			if !client.closed && len(client.pool)+client.inUse < client.opt.PoolSize {
				client.poolAppendLocked(rr.pc)
			} else {
				_ = rr.pc.conn.Close()
			}
		}
		client.mu.Unlock()
		return
	}
	old := client.pool
	client.pool = newConns
	client.lastAddr = newConns[0].addr
	client.mu.Unlock()
	for _, pc := range old {
		_ = pc.conn.Close()
	}
	// 切换后余量连接指向其他地址，不再保留，统一关闭回收。
	for _, rr := range collected {
		_ = rr.pc.conn.Close()
	}
}

// probeResult 为一次探活验证的结果：`address` 为 `host:port` 地址，`reachable` 表示
// 拨号与模式握手均成功，`latency` 为总耗时（握手成功时为拨号加握手耗时，否则为拨号耗时）。
type probeResult struct {
	address   string
	reachable bool
	latency   time.Duration
}

// probeFirstReachable 流水线探测首个可用连接：以 `probeConcurrencyWindow` 为并发窗口
// 滚动拨号候选，任一地址拨号成功立即执行当前模式握手，首个握手成功者连同握手应答一起
// 返回并通知其余候选放弃排队拨号——建池兜底与池空补建只求最快可用，不等全部候选完成
// （与全量排序的 `probeReachable` 互补）。全部失败返回 `nil`；提前返回后其余探测
// goroutine 自然结束，多余连接自行关闭，不泄漏。
func (client *Client) probeFirstReachable(addresses []string, timeout time.Duration) (*pooledConn, any) {
	if len(addresses) == 0 {
		return nil, nil
	}
	type connRes struct {
		pc    *pooledConn
		reply any
	}
	resultsCh := make(chan connRes, len(addresses))
	sem := make(chan struct{}, probeConcurrencyWindow)
	stopped := make(chan struct{})
	var stopOnce sync.Once
	stop := func() { stopOnce.Do(func() { close(stopped) }) }
	var wg sync.WaitGroup
	for _, address := range addresses {
		address := address
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 已有成功者抢先返回后，未开始的候选直接放弃，避免排队拨号占用窗口。
			select {
			case <-stopped:
				return
			default:
			}
			sem <- struct{}{}
			defer func() { <-sem }()
			conn, err := net.DialTimeout("tcp", address, timeout)
			if err != nil {
				return
			}
			pc := &pooledConn{conn: conn, addr: address}
			pc.mu.Lock()
			reply, hErr := client.handshakeOn(pc)
			pc.mu.Unlock()
			if hErr != nil {
				_ = conn.Close()
				return
			}
			select {
			case resultsCh <- connRes{pc: pc, reply: reply}:
				stop() // 首个成功者通知其余候选放弃排队
			default:
				// 已有更快的成功者抢先返回，本条连接不再需要。
				_ = conn.Close()
			}
		}()
	}
	doneCh := make(chan struct{})
	go func() { wg.Wait(); close(doneCh) }()
	for {
		select {
		case r := <-resultsCh:
			return r.pc, r.reply
		case <-doneCh:
			return nil, nil
		}
	}
}

// probeReachable 探活验证候选地址：先并发拨号筛选（单次拨号超时 `timeout`），再对
// 拨号可达的地址并发执行当前模式握手。返回结果覆盖所有拨号成功的地址：握手成功者
// 在前（按拨号加握手总耗时升序），仅拨号成功但握手失败者排后（按拨号耗时升序），
// 供建池逐级回退时优先尝试已验证可用的地址。握手协议不存在的模式（MAC）以拨号
// 成功即判定可用。
func (client *Client) probeReachable(addresses []string, timeout time.Duration) []probeResult {
	if len(addresses) == 0 {
		return nil
	}
	// 阶段一：以 `probeConcurrencyWindow` 为窗口并发拨号筛选，保留 TCP 可达的地址
	// 并记录拨号耗时；窗口内滚动调度，避免大池全量并发超时连接拥塞网络栈。
	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		dialed = make([]probeResult, 0, len(addresses))
	)
	sem := make(chan struct{}, probeConcurrencyWindow)
	for _, address := range addresses {
		address := address
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			started := time.Now()
			conn, err := net.DialTimeout("tcp", address, timeout)
			if err != nil {
				return
			}
			_ = conn.Close()
			mu.Lock()
			dialed = append(dialed, probeResult{address: address, latency: time.Since(started)})
			mu.Unlock()
		}()
	}
	wg.Wait()
	if len(dialed) == 0 {
		return nil
	}

	// 阶段二：对拨号可达的地址并发握手验证；握手成功标记可用并计入握手耗时。
	var verified []probeResult
	for _, r := range dialed {
		r := r
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := net.DialTimeout("tcp", r.address, timeout)
			if err != nil {
				return
			}
			started := time.Now()
			pc := &pooledConn{conn: conn, addr: r.address}
			pc.mu.Lock()
			_, hErr := client.handshakeOn(pc)
			pc.mu.Unlock()
			_ = conn.Close()
			if hErr != nil {
				return
			}
			mu.Lock()
			verified = append(verified, probeResult{address: r.address, reachable: true, latency: time.Since(started)})
			mu.Unlock()
		}()
	}
	wg.Wait()

	// 合并：握手成功者在前（总耗时升序），仅拨号成功者排后（拨号耗时升序）。
	merged := make([]probeResult, 0, len(dialed))
	merged = append(merged, verified...)
	for _, r := range dialed {
		duplicated := false
		for _, v := range merged {
			if v.address == r.address {
				duplicated = true
				break
			}
		}
		if !duplicated {
			merged = append(merged, r)
		}
	}
	sort.SliceStable(merged, func(i, j int) bool {
		if merged[i].reachable != merged[j].reachable {
			return merged[i].reachable
		}
		return merged[i].latency < merged[j].latency
	})
	return merged
}

// ConnectEx 连接扩展市场服务器并完成登录握手。
//
// 本方法无业务入参；按候选地址顺序逐个拨号，登录成功后返回扩展站登录应答 `*proto.ExLoginReply`；
// 全部候选地址拨号或登录失败时返回最后一次错误，应答为 `nil`。主站客户端会复用内嵌扩展站连接完成
// 本次登录。
func (client *Client) ConnectEx() (*proto.ExLoginReply, error) {
	// 主站客户端复用内嵌扩展站连接完成扩展站登录。
	if client.mode == clientModeMain {
		client.mu.Lock()
		if client.ex == nil {
			client.ex = newClientWithOptions(client.opt, clientModeEx)
		}
		ex := client.ex
		client.mu.Unlock()
		return ex.ConnectEx()
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closed {
		return nil, errors.New("client closed")
	}
	// 幂等：已有池直接复用，避免重复拨号。
	if len(client.pool) > 0 {
		return nil, nil
	}
	reply, err := client.buildPoolLocked()
	if err != nil {
		return nil, err
	}
	r, _ := reply.(*proto.ExLoginReply)
	return r, nil
}

// Disconnect 关闭连接池与补建循环，并级联关闭关联的内嵌主站、扩展站连接。
//
// 本方法无业务入参；返回关闭过程中遇到的首个错误，后续内嵌连接的关闭错误不会覆盖先出现的错误；
// 已关闭的内嵌连接会跳过重复关闭。关闭后客户端不可再借出连接，需重新 `Connect` 建池。
func (client *Client) Disconnect() error {
	client.mu.Lock()
	client.closed = true
	if client.done != nil {
		close(client.done)
		client.done = nil
	}
	pool := client.pool
	client.pool = nil
	main := client.main
	ex := client.ex
	client.main = nil
	client.ex = nil
	client.mu.Unlock()

	var err error
	for _, pc := range pool {
		if closeErr := pc.conn.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	// 内嵌主站可能就是自身，跳过避免重复关闭；错误不覆盖主连接错误。
	if main != nil && main != client {
		if closeErr := main.Disconnect(); err == nil {
			err = closeErr
		}
	}
	// 扩展站与主站为不同连接时再级联关闭。
	if ex != nil && ex != client && ex != main {
		if closeErr := ex.Disconnect(); err == nil {
			err = closeErr
		}
	}
	return err
}
