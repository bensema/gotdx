package cache

import (
	"context"
	"errors"
	"testing"

	"github.com/bensema/gotdx/proto"
)

// TestExGetQuoteCacheHit 验证扩展市场单品种行情快照缓存命中：预写缓存后直接
// 返回缓存数据，未连接行情源不被触碰。
func TestExGetQuoteCacheHit(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	want := &proto.ExGetQuoteReply{Item: proto.ExQuoteItem{Category: 1, Code: "sh600000", Open: 10.5, Close: 10.8}}
	seedKV(t, c, "ex:quote:1:sh600000", want)

	out, err := c.ExGetQuote(context.Background(), 1, "sh600000")
	if err != nil {
		t.Fatalf("cache hit should succeed, got %v", err)
	}
	if out.Item.Code != "sh600000" || out.Item.Close != 10.8 {
		t.Fatalf("unexpected cached quote: %+v", out)
	}
}

// TestExGetQuoteMissFails 验证快照未命中且行情源不可用时：首次回源失败返回
// 底层错误并写入失败空标记，二次查询直接命中标记返回 `ErrSourceFailed`。
func TestExGetQuoteMissFails(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	if _, err := c.ExGetQuote(context.Background(), 1, "sh600000"); err == nil {
		t.Fatal("first miss must fail when source is down")
	}
	if _, err := c.ExGetQuote(context.Background(), 1, "sh600000"); !errors.Is(err, ErrSourceFailed) {
		t.Fatalf("second query must hit fail marker, got %v", err)
	}
}

// TestExGetQuotesCacheHit 验证批量行情快照缓存命中（键含切片参数拼接）。
func TestExGetQuotesCacheHit(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	want := &proto.ExGetQuotesReply{Count: 2, List: []proto.ExQuoteItem{
		{Category: 1, Code: "sh600000", Close: 10.8},
		{Category: 1, Code: "sh600001", Close: 9.2},
	}}
	seedKV(t, c, "ex:quotes:[1]:[sh600000 sh600001]", want)

	out, err := c.ExGetQuotes(context.Background(), []uint8{1}, []string{"sh600000", "sh600001"})
	if err != nil {
		t.Fatalf("cache hit should succeed, got %v", err)
	}
	if out.Count != 2 || out.List[0].Code != "sh600000" || out.List[1].Close != 9.2 {
		t.Fatalf("unexpected cached quotes: %+v", out)
	}
}

// TestExGetKLineCacheHit 验证扩展市场 K 线缓存命中。
func TestExGetKLineCacheHit(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	want := &proto.ExGetKLineReply{Category: 1, Period: 9, Count: 2, List: []proto.ExKLineItem{{Close: 10.5}, {Close: 10.8}}}
	seedKV(t, c, "ex:kline:1:sh600000:9:0:800:1", want)

	out, err := c.ExGetKLine(context.Background(), 1, "sh600000", 9, 0, 800, 1)
	if err != nil {
		t.Fatalf("cache hit should succeed, got %v", err)
	}
	if out.Count != 2 || out.List[1].Close != 10.8 {
		t.Fatalf("unexpected cached kline: %+v", out)
	}
}

// TestExGetKLineMissFails 验证 K 线未命中且行情源不可用时回源失败、二次命中
// 失败标记。
func TestExGetKLineMissFails(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	if _, err := c.ExGetKLine(context.Background(), 1, "sh600000", 9, 0, 800, 1); err == nil {
		t.Fatal("first miss must fail when source is down")
	}
	if _, err := c.ExGetKLine(context.Background(), 1, "sh600000", 9, 0, 800, 1); !errors.Is(err, ErrSourceFailed) {
		t.Fatalf("second query must hit fail marker, got %v", err)
	}
}

// TestExGetHistoryTransactionCacheHit 验证历史逐笔缓存命中（键含日期参数）。
func TestExGetHistoryTransactionCacheHit(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	want := &proto.ExGetHistoryTransactionReply{Date: "2026-10-30", Count: 1}
	seedKV(t, c, "ex:histtrans:20261030:1:sh600000", want)

	out, err := c.ExGetHistoryTransaction(context.Background(), 20261030, 1, "sh600000")
	if err != nil {
		t.Fatalf("cache hit should succeed, got %v", err)
	}
	if out.Date != "2026-10-30" || out.Count != 1 {
		t.Fatalf("unexpected cached transaction: %+v", out)
	}
}

// TestExGetFileMetaCacheHit 验证扩展站文件元信息缓存命中（`exfile:` 前缀隔离）。
func TestExGetFileMetaCacheHit(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	want := &proto.GetFileMetaReply{Size: 4096, HashValue: [32]byte{9}}
	seedKV(t, c, "exfile:meta:exblock.dat", want)

	out, err := c.ExGetFileMeta(context.Background(), "exblock.dat")
	if err != nil {
		t.Fatalf("cache hit should succeed, got %v", err)
	}
	if out.Size != 4096 || out.HashValue[0] != 9 {
		t.Fatalf("unexpected cached file meta: %+v", out)
	}
}

// TestExDownloadFullFileCacheHit 验证扩展站文件整包下载缓存命中（字节流返回形态）。
func TestExDownloadFullFileCacheHit(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	want := []byte{0x10, 0x20, 0x30}
	seedKV(t, c, "exfile:full:exblock.dat:0", want)

	out, err := c.ExDownloadFullFile(context.Background(), "exblock.dat", 0)
	if err != nil {
		t.Fatalf("cache hit should succeed, got %v", err)
	}
	if len(out) != 3 || out[2] != 0x30 {
		t.Fatalf("unexpected cached full file: %v", out)
	}
}

// TestExGetTableCacheHit 验证扩展站表格缓存命中（文本返回形态）。
func TestExGetTableCacheHit(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	seedKV(t, c, "ex:table", "col1\tcol2\nv1\tv2")

	out, err := c.ExGetTable(context.Background())
	if err != nil {
		t.Fatalf("cache hit should succeed, got %v", err)
	}
	if out != "col1\tcol2\nv1\tv2" {
		t.Fatalf("unexpected cached table: %q", out)
	}
}

// TestExGetServerInfoCacheHit 验证扩展行情服务器信息缓存命中（无参键）。
func TestExGetServerInfoCacheHit(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	want := &proto.ExServerInfoReply{ServerName: "上海扩展行情", Version: "6.4.1.9"}
	seedKV(t, c, "ex:server", want)

	out, err := c.GetExServerInfo(context.Background())
	if err != nil {
		t.Fatalf("cache hit should succeed, got %v", err)
	}
	if out.ServerName != "上海扩展行情" || out.Version != "6.4.1.9" {
		t.Fatalf("unexpected cached server info: %+v", out)
	}
}
