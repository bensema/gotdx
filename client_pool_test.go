package gotdx

import (
	"sync"
	"testing"
	"time"
)

// TestPoolBuildsExpectedConns 验证 `Connect` 按 `PoolSize` 预建指定条数的连接，
// 且各连接实际地址一致（同一候选地址）。
func TestPoolBuildsExpectedConns(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(3))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	c.mu.Lock()
	got := len(c.pool)
	c.mu.Unlock()
	// 建池同步只建 1 条，余量由补建循环每秒补齐；轮询等待池补满。
	deadline := time.Now().Add(4 * time.Second)
	for got < 3 && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		c.mu.Lock()
		got = len(c.pool)
		c.mu.Unlock()
	}
	if got != 3 {
		t.Fatalf("want pool of 3, got %d", got)
	}
	if c.CurrentAddress() != addr {
		t.Fatalf("want current address %s, got %s", addr, c.CurrentAddress())
	}
}

// TestPoolRoundRobin 验证借出连接按轮询分散：连续两次借出应命中不同连接。
func TestPoolRoundRobin(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(2))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	// 建池同步只建 1 条，等待补建循环补满后轮询语义才可验证。
	deadline := time.Now().Add(4 * time.Second)
	for {
		c.mu.Lock()
		filled := len(c.pool)
		c.mu.Unlock()
		if filled == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("pool was not refilled in time")
		}
		time.Sleep(100 * time.Millisecond)
	}

	pc1, err := c.acquire()
	if err != nil {
		t.Fatal(err)
	}
	pc2, err := c.acquire()
	if err != nil {
		t.Fatal(err)
	}
	if pc1 == pc2 {
		t.Fatal("round-robin should hand out different conns")
	}
}

// TestPoolConcurrentExecutes 验证高并发请求分散到池内多条连接且全部成功
// （单连接串行、连接间并行，帧边界不被破坏）。
func TestPoolConcurrentExecutes(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(3))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	const workers = 20
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.StockHistoryTickChart(20260811, 0, "000779"); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent execute failed: %v", err)
	}
}

// TestPoolDiscardAndRefill 验证坏连接被丢弃后由补建循环补齐池容量。
func TestPoolDiscardAndRefill(t *testing.T) {
	addr, stop := fakeTDXServer(t, 59202)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(1))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	// 超长响应触发坏数据错误，唯一连接被丢弃，池清空。
	if _, err := c.StockHistoryTickChart(20260811, 0, "000779"); err == nil {
		t.Fatal("expected ErrBadData")
	}
	c.mu.Lock()
	empty := len(c.pool) == 0
	c.mu.Unlock()
	if !empty {
		t.Fatal("pool must be empty after discarding the only conn")
	}

	// 等待补建循环（间隔 1 秒）把池补回。
	deadline := time.Now().Add(5 * time.Second)
	for {
		c.mu.Lock()
		filled := len(c.pool)
		c.mu.Unlock()
		if filled == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("pool was not refilled in time")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// TestConnectIdempotent 验证重复 `Connect` 复用已有池，不重复拨号。
func TestConnectIdempotent(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(2))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	if _, err := c.Connect(); err != nil {
		t.Fatalf("second connect should be idempotent: %v", err)
	}
	c.mu.Lock()
	got := len(c.pool)
	c.mu.Unlock()
	// 建池同步只建 1 条，余量由补建循环每秒补齐；轮询等待池补满。
	deadline := time.Now().Add(4 * time.Second)
	for got < 2 && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		c.mu.Lock()
		got = len(c.pool)
		c.mu.Unlock()
	}
	if got != 2 {
		t.Fatalf("pool size must stay 2, got %d", got)
	}
}

// TestDisconnectClosesPool 验证 `Disconnect` 清空池且后续借出报错。
func TestDisconnectClosesPool(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(2))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	if err := c.Disconnect(); err != nil {
		t.Fatalf("disconnect failed: %v", err)
	}
	c.mu.Lock()
	empty := len(c.pool) == 0
	c.mu.Unlock()
	if !empty {
		t.Fatal("pool must be empty after disconnect")
	}
	if _, err := c.acquire(); err == nil {
		t.Fatal("acquire must fail after disconnect")
	}
	// 关闭后再查询直接报无可用连接（与单连接模型"connection is nil"等价）。
	if _, err := c.StockHistoryTickChart(20260811, 0, "000779"); err == nil {
		t.Fatal("execute must fail after disconnect")
	}
}

