// 文件 breaker.go 实现回源熔断器：回源（行情源直连）是缓存层的外部
// 依赖，当它连续失败达到阈值时打开熔断，冷却期内所有回源请求快速失败
// （不再尝试打行情源），防止高频重试把行情源打垮；冷却期满后放行单个
// 试探请求验证恢复，成功则关闭熔断，失败则重新打开。已命中的缓存查询
// 不受熔断影响，只有未命中路径的直连回源受保护。

package cache

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen 表示回源被熔断拒绝：冷却期内未命中的查询快速失败返回
// 该错误，能力层可据此降级展示；熔断恢复后自动回到正常路径。
var ErrCircuitOpen = errors.New("circuit open")

// circuitState 为熔断器状态。
type circuitState int

const (
	circuitClosed   circuitState = iota // 关闭：正常回源并统计结果
	circuitOpen                         // 打开：拒绝一切回源
	circuitHalfOpen                     // 半开：放行单个试探请求验证恢复
)

// circuitBreaker 记录连续回源失败次数，达到阈值时打开熔断，冷却期满后
// 转入半开放行一次试探；试探成功关闭熔断并清零计数，失败重新打开。
// 所有状态转换在互斥锁内完成，并发安全。
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

// before 判断当前是否允许发起一次回源：关闭状态放行；打开状态在冷却
// 期内拒绝、冷却期满转入半开并放行试探；半开状态在试探完成前拒绝
// 其余请求，保证只放行一个试探。
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

// afterSuccess 记录一次回源成功：关闭状态清零连续失败；半开状态的试探
// 成功则关闭熔断并清零计数。
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

// afterFailure 记录一次回源失败：半开状态的试探失败重新打开熔断；关闭
// 状态连续失败达到阈值时打开熔断并记录打开时刻。
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

// fetchGuarded 在熔断器保护下执行一次回源：熔断拒绝时直接返回
// ErrCircuitOpen（不执行 fn）并计入熔断拒绝；否则执行 fn、计入回源次数，
// 并按结果更新熔断计数与回源失败计数。
func (c *CacheClient) fetchGuarded(ctx context.Context, fn func(context.Context) error) error {
	if !c.breaker.before() {
		c.stats.circuitRejects.Add(1)
		return ErrCircuitOpen
	}
	c.stats.fetches.Add(1)
	err := fn(ctx)
	if err != nil {
		c.stats.sourceErrors.Add(1)
		c.breaker.afterFailure()
	} else {
		c.breaker.afterSuccess()
	}
	return err
}
