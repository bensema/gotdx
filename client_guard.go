// 文件 client_guard.go 实现底层 `execute` 入口的熔断与防击穿保护：四个入口
// 的全部行情/数据查询方法最终都汇聚到 `execute`（上层方法只做幂等建连与转包），
// 在此统一挂载无需逐函数改动。
//
// 两层保护：
//   - 熔断（`circuitBreaker`）：按协议命令码分桶，连续失败达到阈值后打开熔断，
//     冷却期内所有请求快速失败（不再尝试打行情源），冷却期满放行单个试探请求
//     验证恢复，成功关闭熔断、失败重新打开；防止行情源故障时高频重试持续打垮源。
//   - 防击穿（`flightGroup`）：同参数并发请求按「屏蔽请求序号后的报文摘要」合并
//     为一次网络请求，其余等待共享结果，防止同一请求风暴重复打行情源。
//
// 与既有机制的关系：空数据换节点（`executePlain`）处理"应答为空"的数据完整度
// 问题，不视为失败、不计入熔断；本层只统计网络收发/协议解析失败。熔断打开期间
// 请求快速失败，重平衡探活与连接补建仍按原节奏运行，源恢复后自动回归。

package gotdx

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
	"sync"
	"time"

	"github.com/bensema/gotdx/proto"
)

// ErrCircuitOpen 表示请求被熔断拒绝：冷却期内 `execute` 直接快速失败返回该错误，
// 不再发起网络请求；熔断恢复后自动回到正常路径。能力层可据此降级展示。
var ErrCircuitOpen = errors.New("circuit open")

// circuitState 为熔断器状态。
type circuitState int

const (
	circuitClosed   circuitState = iota // 关闭：正常放行并统计结果
	circuitOpen                         // 打开：拒绝一切请求
	circuitHalfOpen                     // 半开：放行单个试探请求验证恢复
)

// circuitBreaker 记录连续请求失败次数，达到阈值时打开熔断，冷却期满后转入半开
// 放行一次试探；试探成功关闭熔断并清零计数，失败重新打开。所有状态转换在互斥
// 锁内完成，并发安全。
type circuitBreaker struct {
	mu       sync.Mutex
	state    circuitState
	failures int // 关闭状态下的连续失败次数
	openedAt time.Time

	failThreshold int           // 连续失败多少次打开熔断
	openTimeout   time.Duration // 打开后冷却多久再试探
}

// newCircuitBreaker 按阈值与冷却时长构建熔断器。
func newCircuitBreaker(failThreshold int, openTimeout time.Duration) *circuitBreaker {
	return &circuitBreaker{
		state:         circuitClosed,
		failThreshold: failThreshold,
		openTimeout:   openTimeout,
	}
}

// before 判断当前是否允许发起一次请求：关闭状态放行；打开状态在冷却期内拒绝、
// 冷却期满转入半开并放行试探；半开状态在试探完成前拒绝其余请求，保证只放行
// 一个试探。
func (b *circuitBreaker) before() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case circuitClosed:
		return true
	case circuitOpen:
		if time.Since(b.openedAt) >= b.openTimeout {
			b.state = circuitHalfOpen
			return true
		}
		return false
	case circuitHalfOpen:
		return false
	}
	return false
}

// afterSuccess 记录一次请求成功：关闭状态清零连续失败；半开状态的试探成功则
// 关闭熔断并清零计数。
func (b *circuitBreaker) afterSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case circuitHalfOpen:
		b.state = circuitClosed
		b.failures = 0
	case circuitClosed:
		b.failures = 0
	}
}

// afterFailure 记录一次请求失败：半开状态的试探失败重新打开熔断；关闭状态连续
// 失败达到阈值时打开熔断并记录打开时刻。
func (b *circuitBreaker) afterFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case circuitHalfOpen:
		b.state = circuitOpen
		b.openedAt = time.Now()
		b.failures = 0
	case circuitClosed:
		b.failures++
		if b.failures >= b.failThreshold {
			b.state = circuitOpen
			b.openedAt = time.Now()
			b.failures = 0
		}
	}
}

