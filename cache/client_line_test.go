// 文件 client_line_test.go 覆盖 K 线缓存的离线用例：命中窗口位置与
// 返回顺序、键隔离、空标记、同分覆盖、跨分流转幂等与深度不足回源；
// 真实行情用例由 `GOTDX_INTEGRATION=1` 环境变量控制。

package cache

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/proto"
	"github.com/redis/go-redis/v9"
	"github.com/vmihailenco/msgpack/v5"
)

// newTestCache 构造基于内存 `Redis` 的缓存实例，行情源指向不可达地址，
// 保证离线用例不会真的请求行情服务器。
func newTestCache(t *testing.T) *CacheClient {
	t.Helper()
	mr := miniredis.RunT(t)
	cli := gotdx.New(
		gotdx.WithTCPAddress("127.0.0.1:1"),
		gotdx.WithTCPAddressPool("127.0.0.1:1"),
		gotdx.WithTimeoutSec(1),
	)
	c := NewCacheClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), cli)
	t.Cleanup(func() { _ = c.rds.Close() })
	return c
}

// testBars 构造从 09:31 起每分钟一根的 K 线序列（从旧到新），收盘价按
// 序号递增便于断言窗口位置。
func testBars(n int) []proto.SecurityBar {
	base := time.Date(2026, 11, 6, 9, 31, 0, 0, time.Local)
	bars := make([]proto.SecurityBar, n)
	for i := range bars {
		t := base.Add(time.Duration(i) * time.Minute)
		bars[i] = proto.SecurityBar{
			Open:     float64(i + 1),
			Close:    float64(i + 1),
			High:     float64(i + 1),
			Low:      float64(i + 1),
			DateTime: t,
		}
	}
	return bars
}

// assertAscending 断言返回序列按时间从早到晚严格递增。
func assertAscending(tb testing.TB, bars []proto.SecurityBar) {
	tb.Helper()
	for i := 1; i < len(bars); i++ {
		if !bars[i-1].DateTime.Before(bars[i].DateTime) {
			tb.Fatalf("bars not ascending at %d: %s >= %s",
				i, bars[i-1].DateTime, bars[i].DateTime)
		}
	}
}

// TestStockKLineCacheHitWindow 验证命中路径：start=0 返回最新一根在末尾、
// 深窗口按 offset 正确定位，且返回顺序从早到晚、不触碰行情源。
func TestStockKLineCacheHitWindow(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	zkey := klineCacheKey(8, 1, "000001", 1, 0)
	bars := testBars(200)
	if err := c.klineStore(ctx, zkey, bars); err != nil {
		t.Fatalf("store kline: %v", err)
	}

	cases := []struct {
		name       string
		start      uint16
		count      uint16
		firstClose float64 // 窗口最旧一根的收盘价
		lastClose  float64 // 窗口最新一根的收盘价
	}{
		{"start=0 取整窗", 0, 200, 1, 200},
		{"start=0 取最新 20 根", 0, 20, 181, 200},
		{"start=10 count=20", 10, 20, 171, 190},
		{"start=199 count=1", 199, 1, 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := c.Stats()
			got, err := c.StockKLine(8, 1, "000001", tc.start, tc.count, 1, 0)
			if err != nil {
				t.Fatalf("StockKLine: %v", err)
			}
			if len(got) != int(tc.count) {
				t.Fatalf("len = %d, want %d", len(got), tc.count)
			}
			if got[0].Close != tc.firstClose || got[len(got)-1].Close != tc.lastClose {
				t.Fatalf("window = [%v, %v], want [%v, %v]",
					got[0].Close, got[len(got)-1].Close, tc.firstClose, tc.lastClose)
			}
			assertAscending(t, got)
			after := c.Stats()
			if after.Hits != before.Hits+1 || after.Fetches != before.Fetches {
				t.Fatalf("stats hits/fetches = %d/%d, want %d/%d",
					after.Hits, after.Fetches, before.Hits+1, before.Fetches)
			}
		})
	}
}

// TestStockKLineCacheKeyIsolation 验证周期倍数与复权方式各自独立成键，
// 相同证券的不同参数不会串用缓存；times 为零按 1 归一。
func TestStockKLineCacheKeyIsolation(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	first := testBars(20)
	if err := c.klineStore(ctx, klineCacheKey(8, 1, "000001", 1, 0), first); err != nil {
		t.Fatalf("store times=1: %v", err)
	}
	second := testBars(10)
	for i := range second {
		second[i].Close += 100
	}
	if err := c.klineStore(ctx, klineCacheKey(8, 1, "000001", 5, 0), second); err != nil {
		t.Fatalf("store times=5: %v", err)
	}

	got, err := c.StockKLine(8, 1, "000001", 0, 10, 1, 0)
	if err != nil || len(got) != 10 || got[len(got)-1].Close != 20 {
		t.Fatalf("times=1 window = %v (err %v), want last 20", len(got), err)
	}
	got, err = c.StockKLine(8, 1, "000001", 0, 10, 5, 0)
	if err != nil || len(got) != 10 || got[len(got)-1].Close != 110 {
		t.Fatalf("times=5 window = %v (err %v), want last 110", len(got), err)
	}
	// times 为零与为 1 归一为同一个键。
	got, err = c.StockKLine(8, 1, "000001", 0, 10, 0, 0)
	if err != nil || len(got) != 10 || got[len(got)-1].Close != 20 {
		t.Fatalf("times=0 window = %v (err %v), want same as times=1", len(got), err)
	}
}

// TestStockKLineEmptyMarker 验证空标记命中：确认无数据返回空切片与空
// 错误，回源失败标记返回 ErrSourceFailed，两者都不再回源。
func TestStockKLineEmptyMarker(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	zkey := klineCacheKey(8, 1, "000001", 1, 0)
	// 先写入一根并建槽，使实时槽处于新鲜状态，避免刷新路径触发回源。
	if err := c.klineStore(ctx, zkey, testBars(1)); err != nil {
		t.Fatalf("store kline: %v", err)
	}

	// 直接通过实例的 Redis 写入空标记。
	if err := c.rds.Set(ctx, c.tsEmptyKey(zkey), emptyFlagOK, 0).Err(); err != nil {
		t.Fatalf("set empty flag: %v", err)
	}
	got, err := c.StockKLine(8, 1, "000001", 0, 200, 1, 0)
	if err != nil || got != nil {
		t.Fatalf("empty flag should return nil,nil, got %v err %v", got, err)
	}

	if err := c.rds.Set(ctx, c.tsEmptyKey(zkey), emptyFlagFail, 0).Err(); err != nil {
		t.Fatalf("set fail flag: %v", err)
	}
	_, err = c.StockKLine(8, 1, "000001", 0, 200, 1, 0)
	if !errors.Is(err, ErrSourceFailed) {
		t.Fatalf("fail flag should return ErrSourceFailed, got %v", err)
	}
}

// TestStockKLineCountZero 验证 count 为零时不触网、返回空切片无错误。
func TestStockKLineCountZero(t *testing.T) {
	c := newTestCache(t)
	got, err := c.StockKLine(8, 1, "000001", 0, 0, 1, 0)
	if err != nil || len(got) != 0 {
		t.Fatalf("count=0 should return empty, got %v err %v", got, err)
	}
}

