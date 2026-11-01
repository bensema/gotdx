package gotdx

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/bensema/gotdx/proto"
)

// TestSubscriberPushesEveryChange 验证假数据源每次快照变化都推送、取消后停止：
// `Fetch` 返回递增计数，默认深比较判定每次变化，推送次数应等于轮询次数。
func TestSubscriberPushesEveryChange(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(1))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	var seed atomic.Int64
	var pushes atomic.Int64
	cancel, err := c.Subscriber().Subscribe(
		SubscribeSource[int64]{
			Fetch: func() (int64, error) { return seed.Add(1), nil },
		},
		100*time.Millisecond,
		func(v int64) bool {
			pushes.Add(1)
			return true
		})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	// 基线 + 至少两次轮询推送。
	deadline := time.Now().Add(2 * time.Second)
	for pushes.Load() < 3 {
		if time.Now().After(deadline) {
			t.Fatalf("expected at least 3 pushes, got %d", pushes.Load())
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	time.Sleep(300 * time.Millisecond)
	after := pushes.Load()
	time.Sleep(200 * time.Millisecond)
	if got := pushes.Load(); got != after {
		t.Fatalf("pushes must stop after cancel: before=%d after=%d", after, got)
	}
}

// TestSubscriberSkipsUnchanged 验证快照不变时不推送：固定值数据源只推送基线一次。
func TestSubscriberSkipsUnchanged(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(1))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	var pushes atomic.Int64
	cancel, err := c.Subscriber().Subscribe(
		SubscribeSource[string]{
			Fetch: func() (string, error) { return "fixed", nil },
		},
		100*time.Millisecond,
		func(v string) bool {
			pushes.Add(1)
			return true
		})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	defer cancel()

	deadline := time.Now().Add(2 * time.Second)
	for pushes.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("baseline push never happened")
		}
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(350 * time.Millisecond) // 覆盖至少 3 个轮询周期
	if got := pushes.Load(); got != 1 {
		t.Fatalf("expected only baseline push, got %d", got)
	}
}

// TestSubscriberCustomChanged 验证自定义变化判定生效：变化幅度不足阈值时不推送。
func TestSubscriberCustomChanged(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(1))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	var seed atomic.Int64 // 每次 +10，但阈值 50：每 5 次拉取算一次"变化"
	var pushes atomic.Int64
	cancel, err := c.Subscriber().Subscribe(
		SubscribeSource[int64]{
			Fetch: func() (int64, error) { return seed.Add(10), nil },
			Changed: func(old, new int64) bool {
				return new-old >= 50
			},
		},
		50*time.Millisecond,
		func(v int64) bool {
			pushes.Add(1)
			return true
		})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	defer cancel()

	// 基线一次；随后每 5 次拉取（250ms）推一次，500ms 内应只多推约 2 次。
	deadline := time.Now().Add(2 * time.Second)
	for pushes.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("baseline push never happened")
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond)
	got := pushes.Load()
	if got < 2 || got > 4 {
		t.Fatalf("expected ~3 pushes (baseline + 2 threshold hits) in 500ms, got %d", got)
	}
}

// TestSubscriberQuotesViaFactory 验证工厂可直接订阅真实报价数据源（复用
// `GetSecurityQuotes` 拉取链路与 fake 服务器的固定报价响应）。
func TestSubscriberQuotesViaFactory(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(1))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	var pushes atomic.Int64
	cancel, err := c.Subscriber().Subscribe(
		SubscribeSource[*proto.GetSecurityQuotesReply]{
			Fetch: func() (*proto.GetSecurityQuotesReply, error) {
				return c.GetSecurityQuotes([]uint8{0}, []string{"000000"})
			},
		},
		200*time.Millisecond,
		func(q *proto.GetSecurityQuotesReply) bool {
			pushes.Add(1)
			if q.Count != 1 || len(q.List) != 1 {
				t.Fatalf("unexpected quotes reply: count=%d list=%d", q.Count, len(q.List))
			}
			return true
		})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	defer cancel()

	deadline := time.Now().Add(2 * time.Second)
	for pushes.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("baseline push never happened")
		}
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(400 * time.Millisecond)
	if got := pushes.Load(); got != 1 {
		t.Fatalf("fixed quotes must push once (baseline), got %d", got)
	}
}