// guardFor 返回指定协议命令码对应的熔断器，按命令码惰性分桶，互斥保护。
func (client *Client) guardFor(method uint16) *circuitBreaker {
	client.guardMu.Lock()
	defer client.guardMu.Unlock()
	if client.guards == nil {
		client.guards = make(map[uint16]*circuitBreaker)
	}
	br := client.guards[method]
	if br == nil {
		br = newCircuitBreaker(client.opt.GuardFailThreshold, client.opt.GuardOpenTimeout)
		client.guards[method] = br
	}
	return br
}

// flightCall 为一次合并的飞行中请求：所有 follower 等待 `wg` 放行后共享同一份
// 结果。`val`/`err` 由 leader 填充，只读共享。
type flightCall struct {
	wg  sync.WaitGroup
	val any
	err error
}

// flightGroup 是防击穿的 `singleflight` 合并器：同一键的并发请求只执行一次 `fn`，
// 其余请求等待共享结果（失败同样共享，避免同源并发重复打行情源）。键由请求
// 报文摘要生成，同参数请求共享同一键。
type flightGroup struct {
	mu sync.Mutex
	m  map[string]*flightCall
}

// newFlightGroup 构建空的合并器。
func newFlightGroup() *flightGroup {
	return &flightGroup{m: make(map[string]*flightCall)}
}

// Do 按键执行 `fn`：键在途时等待共享其结果，否则由本调用方充当 leader 执行
// `fn` 并广播结果。返回与 `fn` 相同的值或错误。
func (g *flightGroup) Do(key string, fn func() (any, error)) (any, error) {
	g.mu.Lock()
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := &flightCall{}
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	// panic 防御：`fn` 内部 panic 时接住并转为错误。若不接住，`wg.Done` 与
	// `delete` 会随 panic 中断而永不执行——等待者永久阻塞、键泄漏；接住后
	// 执行流必然走到广播与清理，等待者拿到错误正常返回（对齐本库"错误走
	// 返回值"的风格，而非原生 singleflight 的 panic 传播）。
	//
	// 注意同步顺序：`c.err` 等共享字段必须在 `wg.Done()` 之前完成写入，
	// 等待者经 `Wait()` 的 happens-before 边界读取——在 `Done()` 之后写会
	// 与等待者的读竞争（race detector 可复现）。
	var fnErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				fnErr = fmt.Errorf("flight fn panic: %v", r)
				c.err = fnErr
			}
		}()
		c.val, c.err = fn()
	}()
	c.wg.Done()
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
	return c.val, c.err
}

// guardKey 由请求报文生成防击穿合并键：屏蔽请求头请求序号（offset 1..5，主站
// 协议为全局自增 `SeqID`；扩展站/MAC 协议该段固定为零）后取 FNV-1a 64 摘要。
// 屏蔽后同参数请求（各协议对象独立构造、序号不同）共享同一键，不同参数请求
// 键不同。报文不足 6 字节时直接按原字节取摘要。
func guardKey(sendData []byte) string {
	b := make([]byte, len(sendData))
	copy(b, sendData)
	if len(b) >= 6 {
		binary.LittleEndian.PutUint32(b[1:5], 0)
	}
	h := fnv.New64a()
	_, _ = h.Write(b)
	return strconv.FormatUint(h.Sum64(), 16)
}

// executeGuarded 是 `execute` 的受保护路径：先构建请求报文取命令码与合并键，
// 熔断拒绝时快速失败；放行后经 `flightGroup` 合并并发，leader 执行真实请求并
// 按结果更新熔断计数。报文构建失败或报文过短无法取命令码时直连 `executePlain`
// 兜底，保证协议异常不阻塞调用。
func (client *Client) executeGuarded[T any](protocol proto.Protocol[T]) (T, error) {
	var zero T
	sendData, err := protocol.BuildRequest()
	if err != nil || len(sendData) < 12 {
		return client.executePlain(protocol)
	}
	method := binary.LittleEndian.Uint16(sendData[10:12])
	if !client.guardFor(method).before() {
		return zero, ErrCircuitOpen
	}
	key := guardKey(sendData)
	v, err := client.flights.Do(key, func() (any, error) {
		reply, e := client.executePlain(protocol)
		br := client.guardFor(method)
		if e != nil {
			br.afterFailure()
		} else {
			br.afterSuccess()
		}
		return reply, e
	})
	if err != nil {
		return zero, err
	}
	return v.(T), nil
}