// TestKlineLiveSameScoreOverwrite 验证同分覆盖：同一根 K 线在盘中演化时
// 每次刷新都能更新槽内数值，而不是停在该分钟的首笔快照。
func TestKlineLiveSameScoreOverwrite(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	zkey := klineCacheKey(8, 1, "000001", 1, 0)

	bar := proto.SecurityBar{Close: 10, High: 12, Low: 9,
		DateTime: time.Date(2026, 11, 6, 10, 0, 0, 0, time.Local)}
	if _, err := c.klineUpsertLive(ctx, zkey, bar); err != nil {
		t.Fatalf("upsert first: %v", err)
	}
	bar.Close, bar.High = 15, 16 // 同一分钟，数值继续演化
	if ok, err := c.klineUpsertLive(ctx, zkey, bar); err != nil || !ok {
		t.Fatalf("upsert same score should overwrite, ok=%v err=%v", ok, err)
	}
	raw, err := c.rds.Get(ctx, c.liveKey(zkey)).Result()
	if err != nil {
		t.Fatalf("read live slot: %v", err)
	}
	lv, err := decodeLiveValue(raw)
	if err != nil {
		t.Fatalf("decode live slot: %v", err)
	}
	cur, err := msgpackDecode[proto.SecurityBar](lv.Data)
	if err != nil {
		t.Fatalf("decode bar: %v", err)
	}
	if cur.Close != 15 || cur.High != 16 {
		t.Fatalf("live slot not overwritten: %+v", cur)
	}
}

// TestKlineLiveRollForwardIdempotent 验证分数推进时旧槽根幂等流转进
// `ZSet`：重复写同一根不会堆积重复成员。
func TestKlineLiveRollForwardIdempotent(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	zkey := klineCacheKey(8, 1, "000001", 1, 0)
	base := time.Date(2026, 11, 6, 10, 0, 0, 0, time.Local)

	first := proto.SecurityBar{Close: 10, DateTime: base}
	second := proto.SecurityBar{Close: 11, DateTime: base.Add(time.Minute)}
	if _, err := c.klineUpsertLive(ctx, zkey, first); err != nil {
		t.Fatalf("upsert first: %v", err)
	}
	if _, err := c.klineUpsertLive(ctx, zkey, second); err != nil {
		t.Fatalf("upsert second: %v", err)
	}
	card, err := c.rds.ZCard(ctx, c.tsKeyBase(zkey)).Result()
	if err != nil || card != 1 {
		t.Fatalf("zset card = %d (err %v), want 1", card, err)
	}
	// 重复写入同一根：同分覆盖，基数不变。
	if _, err := c.klineUpsertLive(ctx, zkey, second); err != nil {
		t.Fatalf("upsert second again: %v", err)
	}
	card, err = c.rds.ZCard(ctx, c.tsKeyBase(zkey)).Result()
	if err != nil || card != 1 {
		t.Fatalf("zset card after rewrite = %d (err %v), want 1", card, err)
	}
}

// TestStockKLineDepthShortFallback 验证深度不足不会静默返回短窗口：命中
// 判定要求条数完整，不足时走回源；回源不可达则返回错误，而不是把短的
// 缓存窗口当作完整结果返回给调用方。
func TestStockKLineDepthShortFallback(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	zkey := klineCacheKey(8, 1, "000001", 1, 0)
	if err := c.klineStore(ctx, zkey, testBars(100)); err != nil {
		t.Fatalf("store kline: %v", err)
	}

	got, err := c.StockKLine(8, 1, "000001", 0, 200, 1, 0)
	if err == nil {
		t.Fatalf("short depth must not be treated as hit, got %d bars", len(got))
	}
	if c.Stats().Fetches == 0 {
		t.Fatalf("short depth should trigger a source fetch")
	}
}

// TestStockKLineDeepOffsetSingleFetch 验证深 offset 只按原窗口回源一次：
// 不允许先发一次 (0, start+count) 的整段请求（那种请求在深 offset 下会被
// 源端拒绝，还会把失败计进熔断）。
func TestStockKLineDeepOffsetSingleFetch(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	zkey := klineCacheKey(8, 1, "000001", 1, 0)
	if err := c.klineStore(ctx, zkey, testBars(10)); err != nil {
		t.Fatalf("store kline: %v", err)
	}
	before := c.Stats()
	_, _ = c.StockKLine(8, 1, "000001", 5000, 200, 1, 0)
	after := c.Stats()
	// (5000,200) 的窗口回源一次；实时槽刷新是另一路，最多再取一根 (0,2)。
	if delta := after.Fetches - before.Fetches; delta > 2 {
		t.Fatalf("deep offset fetched %d times, want <= 2 (window + live refresh)", delta)
	}
	if delta := after.SourceErrors - before.SourceErrors; delta != 1 {
		t.Fatalf("deep offset source errors = %d, want exactly 1 (no doomed wide query)", delta)
	}
}

// TestKlineStorageUsesMsgpack 验证存储格式为 MessagePack：`ZSet` 成员
// （base64 包裹）与实时槽值（`分数|时刻|base64`）解包后都能按 MessagePack
// 还原整根 SecurityBar，字段端到端无损（不是 JSON，也不是 gob）。
func TestKlineStorageUsesMsgpack(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	zkey := klineCacheKey(8, 1, "000001", 1, 0)
	bars := testBars(3)
	if err := c.klineStore(ctx, zkey, bars); err != nil {
		t.Fatalf("store kline: %v", err)
	}

	members, err := c.rds.ZRange(ctx, c.tsKeyBase(zkey), 0, -1).Result()
	if err != nil {
		t.Fatalf("read zset: %v", err)
	}
	if len(members) != len(bars)-1 {
		t.Fatalf("zset members = %d, want %d", len(members), len(bars)-1)
	}
	raw, err := base64.StdEncoding.DecodeString(members[0])
	if err != nil {
		t.Fatalf("member is not base64: %v", err)
	}
	if len(raw) > 0 && raw[0] == '{' {
		t.Fatalf("member looks like JSON, want MessagePack")
	}
	var closed proto.SecurityBar
	if err := msgpack.Unmarshal(raw, &closed); err != nil {
		t.Fatalf("decode member as msgpack: %v", err)
	}
	if closed.Close != bars[0].Close || !closed.DateTime.Equal(bars[0].DateTime) ||
		closed.Vol != bars[0].Vol || closed.Amount != bars[0].Amount {
		t.Fatalf("member round-trip mismatch: %+v vs %+v", closed, bars[0])
	}

	liveRaw, err := c.rds.Get(ctx, c.liveKey(zkey)).Result()
	if err != nil {
		t.Fatalf("read live: %v", err)
	}
	lv, err := decodeLiveValue(liveRaw)
	if err != nil {
		t.Fatalf("decode live: %v", err)
	}
	var live proto.SecurityBar
	if err := msgpack.Unmarshal(lv.Data, &live); err != nil {
		t.Fatalf("decode live as msgpack: %v", err)
	}
	if live.Close != bars[len(bars)-1].Close ||
		!live.DateTime.Equal(bars[len(bars)-1].DateTime) {
		t.Fatalf("live round-trip mismatch: %+v vs %+v", live, bars[len(bars)-1])
	}
}

