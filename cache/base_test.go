package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/bensema/gotdx"
	"github.com/redis/go-redis/v9"
)

// TestNewCacheClientNormalizesOptions 验证非法选项值回退默认：阈值 0 不
// 会让熔断永久打开、TTL 0 不会立即过期、空前缀不会产生畸形键。
func TestNewCacheClientNormalizesOptions(t *testing.T) {
	rds := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"}) // 不连接，仅构造
	defer func() { _ = rds.Close() }()
	c := NewCacheClient(rds, gotdx.New(),
		WithKeyPrefix(""),
		WithEmptyTTL(0),
		WithLockTimeout(-time.Second),
		WithLiveTTL(0),
		WithCircuitFailThreshold(0),
		WithCircuitOpenTimeout(-5*time.Second),
	)
	if c.opt.KeyPrefix != DefaultKeyPrefix {
		t.Fatalf("empty prefix should fall back, got %q", c.opt.KeyPrefix)
	}
	if c.opt.EmptyTTL != DefaultEmptyTTL || c.opt.LockTimeout != DefaultLockTimeout ||
		c.opt.LiveTTL != DefaultLiveTTL {
		t.Fatalf("non-positive ttl should fall back: %+v", c.opt)
	}
	if c.opt.CircuitFailThreshold != DefaultCircuitFailThreshold {
		t.Fatalf("zero threshold should fall back, got %d", c.opt.CircuitFailThreshold)
	}
	if c.opt.CircuitOpenTimeout != DefaultCircuitOpenTimeout {
		t.Fatalf("negative timeout should fall back, got %v", c.opt.CircuitOpenTimeout)
	}
}

// TestTsQueryRefreshRespectsCircuit 验证实时槽刷新走熔断保护：熔断打开时
// 即使刷新条件满足也不调用 latest 回调打行情源，返回已有缓存数据。
func TestTsQueryRefreshRespectsCircuit(t *testing.T) {
	mr := miniredis.RunT(t)
	c := NewCacheClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), gotdx.New(),
		WithCircuitFailThreshold(1),
		WithCircuitOpenTimeout(time.Hour))
	defer func() { _ = c.rds.Close() }()

	// 一次失败即打开熔断，冷却 1 小时保持打开。
	_ = c.fetchGuarded(context.Background(), func(ctx context.Context) error {
		return errors.New("source down")
	})

	zkey := "k:000001:min"
	lk := c.liveKey(zkey)
	// 实时槽更新时刻已过期 → 触发刷新条件。
	mr.Set(lk, encodeLiveValue(100, time.Now().Add(-time.Hour).UnixNano(), []byte("old")))

	latestCalled := false
	_, _ = c.tsQuery[int](context.Background(), zkey, 0, 10, time.Second,
		func(ctx context.Context) (int, float64, error) {
			latestCalled = true
			return 0, 0, nil
		})
	if latestCalled {
		t.Fatal("latest should not be called while circuit is open")
	}
}

// TestTsGetEmptyMarkerSkipsSource 验证空结果标记命中时不回源、返回空数据。
func TestTsGetEmptyMarkerSkipsSource(t *testing.T) {
	mr := miniredis.RunT(t)
	c := NewCacheClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), gotdx.New())
	defer func() { _ = c.rds.Close() }()

	zkey := "k:000001:min"
	if err := c.tsMarkEmpty(context.Background(), zkey, false); err != nil {
		t.Fatal(err)
	}
	fetched := false
	out, err := c.tsGet[int](context.Background(), zkey, 0, 10, true, true, 0, nil,
		func(ctx context.Context) ([]int, []float64, bool, error) {
			fetched = true
			return nil, nil, false, nil
		})
	if err != nil {
		t.Fatalf("empty marker should return no error, got %v", err)
	}
	if fetched {
		t.Fatal("should not fetch source when empty marker hit")
	}
	if len(out) != 0 {
		t.Fatalf("empty marker should return empty slice, got %d items", len(out))
	}
}

// TestTsGetFailureMarkerPropagates 验证失败空标记命中时返回 ErrSourceFailed
// 且不回源。
func TestTsGetFailureMarkerPropagates(t *testing.T) {
	mr := miniredis.RunT(t)
	c := NewCacheClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), gotdx.New())
	defer func() { _ = c.rds.Close() }()

	zkey := "k:000001:min"
	if err := c.tsMarkEmpty(context.Background(), zkey, true); err != nil {
		t.Fatal(err)
	}
	fetched := false
	_, err := c.tsGet[int](context.Background(), zkey, 0, 10, true, true, 0, nil,
		func(ctx context.Context) ([]int, []float64, bool, error) {
			fetched = true
			return nil, nil, false, nil
		})
	if !errors.Is(err, ErrSourceFailed) {
		t.Fatalf("want ErrSourceFailed, got %v", err)
	}
	if fetched {
		t.Fatal("should not fetch source when failure marker hit")
	}
}
