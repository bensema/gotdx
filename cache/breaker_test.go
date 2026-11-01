package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestCircuitBreakerOpenOnFailures 验证连续失败达到阈值后打开熔断并拒绝回源。
func TestCircuitBreakerOpenOnFailures(t *testing.T) {
	b := newCircuitBreaker(3, time.Minute)
	for i := 0; i < 3; i++ {
		if !b.before() {
			t.Fatalf("closed state should allow, round %d", i+1)
		}
		b.afterFailure()
	}
	if b.before() {
		t.Fatal("open state should reject")
	}
}

// TestCircuitBreakerRecoversAfterTimeout 验证冷却期满后放行试探、成功后
// 关闭熔断并清零计数。
func TestCircuitBreakerRecoversAfterTimeout(t *testing.T) {
	b := newCircuitBreaker(2, 20*time.Millisecond)
	for i := 0; i < 2; i++ {
		b.before()
		b.afterFailure()
	}
	if b.before() {
		t.Fatal("open state should reject before timeout")
	}
	time.Sleep(30 * time.Millisecond)
	if !b.before() {
		t.Fatal("half-open should allow the probe")
	}
	b.afterSuccess()
	if b.state != circuitClosed {
		t.Fatalf("probe success should close circuit, got %v", b.state)
	}
	if !b.before() {
		t.Fatal("closed state should allow again")
	}
}

// TestCircuitBreakerReopensOnProbeFailure 验证试探失败后重新打开熔断。
func TestCircuitBreakerReopensOnProbeFailure(t *testing.T) {
	b := newCircuitBreaker(2, 10*time.Millisecond)
	for i := 0; i < 2; i++ {
		b.before()
		b.afterFailure()
	}
	time.Sleep(20 * time.Millisecond)
	if !b.before() {
		t.Fatal("half-open should allow the probe")
	}
	b.afterFailure()
	if b.state != circuitOpen {
		t.Fatalf("probe failure should reopen circuit, got %v", b.state)
	}
	if b.before() {
		t.Fatal("reopened circuit should reject")
	}
}

// TestCircuitBreakerHalfOpenSingleProbe 验证半开期间只放行一个试探请求，
// 其余被拒，防止试探期并发打爆行情源。
func TestCircuitBreakerHalfOpenSingleProbe(t *testing.T) {
	b := newCircuitBreaker(1, 10*time.Millisecond)
	b.before()
	b.afterFailure()
	time.Sleep(20 * time.Millisecond)
	if !b.before() {
		t.Fatal("first request should become the probe")
	}
	if b.before() {
		t.Fatal("second request in half-open should be rejected")
	}
	b.afterSuccess()
}

// TestFetchGuardedSkipsSourceWhenOpen 验证熔断打开时 fetchGuarded 不执行
// 回调而直接返回 ErrCircuitOpen。
func TestFetchGuardedSkipsSourceWhenOpen(t *testing.T) {
	c := &CacheClient{breaker: newCircuitBreaker(2, time.Minute)}
	for i := 0; i < 2; i++ {
		_ = c.fetchGuarded(context.Background(), func(ctx context.Context) error {
			return errors.New("source down")
		})
	}
	called := false
	err := c.fetchGuarded(context.Background(), func(ctx context.Context) error {
		called = true
		return nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("want ErrCircuitOpen, got %v", err)
	}
	if called {
		t.Fatal("source should not be called while open")
	}
}

// TestFetchGuardedRecovery 验证熔断冷却后试探请求被放行，成功后熔断关闭。
func TestFetchGuardedRecovery(t *testing.T) {
	c := &CacheClient{breaker: newCircuitBreaker(2, 20*time.Millisecond)}
	for i := 0; i < 2; i++ {
		_ = c.fetchGuarded(context.Background(), func(ctx context.Context) error {
			return errors.New("source down")
		})
	}
	if err := c.fetchGuarded(context.Background(), func(ctx context.Context) error {
		return errors.New("source down")
	}); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("open state should fast-fail, got %v", err)
	}
	time.Sleep(30 * time.Millisecond)
	if err := c.fetchGuarded(context.Background(), func(ctx context.Context) error {
		return nil
	}); err != nil {
		t.Fatalf("probe should be allowed and succeed, got %v", err)
	}
	if err := c.fetchGuarded(context.Background(), func(ctx context.Context) error {
		return nil
	}); err != nil {
		t.Fatalf("closed state should allow again, got %v", err)
	}
}