// TestIntegrationStockKLineCache 真实行情端到端验证，需 `GOTDX_INTEGRATION=1`。
//
// 断言要点：Redis 只出现 `ZSet` 与实时槽两种键；`ZSet` 基数为回源条数减
// 一（最新一根只在实时槽）；`ZSet` 最大分数等于第二新根；二次查询命中且
// 不新增回源；缓存结果与原始接口一致（允许协议多取导致的 1~2 根边界差）。
func TestIntegrationStockKLineCache(t *testing.T) {
	if os.Getenv("GOTDX_INTEGRATION") != "1" {
		t.Skip("set GOTDX_INTEGRATION=1 to run integration test")
	}
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = rdb.Close() }()
	cli := gotdx.New()
	c := NewCacheClient(rdb, cli)
	ctx := context.Background()

	if keys, err := rdb.Keys(ctx, "*").Result(); err == nil && len(keys) > 0 {
		_ = rdb.Del(ctx, keys...).Err()
	}

	const category, market, code = uint16(8), uint8(0), "000001"
	zkey := klineCacheKey(category, market, code, 1, 0)

	first, err := c.StockKLine(category, market, code, 0, 200, 1, 0)
	if err != nil {
		t.Fatalf("first query: %v", err)
	}
	if len(first) < 100 {
		t.Fatalf("first query returned only %d bars", len(first))
	}
	assertAscending(t, first)

	// 键形态：只有 :ts: 的 ZSet 与 :live，不出现 :kv: / :empty。
	keys, err := rdb.Keys(ctx, "gotdx:ts:kline:*").Result()
	if err != nil {
		t.Fatalf("list keys: %v", err)
	}
	var live bool
	for _, k := range keys {
		switch {
		case k == c.liveKey(zkey):
			live = true
		case k == c.tsKeyBase(zkey):
		default:
			t.Fatalf("unexpected key: %s", k)
		}
	}
	if !live {
		t.Fatalf("live slot missing, keys = %v", keys)
	}
	if kvKeys, _ := rdb.Keys(ctx, "gotdx:kv:*").Result(); len(kvKeys) > 0 {
		t.Fatalf("unexpected kv keys: %v", kvKeys)
	}

	card, err := rdb.ZCard(ctx, c.tsKeyBase(zkey)).Result()
	if err != nil {
		t.Fatalf("zset card: %v", err)
	}
	if int(card) != len(first)-1 {
		t.Fatalf("zset card = %d, want %d (latest bar must stay in live slot)", card, len(first)-1)
	}

	// 实时槽分数应等于最新一根；ZSet 最大分数应等于第二新根（最新根的前一根）。
	last := first[len(first)-1]
	prev := first[len(first)-2]
	liveRaw, err := rdb.Get(ctx, c.liveKey(zkey)).Result()
	if err != nil {
		t.Fatalf("read live: %v", err)
	}
	lv, err := decodeLiveValue(liveRaw)
	if err != nil {
		t.Fatalf("decode live: %v", err)
	}
	if int64(lv.Score) != last.DateTime.UnixMilli() {
		t.Fatalf("live score = %d, want %d", int64(lv.Score), last.DateTime.UnixMilli())
	}
	top, err := rdb.ZRevRangeWithScores(ctx, c.tsKeyBase(zkey), 0, 0).Result()
	if err != nil || len(top) != 1 {
		t.Fatalf("zset top: %v (err %v)", top, err)
	}
	if int64(top[0].Score) != prev.DateTime.UnixMilli() {
		t.Fatalf("zset top score = %d, want %d", int64(top[0].Score), prev.DateTime.UnixMilli())
	}

	// 二次查询：命中且不新增回源（5 秒内实时槽仍新鲜）。
	before := c.Stats()
	second, err := c.StockKLine(category, market, code, 0, 200, 1, 0)
	if err != nil {
		t.Fatalf("second query: %v", err)
	}
	if len(second) != len(first) {
		t.Fatalf("second len = %d, want %d", len(second), len(first))
	}
	if second[len(second)-1].DateTime.UnixMilli() != last.DateTime.UnixMilli() {
		t.Fatalf("second latest = %s, want %s",
			second[len(second)-1].DateTime, last.DateTime)
	}
	after := c.Stats()
	if after.Hits != before.Hits+1 {
		t.Fatalf("second query hits = %d, want %d", after.Hits, before.Hits+1)
	}
	if after.Fetches > before.Fetches+1 {
		t.Fatalf("second query fetched %d times, want <= 1", after.Fetches-before.Fetches)
	}

	// 与原始接口对比：条数与首尾时间允许 1~2 根边界差。
	raw, err := cli.StockKLine(category, market, code, 0, 200, 1, 0)
	if err != nil {
		t.Fatalf("raw query: %v", err)
	}
	if diff := len(raw) - len(first); diff > 2 || diff < -2 {
		t.Fatalf("raw len = %d, cached len = %d", len(raw), len(first))
	}
	if raw[len(raw)-1].DateTime.UnixMilli() != first[len(first)-1].DateTime.UnixMilli() {
		t.Fatalf("latest bar mismatch: raw %s, cached %s",
			raw[len(raw)-1].DateTime, first[len(first)-1].DateTime)
	}

	// 深分页：窗口位置必须落在正确 offset，用「连续子序列定位」校验，
	// 避免拿两次独立请求逐条硬比时被 1~2 根边界差误判成错位。
	wide, err := cli.StockKLine(category, market, code, 0, 600, 1, 0)
	if err != nil {
		t.Fatalf("raw wide query: %v", err)
	}
	for _, tc := range []struct {
		start uint16
		count uint16
	}{
		{200, 200},
		{400, 200},
	} {
		page, perr := c.StockKLine(category, market, code, tc.start, tc.count, 1, 0)
		if perr != nil {
			t.Fatalf("deep page %d: %v", tc.start, perr)
		}
		if len(page) == 0 {
			continue
		}
		assertAscending(t, page)
		off := subseqOffsetFromNewest(page, wide)
		if off < 0 {
			t.Fatalf("page start=%d is not a contiguous slice of raw data", tc.start)
		}
		if diff := off - int(tc.start); diff > 2 || diff < -2 {
			t.Fatalf("page start=%d located at offset %d (diff %d)", tc.start, off, diff)
		}
	}

	// 极深 offset：缓存深度远不足，按原窗口 (5000,200) 直通一次（不发
	// 注定失败的整段请求），结果须与原始接口一致、且条数不被截断。
	deep, err := c.StockKLine(category, market, code, 5000, 200, 1, 0)
	if err != nil {
		t.Fatalf("deep offset: %v", err)
	}
	rawDeep, err := cli.StockKLine(category, market, code, 5000, 200, 1, 0)
	if err != nil {
		t.Fatalf("raw deep offset: %v", err)
	}
	if len(deep) != len(rawDeep) {
		t.Fatalf("deep offset len = %d, raw = %d", len(deep), len(rawDeep))
	}
}

