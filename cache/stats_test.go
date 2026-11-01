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

// TestStatsHitAndEmptyHits 验证缓存命中与空标记命中分别计数。
func TestStatsHitAndEmptyHits(t *testing.T) {
	mr := miniredis.RunT(t)
	c := NewCacheClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), gotdx.New())
	defer func() { _ = c.rds.Close() }()
	ctx := context.Background()

	// 预写缓存 → 命中。
	seedKV(t, c, "block:parsed:zs.dat", []gotdx.BlockFlatItem{{Code: "000001"}})
	if _, err := c.GetParsedBlockFile(ctx, "zs.dat"); err != nil {
		t.Fatal(err)
	}
	// 空结果标记 → 空标记命中。
	if err := c.rds.Set(ctx, c.kvEmptyKey("block:parsed:empty.dat"), emptyFlagOK, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetParsedBlockFile(ctx, "empty.dat"); err != nil {
		t.Fatal(err)
	}
	s := c.Stats()
	if s.Hits != 1 {
		t.Fatalf("want 1 hit, got %d", s.Hits)
	}
	if s.EmptyHits != 1 {
		t.Fatalf("want 1 empty hit, got %d", s.EmptyHits)
	}
	if s.Fetches != 0 {
		t.Fatalf("want 0 fetches on cache paths, got %d", s.Fetches)
	}
}

// TestStatsFetchFailure 验证回源失败同时计入回源次数与失败次数。
func TestStatsFetchFailure(t *testing.T) {
	mr := miniredis.RunT(t)
	c := NewCacheClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), gotdx.New())
	defer func() { _ = c.rds.Close() }()

	_, _ = c.GetParsedBlockFile(context.Background(), "a.dat") // 未连接 → 回源失败
	s := c.Stats()
	if s.Fetches != 1 {
		t.Fatalf("want 1 fetch, got %d", s.Fetches)
	}
	if s.SourceErrors != 1 {
		t.Fatalf("want 1 source error, got %d", s.SourceErrors)
	}
}

// TestStatsCircuitRejects 验证熔断拒绝计入熔断计数且不执行回源。
func TestStatsCircuitRejects(t *testing.T) {
	c := &CacheClient{breaker: newCircuitBreaker(1, time.Hour)}
	ctx := context.Background()
	_ = c.fetchGuarded(ctx, func(ctx context.Context) error { return errors.New("down") }) // 打开熔断
	_ = c.fetchGuarded(ctx, func(ctx context.Context) error { return nil })                // 被拒绝
	if c.stats.fetches.Load() != 1 {
		t.Fatalf("want 1 fetch (reject must not fetch), got %d", c.stats.fetches.Load())
	}
	if c.stats.sourceErrors.Load() != 1 {
		t.Fatalf("want 1 source error, got %d", c.stats.sourceErrors.Load())
	}
	if c.stats.circuitRejects.Load() != 1 {
		t.Fatalf("want 1 circuit reject, got %d", c.stats.circuitRejects.Load())
	}
}
