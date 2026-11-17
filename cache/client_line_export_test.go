// 文件 client_line_export_test.go 以包外视角（package cache_test）验证对外导出的
// K 线写入与读取符号确实可被外部调用，且「外部写入 → 读回」闭环成立。

package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/cache"
	"github.com/bensema/gotdx/proto"
	"github.com/redis/go-redis/v9"
)

// TestExportedKLineWriteAPIFromOutsidePackage 验证 KLineCacheKey、KLineStore、
// KLineUpsertLive、KLineList 四个导出符号能在包外直接调用：外部按键写入后能读回
// 同一批数据，实时槽同分覆盖生效，更旧的一根不会顶掉槽里的新值。
func TestExportedKLineWriteAPIFromOutsidePackage(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	// 行情源指向不可达地址：本用例只走缓存，任何回源都会立刻失败并暴露问题。
	cli := gotdx.New(
		gotdx.WithTCPAddress("127.0.0.1:1"),
		gotdx.WithTCPAddressPool("127.0.0.1:1"),
		gotdx.WithTimeoutSec(1),
	)
	cl := cache.NewCacheClient(rdb, cli)

	// times 传 0，验证键构造的「零值按 1」归一对外可见。
	key := cache.KLineCacheKey(8, 1, "000001", 0, 0)
	bars := make([]proto.SecurityBar, 5)
	base := time.Date(2026, 11, 6, 9, 31, 0, 0, time.Local)
	for i := range bars {
		bars[i] = proto.SecurityBar{
			Close:    float64(i + 1),
			DateTime: base.Add(time.Duration(i) * time.Minute),
		}
	}
	score := func(b proto.SecurityBar) float64 { return float64(b.DateTime.UnixMilli()) }

	if err := cl.KLineStore[proto.SecurityBar](ctx, key, bars, score); err != nil {
		t.Fatalf("KLineStore: %v", err)
	}

	got, err := cl.KLineList[proto.SecurityBar](ctx, key, 0, 5)
	if err != nil {
		t.Fatalf("KLineList: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("len = %d, want 5", len(got))
	}
	for i, b := range got {
		if b.Close != bars[i].Close {
			t.Fatalf("idx %d close = %v, want %v（外部写入与读回不一致）", i, b.Close, bars[i].Close)
		}
	}

	// 同分覆盖：把最新一根的收盘价改掉再 upsert，槽内应刷新为新值。
	newest := bars[4]
	newest.Close = 99
	ok, err := cl.KLineUpsertLive[proto.SecurityBar](ctx, key, newest, score(newest))
	if err != nil {
		t.Fatalf("KLineUpsertLive: %v", err)
	}
	if !ok {
		t.Fatalf("同分覆盖应生效，返回 false")
	}
	latest, err := cl.KLineList[proto.SecurityBar](ctx, key, 0, 1)
	if err != nil || len(latest) != 1 {
		t.Fatalf("读最新一根失败: len=%d err=%v", len(latest), err)
	}
	if latest[0].Close != 99 {
		t.Fatalf("最新一根 close = %v, want 99", latest[0].Close)
	}

	// 更旧的一根 upsert：返回 false 表示未生效，槽内容保持不变。
	older := bars[3]
	older.Close = 77
	ok, err = cl.KLineUpsertLive[proto.SecurityBar](ctx, key, older, score(older))
	if err != nil {
		t.Fatalf("KLineUpsertLive(older): %v", err)
	}
	if ok {
		t.Fatalf("更旧的一根不应覆盖实时槽，返回 true")
	}
	latest, err = cl.KLineList[proto.SecurityBar](ctx, key, 0, 1)
	if err != nil || len(latest) != 1 {
		t.Fatalf("读最新一根失败: len=%d err=%v", len(latest), err)
	}
	if latest[0].Close != 99 {
		t.Fatalf("槽被更旧的一根顶掉: close = %v, want 99", latest[0].Close)
	}

	if stats := cl.Stats(); stats.Fetches != 0 {
		t.Fatalf("全程不应回源，fetches = %d", stats.Fetches)
	}
}