// TestPositionPagerLiveRefreshFrequency 统计 head 回源函数体被调用的真实
// 次数，确认「每次查询都回源」不成立：槽在新鲜度窗口内应直接短路。
func TestPositionPagerLiveRefreshFrequency(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	data := make([]int, 20)
	for i := range data {
		data[i] = i + 1
	}
	headCalls, fetchCalls := 0, 0
	p := c.newPositionPager(
		"refresh:demo",
		func(ctx context.Context, start, count int) ([]int, error) {
			fetchCalls++
			hi := len(data) - int(start)
			if hi <= 0 {
				return nil, nil
			}
			lo := hi - int(count)
			if lo < 0 {
				lo = 0
			}
			return data[lo:hi], nil
		},
		func(ctx context.Context) ([]int, error) {
			headCalls++
			return data[len(data)-2:], nil
		},
		2,
		func(v int) float64 { return float64(v) },
		5*time.Second,
	)

	for i := 0; i < 5; i++ {
		if _, err := p.query(ctx, 0, 10); err != nil {
			t.Fatalf("query %d: %v", i, err)
		}
	}
	if headCalls != 1 {
		t.Fatalf("head called %d times in 5 consecutive queries, want 1", headCalls)
	}
	if fetchCalls != 1 {
		t.Fatalf("fetch called %d times in 5 consecutive queries, want 1", fetchCalls)
	}
}

// deepPageFixture 构造一个「index 0 为最新端」的序列与窗口函数，供位置
// 分页测试复用。fetch/head 的约定是旧 -> 新（与协议层返回一致）。
type deepPageFixture struct {
	total  int
	window func(start, count int) []int
}

func newDeepPageFixture(total int) deepPageFixture {
	data := make([]int, total)
	for i := range data {
		data[i] = total - i // data[0] 最新
	}
	return deepPageFixture{
		total: total,
		window: func(start, count int) []int {
			if start >= total {
				return nil
			}
			hi := start + count
			if hi > total {
				hi = total
			}
			out := make([]int, hi-start)
			for k := 0; k < hi-start; k++ {
				out[k] = data[hi-1-k] // 旧 -> 新
			}
			return out
		},
	}
}

// TestPositionPagerSkipsRefreshForDeepPage 验证 start>0 的翻页不因新鲜度
// 刷新实时槽：深页窗口不含未闭合的最新一条，值旧不影响返回内容，刷新
// 只会白打一次源。
func TestPositionPagerSkipsRefreshForDeepPage(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	fx := newDeepPageFixture(50)

	headCalls := 0
	p := c.newPositionPager(
		"deep:demo",
		func(ctx context.Context, start, count int) ([]int, error) {
			return fx.window(int(start), int(count)), nil
		},
		func(ctx context.Context) ([]int, error) {
			headCalls++
			return fx.window(0, 2), nil
		},
		2,
		func(v int) float64 { return float64(v) },
		5*time.Second,
	)

	// 第一页写满缓存并刷新一次实时槽。
	if _, err := p.query(ctx, 0, 10); err != nil {
		t.Fatalf("query page 1: %v", err)
	}
	if headCalls != 1 {
		t.Fatalf("first page head calls = %d, want 1", headCalls)
	}

	// 槽仍在（只是不再需要更新新鲜度），查深页不得刷新。
	headCalls = 0
	if _, err := p.query(ctx, 10, 10); err != nil {
		t.Fatalf("query page 2: %v", err)
	}
	if headCalls != 0 {
		t.Fatalf("deep page refreshed live slot %d times, want 0", headCalls)
	}
}

// TestPositionPagerRestoresSlotForDeepPage 是上一条的兜底：`ZSet` 只存已
// 闭合条目、最新一条在实时槽里，槽过期后位置基准会整体前移一位，深页窗口
// 将静默返回错位数据。因此槽缺失时深页必须补刷一次，且返回内容必须与
// 直连一致。
func TestPositionPagerRestoresSlotForDeepPage(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	const total = 40
	fx := newDeepPageFixture(total)

	headCalls := 0
	p := c.newPositionPager(
		"restore:demo",
		func(ctx context.Context, start, count int) ([]int, error) {
			return fx.window(int(start), int(count)), nil
		},
		func(ctx context.Context) ([]int, error) {
			headCalls++
			return fx.window(0, 2), nil
		},
		2,
		func(v int) float64 { return float64(v) },
		5*time.Second,
	)

	// 写满前 20 条。
	if _, err := p.query(ctx, 0, 20); err != nil {
		t.Fatalf("query page 1: %v", err)
	}

	// 模拟实时槽过期：`ZSet` 只剩 index 1..19，位置基准已前移一位。
	n, err := c.rds.Del(ctx, c.liveKey("restore:demo")).Result()
	if err != nil || n != 1 {
		t.Fatalf("drop live slot: n=%d err=%v", n, err)
	}
	if exists, _ := c.rds.Exists(ctx, c.liveKey("restore:demo")).Result(); exists != 0 {
		t.Fatalf("live slot still exists after Del")
	}

	headCalls = 0
	got, err := p.query(ctx, 5, 10)
	if err != nil {
		t.Fatalf("query deep page: %v", err)
	}
	if headCalls != 1 {
		t.Fatalf("deep page head calls = %d, want 1 (slot must be restored)", headCalls)
	}
	// 窗口 [5,15) 的正确内容：index 19..5 倒排，即 total-14 .. total-5。
	want := make([]int, 10)
	for i := range want {
		want[i] = total - 14 + i
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("deep window misplaced at %d: got %v want %v", i, got, want)
		}
	}
}

// TestPositionPagerReusable 证明工厂产出的是通用引擎：用 int 序列（与 K 线
// 无关的类型）构造 pager，验证「倒序 offset 窗口 + 实时槽 + 连续前缀」对
// 任意类型同样成立，后续同类接口接入无需重写缓存逻辑。
func TestPositionPagerReusable(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	data := make([]int, 50) // 从旧到新，值单调递增
	for i := range data {
		data[i] = i + 1
	}
	// window 按 start/count 语义从最新端倒数取窗口，返回从旧到新。
	window := func(start, count int) []int {
		hi := len(data) - start
		if hi <= 0 {
			return nil
		}
		lo := hi - count
		if lo < 0 {
			lo = 0
		}
		return data[lo:hi]
	}
	p := c.newPositionPager(
		"positions:demo",
		func(ctx context.Context, start, count int) ([]int, error) {
			return window(int(start), int(count)), nil
		},
		func(ctx context.Context) ([]int, error) {
			return data[len(data)-2:], nil
		},
		2,
		func(v int) float64 { return float64(v) },
		5*time.Second,
	)

	first, err := p.query(ctx, 0, 10)
	if err != nil {
		t.Fatalf("query page 1: %v", err)
	}
	if len(first) != 10 || first[0] != 41 || first[9] != 50 {
		t.Fatalf("page 1 = %v, want 41..50", first)
	}

	second, err := p.query(ctx, 10, 10)
	if err != nil {
		t.Fatalf("query page 2: %v", err)
	}
	if len(second) != 10 || second[0] != 31 || second[9] != 40 {
		t.Fatalf("page 2 = %v, want 31..40", second)
	}

	before := c.Stats()
	if _, err := p.query(ctx, 0, 10); err != nil {
		t.Fatalf("query page 1 again: %v", err)
	}
	if c.Stats().Hits != before.Hits+1 {
		t.Fatalf("third query should hit cache, hits %d -> %d",
			before.Hits, c.Stats().Hits)
	}
}