// TestPoolSizeNormalized 验证非正数 `PoolSize` 回退默认值。
func TestPoolSizeNormalized(t *testing.T) {
	c := New(WithPoolSize(0))
	if c.opt.PoolSize <= 0 {
		t.Fatalf("pool size must be normalized to default, got %d", c.opt.PoolSize)
	}
}

// TestAcquireBeforeConnect 验证未连接时借出连接直接报错。
func TestAcquireBeforeConnect(t *testing.T) {
	c := New(WithPoolSize(1))
	if _, err := c.acquire(); err == nil {
		t.Fatal("acquire must fail before connect")
	}
}

// TestRebalanceSwitchesToFastest 验证重平衡把连接池切换到当前最快的服务器：
// 原最快地址不可达后，池内连接全部替换为指向可达地址的新连接。
func TestRebalanceSwitchesToFastest(t *testing.T) {
	aAddr, aStop := fakeTDXServer(t, 0)
	defer aStop()
	bAddr, bStop := fakeTDXServer(t, 0)
	defer bStop()

	c := New(WithTCPAddress(aAddr), WithTCPAddressPool(bAddr), WithTimeoutSec(5), WithPoolSize(2))
	// 手动构造"已连接旧地址 A"的池，并预置最近一次探活的可达记录（含 A 与 B），
	// 供重平衡收敛候选使用。
	pcA, _, err := c.buildPoolConn(aAddr)
	if err != nil {
		t.Fatalf("build conn to A failed: %v", err)
	}
	c.mu.Lock()
	c.pool = []*pooledConn{pcA}
	c.lastReachable = []string{aAddr, bAddr}
	c.mu.Unlock()

	// 关闭 A 模拟服务器不可达，触发重平衡。
	aStop()
	c.rebalancePool(false)

	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.pool) != 2 {
		t.Fatalf("want pool of 2 after rebalance, got %d", len(c.pool))
	}
	for _, pc := range c.pool {
		if pc.addr != bAddr {
			t.Fatalf("pool must point to fastest %s, got %s", bAddr, pc.addr)
		}
	}
}

// TestRebalanceKeepsFastestPool 验证池内连接已全部指向最快地址时，重平衡不做任何切换。
func TestRebalanceKeepsFastestPool(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTCPAddressPool(), WithTimeoutSec(5), WithPoolSize(2))
	pc1, _, err := c.buildPoolConn(addr)
	if err != nil {
		t.Fatalf("build conn 1 failed: %v", err)
	}
	pc2, _, err := c.buildPoolConn(addr)
	if err != nil {
		t.Fatalf("build conn 2 failed: %v", err)
	}
	c.mu.Lock()
	c.pool = []*pooledConn{pc1, pc2}
	c.mu.Unlock()

	c.rebalancePool(false)

	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.pool) != 2 {
		t.Fatalf("pool size must stay 2, got %d", len(c.pool))
	}
	for _, pc := range c.pool {
		if pc.addr != addr {
			t.Fatalf("pool must stay on %s, got %s", addr, pc.addr)
		}
	}
}

// TestHeartbeatKeepsHealthyConn 验证正常连接的应用层心跳通过，连接保留在池内。
func TestHeartbeatKeepsHealthyConn(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(2))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	// 建池同步只建 1 条，等待补建循环补满后心跳才对整池生效。
	deadline := time.Now().Add(4 * time.Second)
	for {
		c.mu.Lock()
		filled := len(c.pool)
		c.mu.Unlock()
		if filled == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("pool was not refilled in time")
		}
		time.Sleep(100 * time.Millisecond)
	}

	c.heartbeatPool()

	c.mu.Lock()
	got := len(c.pool)
	c.mu.Unlock()
	if got != 2 {
		t.Fatalf("healthy conns must survive heartbeat, pool got %d", got)
	}
}

