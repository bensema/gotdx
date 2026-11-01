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

// seedKV 直接把指定语义键写入缓存（模拟此前回源成功写入的状态），供
// 命中路径测试使用。
func seedKV[T any](t *testing.T, c *CacheClient, key string, val T) {
	t.Helper()
	data, err := msgpackEncode(val)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := encodeReplyEnvelope(&replyEnvelope{Version: 1, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.rds.Set(context.Background(), c.kvKey(key), raw, time.Hour).Err(); err != nil {
		t.Fatal(err)
	}
}

// TestGetParsedBlockFileCacheHit 验证缓存命中路径：预写缓存后调用方法，
// 直接返回缓存数据（未连接的行情客户端不会被触碰，否则会报错）。
func TestGetParsedBlockFileCacheHit(t *testing.T) {
	mr := miniredis.RunT(t)
	c := NewCacheClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), gotdx.New())
	defer func() { _ = c.rds.Close() }()

	items := []gotdx.BlockFlatItem{
		{BlockName: "自选股", BlockType: 1, CodeIndex: 0, Code: "000001"},
		{BlockName: "自选股", BlockType: 1, CodeIndex: 1, Code: "600000"},
	}
	seedKV(t, c, "block:parsed:zs.dat", items)

	out, err := c.GetParsedBlockFile(context.Background(), "zs.dat")
	if err != nil {
		t.Fatalf("cache hit should succeed, got %v", err)
	}
	if len(out) != 2 || out[0].Code != "000001" || out[1].Code != "600000" {
		t.Fatalf("unexpected cached items: %+v", out)
	}
}

// TestGetParsedBlockFileMissFails 验证未命中且行情源不可用（未连接）时：
// 首次回源失败返回底层错误并写入失败空标记，二次查询直接命中标记返回
// ErrSourceFailed。
func TestGetParsedBlockFileMissFails(t *testing.T) {
	mr := miniredis.RunT(t)
	c := NewCacheClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), gotdx.New())
	defer func() { _ = c.rds.Close() }()

	_, err := c.GetParsedBlockFile(context.Background(), "zs.dat")
	if err == nil {
		t.Fatal("unconnected source should fail first fetch")
	}
	_, err = c.GetParsedBlockFile(context.Background(), "zs.dat")
	if !errors.Is(err, ErrSourceFailed) {
		t.Fatalf("want ErrSourceFailed from failure marker, got %v", err)
	}
}

// TestGetGroupedBlockFileCacheHitAndKeyIsolation 验证分组解析的缓存命中
// 与操作标签键隔离：预写 grouped 键后命中；同文件名预写 parsed 键不影响
// grouped 查询（两方法同形参数不串键）。
func TestGetGroupedBlockFileCacheHitAndKeyIsolation(t *testing.T) {
	mr := miniredis.RunT(t)
	c := NewCacheClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), gotdx.New())
	defer func() { _ = c.rds.Close() }()

	groups := []gotdx.BlockGroup{
		{BlockName: "自选股", BlockType: 1, StockCount: 2, Codes: []string{"000001", "600000"}},
	}
	seedKV(t, c, "block:grouped:zs.dat", groups)
	seedKV(t, c, "block:parsed:zs.dat", []gotdx.BlockFlatItem{{BlockName: "干扰项"}})

	out, err := c.GetGroupedBlockFile(context.Background(), "zs.dat")
	if err != nil {
		t.Fatalf("cache hit should succeed, got %v", err)
	}
	if len(out) != 1 || out[0].BlockName != "自选股" || len(out[0].Codes) != 2 {
		t.Fatalf("unexpected cached groups: %+v", out)
	}
}