// subseqOffsetFromNewest 在从旧到新排列的 hay 中定位 needle 连续子序列，
// 返回 needle 最新一根相对 hay 最新端的 offset（即业务 start 的语义）；
// 未找到返回 -1。
func subseqOffsetFromNewest(needle, hay []proto.SecurityBar) int {
	if len(needle) == 0 || len(needle) > len(hay) {
		return -1
	}
	for k := 0; k+len(needle) <= len(hay); k++ {
		match := true
		for i := range needle {
			if hay[k+i].DateTime.UnixMilli() != needle[i].DateTime.UnixMilli() ||
				hay[k+i].Close != needle[i].Close {
				match = false
				break
			}
		}
		if match {
			return len(hay) - k - len(needle)
		}
	}
	return -1
}

// testIndexBars 构造从 09:31 起每分钟一根的指数 K 线序列（从旧到新），
// 收盘点位按序号递增便于断言窗口位置。
func testIndexBars(n int) []proto.IndexBar {
	base := time.Date(2026, 11, 6, 9, 31, 0, 0, time.Local)
	bars := make([]proto.IndexBar, n)
	for i := range bars {
		ts := base.Add(time.Duration(i) * time.Minute)
		bars[i] = proto.IndexBar{
			Open:      float64(i + 1),
			Close:     float64(i + 1),
			High:      float64(i + 1),
			Low:       float64(i + 1),
			UpCount:   uint16(i + 1),
			DownCount: uint16(n - i),
			DateTime:  ts,
		}
	}
	return bars
}

// assertIndexAscending 断言指数 K 线序列按时间从早到晚严格递增。
func assertIndexAscending(tb testing.TB, bars []proto.IndexBar) {
	tb.Helper()
	for i := 1; i < len(bars); i++ {
		if !bars[i-1].DateTime.Before(bars[i].DateTime) {
			tb.Fatalf("index bars not ascending at %d: %s >= %s",
				i, bars[i-1].DateTime, bars[i].DateTime)
		}
	}
}

// TestGetIndexBarsCacheHitWindow 验证命中路径与 StockKLine 同构：窗口按 offset
// 正确定位、返回顺序从早到晚、不触碰行情源，且返回值是 []proto.IndexBar。
func TestGetIndexBarsCacheHitWindow(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	key := indexBarCacheKey(8, 1, "000001")
	bars := testIndexBars(200)
	if err := c.storeWindow(ctx, key, bars, func(b proto.IndexBar) float64 {
		return float64(b.DateTime.UnixMilli())
	}); err != nil {
		t.Fatalf("store index bars: %v", err)
	}

	cases := []struct {
		name       string
		start      uint16
		count      uint16
		firstClose float64
		lastClose  float64
	}{
		{"start=0 取整窗", 0, 200, 1, 200},
		{"start=0 取最新 20 根", 0, 20, 181, 200},
		{"start=10 count=20", 10, 20, 171, 190},
		{"start=199 count=1", 199, 1, 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := c.Stats()
			var got []proto.IndexBar
			var err error
			got, err = c.GetIndexBars(8, 1, "000001", tc.start, tc.count)
			if err != nil {
				t.Fatalf("GetIndexBars: %v", err)
			}
			if len(got) != int(tc.count) {
				t.Fatalf("len = %d, want %d", len(got), tc.count)
			}
			if got[0].Close != tc.firstClose || got[len(got)-1].Close != tc.lastClose {
				t.Fatalf("window = [%v, %v], want [%v, %v]",
					got[0].Close, got[len(got)-1].Close, tc.firstClose, tc.lastClose)
			}
			if got[0].UpCount == 0 || got[0].DownCount == 0 {
				t.Fatalf("IndexBar 专属字段未保留: up=%d down=%d",
					got[0].UpCount, got[0].DownCount)
			}
			assertIndexAscending(t, got)
			after := c.Stats()
			if after.Hits != before.Hits+1 || after.Fetches != before.Fetches {
				t.Fatalf("stats hits/fetches = %d/%d, want %d/%d",
					after.Hits, after.Fetches, before.Hits+1, before.Fetches)
			}
		})
	}
}

// TestGetIndexBarsCacheKeyIsolation 验证周期类别、市场、代码各自独立成键。
func TestGetIndexBarsCacheKeyIsolation(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	store := func(category uint16, market uint8, code string, n int) {
		key := indexBarCacheKey(category, market, code)
		if err := c.storeWindow(ctx, key, testIndexBars(n), func(b proto.IndexBar) float64 {
			return float64(b.DateTime.UnixMilli())
		}); err != nil {
			t.Fatalf("store %s: %v", key, err)
		}
	}
	store(8, 1, "000001", 10)
	store(4, 1, "000001", 20)
	store(8, 0, "000001", 30)
	store(8, 1, "399001", 40)

	cases := []struct {
		category uint16
		market   uint8
		code     string
		wantLen  int
	}{
		{8, 1, "000001", 10},
		{4, 1, "000001", 20},
		{8, 0, "000001", 30},
		{8, 1, "399001", 40},
	}
	for _, tc := range cases {
		got, err := c.GetIndexBars(tc.category, tc.market, tc.code, 0, uint16(tc.wantLen))
		if err != nil {
			t.Fatalf("GetIndexBars(%d,%d,%s): %v", tc.category, tc.market, tc.code, err)
		}
		if len(got) != tc.wantLen {
			t.Fatalf("key(%d,%d,%s) len = %d, want %d",
				tc.category, tc.market, tc.code, len(got), tc.wantLen)
		}
	}
	if !strings.Contains(indexBarCacheKey(8, 1, "000001"), "indexbars:8:1:000001") {
		t.Fatalf("unexpected key: %s", indexBarCacheKey(8, 1, "000001"))
	}
}

// TestGetIndexBarsCountZero 验证 count 为零时不触网、返回空切片无错误。
func TestGetIndexBarsCountZero(t *testing.T) {
	c := newTestCache(t)
	got, err := c.GetIndexBars(8, 1, "000001", 0, 0)
	if err != nil || len(got) != 0 {
		t.Fatalf("count=0 should return empty, got %v err %v", got, err)
	}
	stats := c.Stats()
	if stats.Fetches != 0 {
		t.Fatalf("count=0 should not fetch, fetches = %d", stats.Fetches)
	}
}

