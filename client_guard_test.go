// 文件 client_guard_test.go 覆盖底层 `execute` 熔断与防击穿保护：熔断状态机
// 全路径、防击穿合并、合并键确定性与真实执行入口的熔断打开行为。

package gotdx

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bensema/gotdx/proto"
)

// TestCircuitBreakerStateMachine 覆盖熔断器三态机全路径：关闭→连续失败打开→
// 冷却期内拒绝→冷却期满半开放行单个试探→试探成功关闭；试探失败重新打开。
func TestCircuitBreakerStateMachine(t *testing.T) {
	br := newCircuitBreaker(2, 50*time.Millisecond)

	// 关闭状态放行。
	if !br.before() {
		t.Fatal("closed state should allow")
	}
	br.afterFailure() // 失败 1
	if !br.before() {
		t.Fatal("closed state still allows after 1 failure")
	}
	br.afterFailure() // 失败 2 → 打开
	if br.before() {
		t.Fatal("open state should reject")
	}
	// 冷却期内持续拒绝。
	if br.before() {
		t.Fatal("open state still rejects within cooldown")
	}
	// 冷却期满 → 半开放行单个试探，其余拒绝。
	time.Sleep(60 * time.Millisecond)
	if !br.before() {
		t.Fatal("half-open probe should be allowed after cooldown")
	}
	if br.before() {
		t.Fatal("half-open should reject concurrent probes")
	}
	// 试探成功 → 关闭并清零。
	br.afterSuccess()
	if !br.before() {
		t.Fatal("should be closed after successful probe")
	}
}

// TestCircuitBreakerHalfOpenFailureReopens 覆盖半开试探失败重新打开。
func TestCircuitBreakerHalfOpenFailureReopens(t *testing.T) {
	br := newCircuitBreaker(1, 30*time.Millisecond)
	br.afterFailure() // 立即打开
	time.Sleep(40 * time.Millisecond)
	if !br.before() {
		t.Fatal("probe should be allowed after cooldown")
	}
	br.afterFailure() // 试探失败 → 重新打开
	if br.before() {
		t.Fatal("should be open again after failed probe")
	}
}

// TestCircuitBreakerSuccessResetsFailures 覆盖关闭状态成功清零连续失败。
func TestCircuitBreakerSuccessResetsFailures(t *testing.T) {
	br := newCircuitBreaker(2, time.Minute)
	br.afterFailure()
	br.afterFailure() // → 打开
	if br.before() {
		t.Fatal("should be open after 2 consecutive failures")
	}
	// 冷却期满后放行试探，试探成功关闭并清零。
	br.openedAt = time.Now().Add(-time.Hour)
	if !br.before() {
		t.Fatal("probe should be allowed after cooldown")
	}
	br.afterSuccess()
	// 清零后重新从零计数：再来 2 次失败才打开。
	br.afterFailure()
	if !br.before() {
		t.Fatal("should still be closed after 1 failure following reset")
	}
	br.afterFailure()
	if br.before() {
		t.Fatal("should open after 2 consecutive failures following reset")
	}
}

// TestFlightGroupMerge 覆盖防击穿合并：20 个并发同键请求只执行一次 `fn`，
// 全部拿到同一份结果；失败同样共享。
func TestFlightGroupMerge(t *testing.T) {
	g := newFlightGroup()
	var calls atomic.Int32
	want := "shared-result"
	var wg sync.WaitGroup
	errs := make([]error, 20)
	vals := make([]any, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v, err := g.Do("k", func() (any, error) {
				calls.Add(1)
				time.Sleep(20 * time.Millisecond)
				return want, nil
			})
			vals[i], errs[i] = v, err
		}(i)
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("fn executed %d times, want 1", calls.Load())
	}
	for i := 0; i < 20; i++ {
		if errs[i] != nil || vals[i] != want {
			t.Fatalf("result %d = %v, %v", i, vals[i], errs[i])
		}
	}

	// 失败共享：并发同键全部拿到同一错误。
	var wg2 sync.WaitGroup
	boom := errors.New("boom")
	var gotErr atomic.Value
	for i := 0; i < 5; i++ {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			_, err := g.Do("kfail", func() (any, error) { return nil, boom })
			gotErr.Store(err)
		}()
	}
	wg2.Wait()
	if !errors.Is(gotErr.Load().(error), boom) {
		t.Fatalf("shared failure = %v, want boom", gotErr.Load())
	}
}

// TestFlightGroupPanicBroadcast 覆盖 panic 防御：`fn` 内部 panic 时接住转为
// 错误广播给所有等待者（无死锁），且键清理保证后续同键请求可正常执行。
func TestFlightGroupPanicBroadcast(t *testing.T) {
	g := newFlightGroup()
	start := make(chan struct{})
	const waiters = 8
	errs := make([]error, waiters)
	var wg sync.WaitGroup
	for i := 0; i < waiters; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, err := g.Do("k", func() (any, error) {
				panic("boom")
			})
			errs[i] = err
		}(i)
	}
	close(start)
	wg.Wait() // 若等待者死锁，此处永不返回，测试超时暴露
	for i, err := range errs {
		if err == nil {
			t.Fatalf("waiter %d got nil error, want panic-derived error", i)
		}
	}
	// 键必须已清理：后续同键请求应作为新 leader 正常执行。
	done := make(chan struct{})
	go func() {
		v, err := g.Do("k", func() (any, error) { return 42, nil })
		if err != nil || v != 42 {
			t.Errorf("post-panic request: v=%v err=%v, want 42/nil", v, err)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("key not cleaned: post-panic same-key request blocked")
	}
}