// TestHeartbeatDiscardsStaleConn 验证应用层心跳会丢弃僵死连接：连接 TCP 被对端
// 关闭后心跳失败，该连接从池内移除。
func TestHeartbeatDiscardsStaleConn(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(2))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	// 模拟僵死连接：关闭池内第一条连接的底层 TCP（应用层对此不知情）。
	c.mu.Lock()
	pc := c.pool[0]
	c.mu.Unlock()
	_ = pc.conn.Close()

	// 心跳应发现该连接不可读写并丢弃。
	c.heartbeatPool()

	c.mu.Lock()
	stillIn := false
	for _, p := range c.pool {
		if p == pc {
			stillIn = true
			break
		}
	}
	c.mu.Unlock()
	if stillIn {
		t.Fatal("stale conn still in pool after heartbeat")
	}
}

// TestPoolAcquireRemovesReleaseRestores 验证借还语义：借出即移出池（池内减少、
// 借出计数增加），归还后回池（池内恢复、借出计数归零），容量（池内+借出）守恒。
// 该语义保证在途连接不参与心跳巡检与重平衡切换，避免切换误关在途连接。
func TestPoolAcquireRemovesReleaseRestores(t *testing.T) {
	addr, stop := fakeTDXServer(t, 0)
	defer stop()

	c := New(WithTCPAddress(addr), WithTimeoutSec(5), WithTCPAddressPool(), WithPoolSize(3))
	if _, err := c.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer c.Disconnect()

	// 等待补建循环补满池容量。
	deadline := time.Now().Add(4 * time.Second)
	for {
		c.mu.Lock()
		filled := len(c.pool)
		c.mu.Unlock()
		if filled == 3 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("pool was not refilled in time")
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 借出两条：池内剩 1、借出计数 2，容量合计保持 3。
	pc1, err := c.acquire()
	if err != nil {
		t.Fatal(err)
	}
	pc2, err := c.acquire()
	if err != nil {
		t.Fatal(err)
	}
	c.mu.Lock()
	if len(c.pool) != 1 || c.inUse != 2 {
		t.Fatalf("after acquire: pool=%d inUse=%d, want pool=1 inUse=2", len(c.pool), c.inUse)
	}
	c.mu.Unlock()

	// 归还两条：池内恢复 3、借出计数归零。
	c.release(pc1)
	c.release(pc2)
	c.mu.Lock()
	if len(c.pool) != 3 || c.inUse != 0 {
		t.Fatalf("after release: pool=%d inUse=%d, want pool=3 inUse=0", len(c.pool), c.inUse)
	}
	c.mu.Unlock()
}

// TestSingleConnectionModeConfig 验证 `WithSingleConnectionMode` 同时把池大小置为 1 并关闭
// 空数据换节点；默认构造保持三连接池与换节点开启。
func TestSingleConnectionModeConfig(t *testing.T) {
	c := New(WithSingleConnectionMode())
	if c.opt.PoolSize != 1 {
		t.Fatalf("single connection mode must set PoolSize=1, got %d", c.opt.PoolSize)
	}
	if !c.opt.DisableEmptyNodeSwitch {
		t.Fatal("single connection mode must disable empty node switch")
	}

	def := New()
	if def.opt.PoolSize != _defaultPoolSize {
		t.Fatalf("default pool size must be %d, got %d", _defaultPoolSize, def.opt.PoolSize)
	}
	if def.opt.DisableEmptyNodeSwitch {
		t.Fatal("empty node switch must be enabled by default")
	}
}

// TestEmptyNodeSwitchFlagIndependent 验证换节点开关可独立于池大小控制：单独关闭换节点时
// 池大小保持传入值，显式传 `false` 恢复默认换节点行为。
func TestEmptyNodeSwitchFlagIndependent(t *testing.T) {
	multi := New(WithPoolSize(3), WithEmptyNodeSwitchDisabled(true))
	if multi.opt.PoolSize != 3 || !multi.opt.DisableEmptyNodeSwitch {
		t.Fatalf("independent flag must keep pool size 3 and disable switch, got pool=%d disabled=%v",
			multi.opt.PoolSize, multi.opt.DisableEmptyNodeSwitch)
	}

	enabled := New(WithPoolSize(2), WithEmptyNodeSwitchDisabled(false))
	if enabled.opt.PoolSize != 2 || enabled.opt.DisableEmptyNodeSwitch {
		t.Fatalf("explicit enabled flag must keep switch on, got pool=%d disabled=%v",
			enabled.opt.PoolSize, enabled.opt.DisableEmptyNodeSwitch)
	}
}