// testExKLineItems 构造从 09:31 起每分钟一根的扩展市场 K 线（从旧到新），
// 收盘价与成交量按序号递增便于断言窗口位置。
func testExKLineItems(n int) []proto.ExKLineItem {
	base := time.Date(2026, 11, 6, 9, 31, 0, 0, time.Local)
	items := make([]proto.ExKLineItem, n)
	for i := range items {
		items[i] = proto.ExKLineItem{
			DateTime: base.Add(time.Duration(i) * time.Minute),
			Open:     float64(i + 1),
			High:     float64(i + 1),
			Low:      float64(i + 1),
			Close:    float64(i + 1),
			Vol:      uint32(i + 1),
		}
	}
	return items
}

// assertExKLineAscending 断言扩展 K 线序列按时间从早到晚严格递增。
func assertExKLineAscending(tb testing.TB, items []proto.ExKLineItem) {
	tb.Helper()
	for i := 1; i < len(items); i++ {
		if !items[i-1].DateTime.Before(items[i].DateTime) {
			tb.Fatalf("ex kline not ascending at %d: %s >= %s",
				i, items[i-1].DateTime, items[i].DateTime)
		}
	}
}

// TestExGetKLine2CacheHitWindow 验证命中路径与 StockKLine 同构：窗口按 offset
// 正确定位、返回顺序从早到晚、不触碰行情源，且返回值是 []proto.ExKLineItem。
func TestExGetKLine2CacheHitWindow(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	key := exKLine2CacheKey(1, "TSLA", 4, 1)
	items := testExKLineItems(200)
	if err := c.storeWindow(ctx, key, items, func(b proto.ExKLineItem) float64 {
		return float64(b.DateTime.UnixMilli())
	}); err != nil {
		t.Fatalf("store ex kline: %v", err)
	}

	cases := []struct {
		name      string
		start     uint32
		count     uint32
		firstOpen float64
		lastVol   uint32
	}{
		{"start=0 取整窗", 0, 200, 1, 200},
		{"start=0 取最新 20 根", 0, 20, 181, 200},
		{"start=10 count=20", 10, 20, 171, 190},
		{"start=199 count=1", 199, 1, 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := c.Stats()
			got, err := c.ExGetKLine2(1, "TSLA", 4, tc.start, tc.count, 1)
			if err != nil {
				t.Fatalf("ExGetKLine2: %v", err)
			}
			if len(got) != int(tc.count) {
				t.Fatalf("len = %d, want %d", len(got), tc.count)
			}
			if got[0].Open != tc.firstOpen || got[len(got)-1].Vol != tc.lastVol {
				t.Fatalf("window = [%v, %v], want [%v, %v]",
					got[0].Open, got[len(got)-1].Vol, tc.firstOpen, tc.lastVol)
			}
			assertExKLineAscending(t, got)
			after := c.Stats()
			if after.Hits != before.Hits+1 || after.Fetches != before.Fetches {
				t.Fatalf("stats hits/fetches = %d/%d, want %d/%d",
					after.Hits, after.Fetches, before.Hits+1, before.Fetches)
			}
		})
	}
}

// TestExGetKLine2CacheKeyIsolation 验证市场类别、代码、周期类别与周期倍数
// 各自独立成键；times 为零按 1 归一。
func TestExGetKLine2CacheKeyIsolation(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	store := func(category uint8, code string, period, times uint16, n int) {
		key := exKLine2CacheKey(category, code, period, times)
		if err := c.storeWindow(ctx, key, testExKLineItems(n), func(b proto.ExKLineItem) float64 {
			return float64(b.DateTime.UnixMilli())
		}); err != nil {
			t.Fatalf("store %s: %v", key, err)
		}
	}
	store(1, "TSLA", 4, 1, 10)
	store(1, "TSLA", 0, 1, 20)
	store(2, "TSLA", 4, 1, 40)
	store(1, "AAPL", 4, 1, 50)

	if got := exKLine2CacheKey(1, "TSLA", 4, 0); got != exKLine2CacheKey(1, "TSLA", 4, 1) {
		t.Fatalf("times 0/1 应为同一键: %s", got)
	}

	cases := []struct {
		category uint8
		code     string
		period   uint16
		count    uint32
		wantLen  int
	}{
		{1, "TSLA", 4, 10, 10},
		{1, "TSLA", 0, 20, 20},
		{2, "TSLA", 4, 40, 40},
		{1, "AAPL", 4, 50, 50},
	}
	for _, tc := range cases {
		got, err := c.ExGetKLine2(tc.category, tc.code, tc.period, 0, tc.count, 1)
		if err != nil {
			t.Fatalf("ExGetKLine2(%d,%s,%d): %v", tc.category, tc.code, tc.period, err)
		}
		if len(got) != tc.wantLen {
			t.Fatalf("key(%d,%s,%d) len = %d, want %d",
				tc.category, tc.code, tc.period, len(got), tc.wantLen)
		}
	}
}

// TestExGetKLine2CountZero 验证 count 为零时不触网、返回空切片无错误。
func TestExGetKLine2CountZero(t *testing.T) {
	c := newTestCache(t)
	got, err := c.ExGetKLine2(1, "TSLA", 4, 0, 0, 1)
	if err != nil || len(got) != 0 {
		t.Fatalf("count=0 should return empty, got %v err %v", got, err)
	}
	if stats := c.Stats(); stats.Fetches != 0 {
		t.Fatalf("count=0 should not fetch, fetches = %d", stats.Fetches)
	}
}

// TestExGetKLine2TimesNormalize 验证 times 传 0 与传 1 命中同一份缓存。
func TestExGetKLine2TimesNormalize(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	key := exKLine2CacheKey(1, "TSLA", 4, 1)
	if err := c.storeWindow(ctx, key, testExKLineItems(10), func(b proto.ExKLineItem) float64 {
		return float64(b.DateTime.UnixMilli())
	}); err != nil {
		t.Fatalf("store ex kline: %v", err)
	}
	before := c.Stats()
	got, err := c.ExGetKLine2(1, "TSLA", 4, 0, 10, 0)
	if err != nil {
		t.Fatalf("ExGetKLine2: %v", err)
	}
	if len(got) != 10 {
		t.Fatalf("len = %d, want 10", len(got))
	}
	after := c.Stats()
	if after.Hits != before.Hits+1 || after.Fetches != before.Fetches {
		t.Fatalf("times=0 应命中同一键, hits/fetches = %d/%d, want %d/%d",
			after.Hits, after.Fetches, before.Hits+1, before.Fetches)
	}
}

// testMACBars 构造从 09:31 起每分钟一根的 MAC 证券 K 线（从旧到新），
// 收盘价按序号递增、换手率随序号变化，便于断言窗口位置与回填字段。
func testMACBars(n int) []proto.MACSymbolBar {
	base := time.Date(2026, 11, 6, 9, 31, 0, 0, time.Local)
	bars := make([]proto.MACSymbolBar, n)
	for i := range bars {
		bars[i] = proto.MACSymbolBar{
			DateTime: base.Add(time.Duration(i) * time.Minute),
			Open:     float64(i + 1),
			High:     float64(i + 1),
			Low:      float64(i + 1),
			Close:    float64(i + 1),
			Vol:      float64(i + 1),
			Turnover: float64(i+1) / 100,
		}
	}
	return bars
}