// TestSubscriberCoalescesSameKey 验证相同 `Key` 的订阅合并到同一轮询任务：两个
// 处理函数共享一次拉取，拉取次数等于轮询次数而非订阅数。
func TestSubscriberCoalescesSameKey(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(1))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	var fetches atomic.Int64
	var pushesA, pushesB atomic.Int64
	sub := c.Subscriber()

	_, errA := sub.Subscribe(
		SubscribeSource[int64]{
			Key:   "quotes",
			Fetch: func() (int64, error) { return fetches.Add(1), nil },
		},
		80*time.Millisecond,
		func(v int64) bool { pushesA.Add(1); return true })
	_, errB := sub.Subscribe(
		SubscribeSource[int64]{
			Key:   "quotes",
			Fetch: func() (int64, error) { return fetches.Add(1), nil },
		},
		80*time.Millisecond,
		func(v int64) bool { pushesB.Add(1); return true })
	if errA != nil || errB != nil {
		t.Fatalf("subscribe failed: a=%v b=%v", errA, errB)
	}

	// 等待两个处理函数都收到推送（合并任务同一轮同时服务两者）。
	deadline := time.Now().Add(2 * time.Second)
	for pushesA.Load() == 0 || pushesB.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("coalesced baseline push never happened")
		}
		time.Sleep(20 * time.Millisecond)
	}
	// 继续观察若干轮：每次拉取都变化，两个处理函数都应收到推送。
	deadline = time.Now().Add(600 * time.Millisecond)
	for fetches.Load() < 8 {
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	pushes := pushesA.Load()
	if pushes != pushesB.Load() {
		t.Fatalf("coalesced handlers must receive same push count: a=%d b=%d", pushes, pushesB.Load())
	}
	if pushes < 8 {
		t.Fatalf("expected at least 8 pushes per handler, got %d", pushes)
	}
	// 拉取次数应约为推送次数（一次拉取服务两个处理函数），而非两倍。
	if fetches.Load() > pushes+2 {
		t.Fatalf("fetch count %d far exceeds push count %d (coalescing broken)", fetches.Load(), pushes)
	}
}

// TestSubscriberPanicIsolated 验证拉取函数与处理函数 panic 均被隔离：panic 不
// 崩溃进程，处理函数 panic 后订阅继续接收后续推送。
func TestSubscriberPanicIsolated(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(1))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	var pulls atomic.Int64
	var pushes atomic.Int64
	sub := c.Subscriber()

	cancel, err := sub.Subscribe(
		SubscribeSource[int64]{
			Fetch: func() (int64, error) {
				if pulls.Add(1)%2 == 0 {
					panic("fetch boom") // 每偶数次拉取 panic
				}
				return pulls.Load(), nil
			},
		},
		50*time.Millisecond,
		func(v int64) bool {
			pushes.Add(1)
			if pushes.Load() == 1 {
				panic("handler boom") // 首次推送 panic
			}
			return true
		})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	defer cancel()

	// 处理函数 panic 后订阅仍存活并持续收到后续推送。
	deadline := time.Now().Add(2 * time.Second)
	for pushes.Load() < 3 {
		if time.Now().After(deadline) {
			t.Fatalf("subscription must survive panics, pushes=%d", pushes.Load())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestSubscriberSameKeyDifferentType 验证同 `Key` 不同快照类型返回错误。
func TestSubscriberSameKeyDifferentType(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(1))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	sub := c.Subscriber()
	_, errA := sub.Subscribe(SubscribeSource[int64]{Key: "k", Fetch: func() (int64, error) { return 1, nil }}, 0, func(int64) bool { return true })
	if errA != nil {
		t.Fatalf("first subscribe failed: %v", errA)
	}
	_, errB := sub.Subscribe(SubscribeSource[string]{Key: "k", Fetch: func() (string, error) { return "x", nil }}, 0, func(string) bool { return true })
	if errB == nil {
		t.Fatal("expected type conflict error for same key")
	}
}

// TestSubscriberCloseStopsAll 验证 `Close` 停止工厂内全部订阅，之后不再推送。
func TestSubscriberCloseStopsAll(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(1))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	var pushes atomic.Int64
	sub := c.Subscriber()
	cancel, err := sub.Subscribe(
		SubscribeSource[int64]{
			Fetch: func() (int64, error) { return time.Now().UnixNano(), nil },
		},
		50*time.Millisecond,
		func(v int64) bool { pushes.Add(1); return true })
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	defer cancel()

	deadline := time.Now().Add(2 * time.Second)
	for pushes.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("baseline push never happened")
		}
		time.Sleep(20 * time.Millisecond)
	}
	sub.Close()
	time.Sleep(300 * time.Millisecond)
	after := pushes.Load()
	time.Sleep(200 * time.Millisecond)
	if got := pushes.Load(); got != after {
		t.Fatalf("pushes must stop after close: before=%d after=%d", after, got)
	}
}

// TestSubscriberJoiningHandlerGetsBaseline 验证合并追加的订阅（同 `Key` 第二订阅）
// 即使快照无变化也能立即收到当前快照建立基线，而不是等到数据变化。
func TestSubscriberJoiningHandlerGetsBaseline(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(1))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	var firstPushes, secondPushes atomic.Int64
	sub := c.Subscriber()

	// 第一个订阅：固定值数据源，只有基线一次推送。
	_, errA := sub.Subscribe(
		SubscribeSource[int64]{Key: "k", Fetch: func() (int64, error) { return 42, nil }},
		100*time.Millisecond,
		func(v int64) bool { firstPushes.Add(1); return true })
	if errA != nil {
		t.Fatalf("first subscribe failed: %v", errA)
	}
	deadline := time.Now().Add(2 * time.Second)
	for firstPushes.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("first baseline never happened")
		}
		time.Sleep(20 * time.Millisecond)
	}

	// 第二个订阅（同 Key 追加）：数据仍无变化，但应立即收到当前快照。
	_, errB := sub.Subscribe(
		SubscribeSource[int64]{Key: "k", Fetch: func() (int64, error) { return 42, nil }},
		100*time.Millisecond,
		func(v int64) bool { secondPushes.Add(1); return true })
	if errB != nil {
		t.Fatalf("second subscribe failed: %v", errB)
	}
	deadline = time.Now().Add(2 * time.Second)
	for secondPushes.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("joining handler must get baseline even when unchanged")
		}
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(300 * time.Millisecond)
	if got := secondPushes.Load(); got != 1 {
		t.Fatalf("unchanged source must push joining handler once, got %d", got)
	}
}