// TestFlightGroupSeparateKeys 覆盖不同键互不合并、各自执行：所有请求用启动
// 栅栏同时进入 `Do`，两个键各合并一次。
func TestFlightGroupSeparateKeys(t *testing.T) {
	g := newFlightGroup()
	var calls atomic.Int32
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			key := string(rune('a' + i%2)) // 两个键
			_, _ = g.Do(key, func() (any, error) {
				calls.Add(1)
				time.Sleep(20 * time.Millisecond)
				return i, nil
			})
		}(i)
	}
	close(start)
	wg.Wait()
	if calls.Load() != 2 {
		t.Fatalf("fn executed %d times, want 2 (two distinct keys)", calls.Load())
	}
}

// TestGuardKeyDeterministic 覆盖合并键确定性：同参数两独立协议对象（请求序号
// 不同）生成相同键；不同参数生成不同键；屏蔽序号后主站与扩展站键形态一致。
func TestGuardKeyDeterministic(t *testing.T) {
	ka := proto.NewGetSecurityBars(&proto.GetSecurityBarsRequest{
		Market: 0, Code: [6]byte{'0', '0', '0', '0', '0', '1'}, Start: 100, Count: 10,
	})
	kb := proto.NewGetSecurityBars(&proto.GetSecurityBarsRequest{
		Market: 0, Code: [6]byte{'0', '0', '0', '0', '0', '1'}, Start: 100, Count: 10,
	})
	kc := proto.NewGetSecurityBars(&proto.GetSecurityBarsRequest{
		Market: 0, Code: [6]byte{'0', '0', '0', '0', '0', '1'}, Start: 101, Count: 10,
	})
	ra, err := ka.BuildRequest()
	if err != nil {
		t.Fatal(err)
	}
	rb, err := kb.BuildRequest()
	if err != nil {
		t.Fatal(err)
	}
	rc, err := kc.BuildRequest()
	if err != nil {
		t.Fatal(err)
	}
	if guardKey(ra) != guardKey(rb) {
		t.Fatalf("same-params keys differ: %q vs %q", guardKey(ra), guardKey(rb))
	}
	if guardKey(ra) == guardKey(rc) {
		t.Fatal("different-params keys equal")
	}

	// 扩展站请求字节完全确定，两对象同键。
	ea := proto.NewExGetKLine(&proto.ExGetKLineRequest{
		Category: 1, Code: [9]byte{'A', 'A', 'P', 'L'}, Count: 10,
	})
	eb := proto.NewExGetKLine(&proto.ExGetKLineRequest{
		Category: 1, Code: [9]byte{'A', 'A', 'P', 'L'}, Count: 10,
	})
	xra, _ := ea.BuildRequest()
	xrb, _ := eb.BuildRequest()
	if guardKey(xra) != guardKey(xrb) {
		t.Fatal("same ex-params keys differ")
	}
}

// TestExecuteGuardOpens 覆盖真实执行入口熔断打开：指向不可达地址的客户端连续
// 请求失败达到阈值后返回 `ErrCircuitOpen`，不再发起网络请求；关闭保护时保持
// 原始失败行为（不熔断）。
func TestExecuteGuardOpens(t *testing.T) {
	client := New(WithTCPAddress("127.0.0.1:1"), WithTimeoutSec(1), WithPoolSize(1))
	client.opt.GuardFailThreshold = 3
	client.opt.GuardOpenTimeout = time.Minute // 打开后不再自动恢复，稳定观察拒绝态

	var gotOpen bool
	for i := 0; i < 8; i++ {
		_, err := client.GetSecurityBars(0, 0, "000001", 0, 10)
		if errors.Is(err, ErrCircuitOpen) {
			gotOpen = true
			break
		}
		if err == nil {
			t.Fatal("unreachable address should fail")
		}
	}
	if !gotOpen {
		t.Fatal("expected ErrCircuitOpen after repeated failures")
	}
	client.Disconnect()
}

// TestExecuteGuardDisabled 覆盖关闭保护后不熔断：同一失败请求持续返回原始错误。
func TestExecuteGuardDisabled(t *testing.T) {
	client := New(WithTCPAddress("127.0.0.1:1"), WithTimeoutSec(1), WithPoolSize(1), WithGuardDisabled(true))
	for i := 0; i < 8; i++ {
		_, err := client.GetSecurityBars(0, 0, "000001", 0, 10)
		if errors.Is(err, ErrCircuitOpen) {
			t.Fatal("guard disabled should never report circuit open")
		}
		if err == nil {
			t.Fatal("unreachable address should fail")
		}
	}
	client.Disconnect()
}