// assertMACBarsAscending 断言 MAC K 线序列按时间从早到晚严格递增。
func assertMACBarsAscending(tb testing.TB, bars []proto.MACSymbolBar) {
	tb.Helper()
	for i := 1; i < len(bars); i++ {
		if !bars[i-1].DateTime.Before(bars[i].DateTime) {
			tb.Fatalf("mac bars not ascending at %d: %s >= %s",
				i, bars[i-1].DateTime, bars[i].DateTime)
		}
	}
}

// TestMACSymbolBarsCacheHitWindow 验证命中路径与 StockKLine 同构：窗口按
// offset 正确定位、返回顺序从早到晚、不触碰行情源，回填的换手率保持写入时快照。
func TestMACSymbolBarsCacheHitWindow(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	key := macBarsCacheKey(0, "000001", 4, 1, 0)
	bars := testMACBars(200)
	if err := c.storeWindow(ctx, key, bars, func(b proto.MACSymbolBar) float64 {
		return float64(b.DateTime.UnixMilli())
	}); err != nil {
		t.Fatalf("store mac bars: %v", err)
	}

	cases := []struct {
		name       string
		start      uint32
		count      uint32
		firstClose float64
		turnover   float64
	}{
		{"start=0 取整窗", 0, 200, 1, 0.01},
		{"start=0 取最新 20 根", 0, 20, 181, 1.81},
		{"start=10 count=20", 10, 20, 171, 1.71},
		{"start=199 count=1", 199, 1, 1, 0.01},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := c.Stats()
			got, err := c.MACSymbolBars(0, "000001", 4, 1, tc.start, tc.count, 0)
			if err != nil {
				t.Fatalf("MACSymbolBars: %v", err)
			}
			if len(got) != int(tc.count) {
				t.Fatalf("len = %d, want %d", len(got), tc.count)
			}
			if got[0].Close != tc.firstClose || got[0].Turnover != tc.turnover {
				t.Fatalf("window head = close %v turnover %v, want %v/%v",
					got[0].Close, got[0].Turnover, tc.firstClose, tc.turnover)
			}
			assertMACBarsAscending(t, got)
			after := c.Stats()
			if after.Hits != before.Hits+1 || after.Fetches != before.Fetches {
				t.Fatalf("stats hits/fetches = %d/%d, want %d/%d",
					after.Hits, after.Fetches, before.Hits+1, before.Fetches)
			}
		})
	}
}

// TestMACSymbolBarsCacheKeyIsolation 验证市场、代码、周期类别、周期倍数与
// 复权方式各自独立成键；times 为零按 1 归一。
func TestMACSymbolBarsCacheKeyIsolation(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	store := func(market uint8, code string, period, times, adjust uint16, n int) {
		key := macBarsCacheKey(market, code, period, times, adjust)
		if err := c.storeWindow(ctx, key, testMACBars(n), func(b proto.MACSymbolBar) float64 {
			return float64(b.DateTime.UnixMilli())
		}); err != nil {
			t.Fatalf("store %s: %v", key, err)
		}
	}
	store(0, "000001", 4, 1, 0, 10)
	store(1, "000001", 4, 1, 0, 20)
	store(0, "600000", 4, 1, 0, 30)
	store(0, "000001", 0, 1, 0, 40)
	store(0, "000001", 0, 2, 0, 50)
	store(0, "000001", 0, 2, 1, 60)

	if got := macBarsCacheKey(0, "000001", 4, 0, 0); got != macBarsCacheKey(0, "000001", 4, 1, 0) {
		t.Fatalf("times 0/1 应为同一键: %s", got)
	}

	cases := []struct {
		market  uint8
		code    string
		period  uint16
		times   uint16
		adjust  uint16
		wantLen int
	}{
		{0, "000001", 4, 1, 0, 10},
		{1, "000001", 4, 1, 0, 20},
		{0, "600000", 4, 1, 0, 30},
		{0, "000001", 0, 1, 0, 40},
		{0, "000001", 0, 2, 0, 50},
		{0, "000001", 0, 2, 1, 60},
	}
	for _, tc := range cases {
		got, err := c.MACSymbolBars(tc.market, tc.code, tc.period, tc.times, 0, uint32(tc.wantLen), tc.adjust)
		if err != nil {
			t.Fatalf("MACSymbolBars(%d,%s,%d,%d,%d): %v", tc.market, tc.code, tc.period, tc.times, tc.adjust, err)
		}
		if len(got) != tc.wantLen {
			t.Fatalf("key(%d,%s,%d,%d,%d) len = %d, want %d",
				tc.market, tc.code, tc.period, tc.times, tc.adjust, len(got), tc.wantLen)
		}
	}
}

// TestMACSymbolBarsCountZero 验证 count 为零时不触网、返回空切片无错误。
func TestMACSymbolBarsCountZero(t *testing.T) {
	c := newTestCache(t)
	got, err := c.MACSymbolBars(0, "000001", 4, 1, 0, 0, 0)
	if err != nil || len(got) != 0 {
		t.Fatalf("count=0 should return empty, got %v err %v", got, err)
	}
	if stats := c.Stats(); stats.Fetches != 0 {
		t.Fatalf("count=0 should not fetch, fetches = %d", stats.Fetches)
	}
}

// TestExGetKLineCacheHitWindow 验证命中路径与 StockKLine 同构：窗口按 offset
// 正确定位、返回顺序从早到晚、不触碰行情源，且返回值是 []proto.ExKLineItem。
func TestExGetKLineCacheHitWindow(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	key := exKLineCacheKey(1, "TSLA", 4, 1)
	items := testExKLineItems(200)
	if err := c.storeWindow(ctx, key, items, func(b proto.ExKLineItem) float64 {
		return float64(b.DateTime.UnixMilli())
	}); err != nil {
		t.Fatalf("store ex kline: %v", err)
	}

	cases := []struct {
		name      string
		start     uint32
		count     uint16
		firstOpen float64
		lastVol   uint32
	}{
		{"start=0 取整窗", 0, 200, 1, 200},
		{"start=0 取最新 20 根", 0, 20, 181, 200},
		{"start=10 count=20", 10, 20, 171, 190},
		{"start=199 count=1", 199, 1, 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := c.Stats()
			got, err := c.ExGetKLine(1, "TSLA", 4, tc.start, tc.count, 1)
			if err != nil {
				t.Fatalf("ExGetKLine: %v", err)
			}
			if len(got) != int(tc.count) {
				t.Fatalf("len = %d, want %d", len(got), tc.count)
			}
			if got[0].Open != tc.firstOpen || got[len(got)-1].Vol != tc.lastVol {
				t.Fatalf("window = [%v, %v], want [%v, %v]",
					got[0].Open, got[len(got)-1].Vol, tc.firstOpen, tc.lastVol)
			}
			assertExKLineAscending(t, got)
			after := c.Stats()
			if after.Hits != before.Hits+1 || after.Fetches != before.Fetches {
				t.Fatalf("stats hits/fetches = %d/%d, want %d/%d",
					after.Hits, after.Fetches, before.Hits+1, before.Fetches)
			}
		})
	}
}