// TestSubscriberHandlerFalseCancels 验证处理函数返回 `false` 后该订阅取消、不再
// 推送，同任务其他处理函数不受影响。
func TestSubscriberHandlerFalseCancels(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(1))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	var seed atomic.Int64
	var cancellerPushes, keeperPushes atomic.Int64
	sub := c.Subscriber()

	_, errA := sub.Subscribe(
		SubscribeSource[int64]{
			Key:   "k",
			Fetch: func() (int64, error) { return seed.Add(1), nil },
		},
		60*time.Millisecond,
		func(v int64) bool { cancellerPushes.Add(1); return false }) // 首次即取消
	if errA != nil {
		t.Fatalf("canceller subscribe failed: %v", errA)
	}
	_, errB := sub.Subscribe(
		SubscribeSource[int64]{
			Key:   "k",
			Fetch: func() (int64, error) { return seed.Add(1), nil },
		},
		60*time.Millisecond,
		func(v int64) bool { keeperPushes.Add(1); return true })
	if errB != nil {
		t.Fatalf("keeper subscribe failed: %v", errB)
	}

	deadline := time.Now().Add(2 * time.Second)
	for keeperPushes.Load() < 4 {
		if time.Now().After(deadline) {
			t.Fatalf("keeper pushes insufficient: %d", keeperPushes.Load())
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got := cancellerPushes.Load(); got != 1 {
		t.Fatalf("cancelled handler must not receive more pushes, got %d", got)
	}
}

// TestSubscriberDisconnectStops 验证客户端 `Disconnect` 后工厂内全部订阅停止推送。
func TestSubscriberDisconnectStops(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(1))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	var pushes atomic.Int64
	sub := c.Subscriber()
	_, err := sub.Subscribe(
		SubscribeSource[int64]{
			Fetch: func() (int64, error) { return time.Now().UnixNano(), nil },
		},
		50*time.Millisecond,
		func(v int64) bool { pushes.Add(1); return true })
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for pushes.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("baseline push never happened")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := c.Disconnect(); err != nil {
		t.Fatalf("disconnect failed: %v", err)
	}
	time.Sleep(300 * time.Millisecond)
	after := pushes.Load()
	time.Sleep(200 * time.Millisecond)
	if got := pushes.Load(); got != after {
		t.Fatalf("pushes must stop after disconnect: before=%d after=%d", after, got)
	}
}

// TestSubscriberCancelledHandlersCompacted 验证取消的订阅从任务处理函数列表中
// 剔除：高频订阅-取消后列表长度收敛，不随取消次数无限增长。
func TestSubscriberCancelledHandlersCompacted(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(1))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	var pushes atomic.Int64
	sub := c.Subscriber()

	// 大量订阅-取消（同 Key 独立 handler），保留一个活跃订阅。
	kept, err := sub.Subscribe(
		SubscribeSource[int64]{
			Key:   "k",
			Fetch: func() (int64, error) { return time.Now().UnixNano(), nil },
		},
		30*time.Millisecond,
		func(v int64) bool { pushes.Add(1); return true })
	if err != nil {
		t.Fatalf("keeper subscribe failed: %v", err)
	}
	defer kept()

	for i := 0; i < 50; i++ {
		cancel, err := sub.Subscribe(
			SubscribeSource[int64]{
				Key:   "k",
				Fetch: func() (int64, error) { return time.Now().UnixNano(), nil },
			},
			30*time.Millisecond,
			func(v int64) bool { return true })
		if err != nil {
			t.Fatalf("transient subscribe %d failed: %v", i, err)
		}
		cancel()
	}

	// 等待活跃订阅持续推送，确认任务存活且压缩已生效。
	deadline := time.Now().Add(2 * time.Second)
	for pushes.Load() < 5 {
		if time.Now().After(deadline) {
			t.Fatalf("keeper must keep receiving pushes: %d", pushes.Load())
		}
		time.Sleep(20 * time.Millisecond)
	}

	// 压缩后任务处理函数列表应只保留活跃订阅（加少量轮询窗口内的残留）。
	sub.mu.Lock()
	var task *subTask
	for _, tt := range sub.tasks {
		if tt.key == "k" {
			task = tt
			break
		}
	}
	sub.mu.Unlock()
	if task == nil {
		t.Fatal("task for key k not found")
	}
	sub.mu.Lock()
	n := len(task.handlers)
	sub.mu.Unlock()
	if n > 3 {
		t.Fatalf("handlers must be compacted after cancels, got %d", n)
	}
}