// TestExGetKLineCacheKeyIsolation 验证市场类别、代码、周期类别与周期倍数各自
// 独立成键；times 为零按 1 归一；且与第二变体 0x2489 的键互不重叠。
func TestExGetKLineCacheKeyIsolation(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	store := func(category uint8, code string, period, times uint16, n int) {
		key := exKLineCacheKey(category, code, period, times)
		if err := c.storeWindow(ctx, key, testExKLineItems(n), func(b proto.ExKLineItem) float64 {
			return float64(b.DateTime.UnixMilli())
		}); err != nil {
			t.Fatalf("store %s: %v", key, err)
		}
	}
	store(1, "TSLA", 4, 1, 10)
	store(1, "TSLA", 0, 1, 20)
	store(2, "TSLA", 4, 1, 40)
	store(1, "AAPL", 4, 1, 50)

	if got := exKLineCacheKey(1, "TSLA", 4, 0); got != exKLineCacheKey(1, "TSLA", 4, 1) {
		t.Fatalf("times 0/1 应为同一键: %s", got)
	}
	if exKLineCacheKey(1, "TSLA", 4, 1) == exKLine2CacheKey(1, "TSLA", 4, 1) {
		t.Fatalf("0x23ff 与 0x2489 不能共键: %s", exKLineCacheKey(1, "TSLA", 4, 1))
	}

	cases := []struct {
		category uint8
		code     string
		period   uint16
		count    uint16
		wantLen  int
	}{
		{1, "TSLA", 4, 10, 10},
		{1, "TSLA", 0, 20, 20},
		{2, "TSLA", 4, 40, 40},
		{1, "AAPL", 4, 50, 50},
	}
	for _, tc := range cases {
		got, err := c.ExGetKLine(tc.category, tc.code, tc.period, 0, tc.count, 1)
		if err != nil {
			t.Fatalf("ExGetKLine(%d,%s,%d): %v", tc.category, tc.code, tc.period, err)
		}
		if len(got) != tc.wantLen {
			t.Fatalf("key(%d,%s,%d) len = %d, want %d",
				tc.category, tc.code, tc.period, len(got), tc.wantLen)
		}
	}
}

// TestExGetKLineCountZero 验证 count 为零时不触网、返回空切片无错误。
func TestExGetKLineCountZero(t *testing.T) {
	c := newTestCache(t)
	got, err := c.ExGetKLine(1, "TSLA", 4, 0, 0, 1)
	if err != nil || len(got) != 0 {
		t.Fatalf("count=0 should return empty, got %v err %v", got, err)
	}
	if stats := c.Stats(); stats.Fetches != 0 {
		t.Fatalf("count=0 should not fetch, fetches = %d", stats.Fetches)
	}
}

// TestExGetKLineTimesNormalize 验证 times 传 0 与传 1 命中同一份缓存。
func TestExGetKLineTimesNormalize(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	key := exKLineCacheKey(1, "TSLA", 4, 1)
	if err := c.storeWindow(ctx, key, testExKLineItems(10), func(b proto.ExKLineItem) float64 {
		return float64(b.DateTime.UnixMilli())
	}); err != nil {
		t.Fatalf("store ex kline: %v", err)
	}
	before := c.Stats()
	got, err := c.ExGetKLine(1, "TSLA", 4, 0, 10, 0)
	if err != nil {
		t.Fatalf("ExGetKLine: %v", err)
	}
	if len(got) != 10 {
		t.Fatalf("len = %d, want 10", len(got))
	}
	after := c.Stats()
	if after.Hits != before.Hits+1 || after.Fetches != before.Fetches {
		t.Fatalf("times=0 应命中同一键, hits/fetches = %d/%d, want %d/%d",
			after.Hits, after.Fetches, before.Hits+1, before.Fetches)
	}
}

// TestExKLinesShareSameCache 验证 ExKLine 与 ExGetKLine 共用同一个 exkline 键。
//
// 两者是同一协议（`0x23ff`）的两层封装，数据完全相同，因此不应各存一份。
// 这里在 exkline 键上预置 30 根，然后让两条路径交替读同一份数据的不同窗口：
// 若任一方法用了别的键，它会落到不可达的测试源上回源失败，断言随之报错。
func TestExKLinesShareSameCache(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	if err := c.storeWindow(ctx, exKLineCacheKey(1, "TSLA", 4, 1), testExKLineItems(30), func(b proto.ExKLineItem) float64 {
		return float64(b.DateTime.UnixMilli())
	}); err != nil {
		t.Fatalf("store ex kline: %v", err)
	}

	cases := []struct {
		name  string
		call  func() ([]proto.ExKLineItem, error)
		first float64
		last  uint32
	}{
		{"ExGetKLine(0,10)", func() ([]proto.ExKLineItem, error) {
			return c.ExGetKLine(1, "TSLA", 4, 0, 10, 1)
		}, 21, 30},
		{"ExKLine(0,10)", func() ([]proto.ExKLineItem, error) {
			return c.ExKLine(1, "TSLA", 4, 0, 10, 1)
		}, 21, 30},
		{"ExKLine(10,10)", func() ([]proto.ExKLineItem, error) {
			return c.ExKLine(1, "TSLA", 4, 10, 10, 1)
		}, 11, 20},
		{"ExGetKLine(10,10)", func() ([]proto.ExKLineItem, error) {
			return c.ExGetKLine(1, "TSLA", 4, 10, 10, 1)
		}, 11, 20},
		{"ExKLine(20,10)", func() ([]proto.ExKLineItem, error) {
			return c.ExKLine(1, "TSLA", 4, 20, 10, 1)
		}, 1, 10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := c.Stats()
			got, err := tc.call()
			if err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if len(got) != 10 {
				t.Fatalf("len = %d, want 10", len(got))
			}
			if got[0].Open != tc.first || got[len(got)-1].Vol != tc.last {
				t.Fatalf("window = [%v, %v], want [%v, %v]",
					got[0].Open, got[len(got)-1].Vol, tc.first, tc.last)
			}
			assertExKLineAscending(t, got)
			after := c.Stats()
			if after.Hits != before.Hits+1 || after.Fetches != before.Fetches {
				t.Fatalf("共用 exkline 键应命中: hits/fetches = %d/%d, want %d/%d",
					after.Hits, after.Fetches, before.Hits+1, before.Fetches)
			}
		})
	}
}

// TestExKLineCountZero 验证 ExKLine 在 count 为零时不触网、返回空切片无错误。
func TestExKLineCountZero(t *testing.T) {
	c := newTestCache(t)
	got, err := c.ExKLine(1, "TSLA", 4, 0, 0, 1)
	if err != nil || len(got) != 0 {
		t.Fatalf("count=0 should return empty, got %v err %v", got, err)
	}
	if stats := c.Stats(); stats.Fetches != 0 {
		t.Fatalf("count=0 should not fetch, fetches = %d", stats.Fetches)
	}
}
