package cache

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/proto"
	"github.com/redis/go-redis/v9"
)

// newCompanyCache 构造预置 `miniredis` 与未连接行情客户端的缓存对象，供
// 命中路径与未命中失败路径测试使用。
func newCompanyCache(t *testing.T) (*CacheClient, func()) {
	t.Helper()
	mr := miniredis.RunT(t)
	c := NewCacheClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), gotdx.New())
	return c, func() { _ = c.rds.Close() }
}

// TestGetCompanyCategoriesCacheHit 验证分类目录缓存命中：预写缓存后直接返回
// 缓存数据，未连接行情源不被触碰。
func TestGetCompanyCategoriesCacheHit(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	want := &proto.GetCompanyCategoryReply{Count: 1, Categories: []proto.CompanyCategory{
		{Name: "公司概况", Filename: "000001.txt", Start: 0, Length: 128},
	}}
	seedKV(t, c, "company:cats:0:000001", want)

	out, err := c.GetCompanyCategories(context.Background(), 0, "000001")
	if err != nil {
		t.Fatalf("cache hit should succeed, got %v", err)
	}
	if out.Count != 1 || out.Categories[0].Name != "公司概况" || out.Categories[0].Start != 0 {
		t.Fatalf("unexpected cached categories: %+v", out)
	}
}

// TestGetCompanyCategoriesMissFails 验证分类目录未命中且行情源不可用时：
// 首次回源失败返回底层错误并写入失败空标记，二次查询直接命中标记返回
// `ErrSourceFailed`。
func TestGetCompanyCategoriesMissFails(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	if _, err := c.GetCompanyCategories(context.Background(), 0, "000001"); err == nil {
		t.Fatal("first miss must fail when source is down")
	}
	if _, err := c.GetCompanyCategories(context.Background(), 0, "000001"); !errors.Is(err, ErrSourceFailed) {
		t.Fatalf("second query must hit fail marker, got %v", err)
	}
}

// TestGetCompanyContentCacheHit 验证章节正文缓存命中。
func TestGetCompanyContentCacheHit(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	want := &proto.GetCompanyContentReply{Market: 0, Code: "000001", Length: 11, Content: "公司简介内容"}
	seedKV(t, c, "company:content:0:000001:000001.txt:0:128", want)

	out, err := c.GetCompanyContent(context.Background(), 0, "000001", "000001.txt", 0, 128)
	if err != nil {
		t.Fatalf("cache hit should succeed, got %v", err)
	}
	if out.Content != "公司简介内容" || out.Length != 11 {
		t.Fatalf("unexpected cached content: %+v", out)
	}
}

// TestGetCompanyContentMissFails 验证章节正文未命中且行情源不可用时回源失败、
// 二次命中失败标记。
func TestGetCompanyContentMissFails(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	if _, err := c.GetCompanyContent(context.Background(), 0, "000001", "000001.txt", 0, 128); err == nil {
		t.Fatal("first miss must fail when source is down")
	}
	if _, err := c.GetCompanyContent(context.Background(), 0, "000001", "000001.txt", 0, 128); !errors.Is(err, ErrSourceFailed) {
		t.Fatalf("second query must hit fail marker, got %v", err)
	}
}

// TestGetFinanceInfoCacheHit 验证财务概况缓存命中。
func TestGetFinanceInfoCacheHit(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	want := &proto.GetFinanceInfoReply{Market: 1, Code: "600000", FloatShares: 100.5, TotalShares: 200.25}
	seedKV(t, c, "company:finance:1:600000", want)

	out, err := c.GetFinanceInfo(context.Background(), 1, "600000")
	if err != nil {
		t.Fatalf("cache hit should succeed, got %v", err)
	}
	if out.Code != "600000" || out.FloatShares != 100.5 {
		t.Fatalf("unexpected cached finance: %+v", out)
	}
}

// TestGetFinanceInfoMissFails 验证财务概况未命中且行情源不可用时回源失败、
// 二次命中失败标记。
func TestGetFinanceInfoMissFails(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	if _, err := c.GetFinanceInfo(context.Background(), 1, "600000"); err == nil {
		t.Fatal("first miss must fail when source is down")
	}
	if _, err := c.GetFinanceInfo(context.Background(), 1, "600000"); !errors.Is(err, ErrSourceFailed) {
		t.Fatalf("second query must hit fail marker, got %v", err)
	}
}

// TestGetXDXRInfoCacheHit 验证除权除息缓存命中。
func TestGetXDXRInfoCacheHit(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	want := &proto.GetXDXRInfoReply{Market: 1, Code: "600000", Count: 1, List: []proto.XDXRItem{
		{Market: 1, Code: "600000"},
	}}
	seedKV(t, c, "company:xdxr:1:600000", want)

	out, err := c.GetXDXRInfo(context.Background(), 1, "600000")
	if err != nil {
		t.Fatalf("cache hit should succeed, got %v", err)
	}
	if out.Count != 1 || out.List[0].Code != "600000" {
		t.Fatalf("unexpected cached xdxr: %+v", out)
	}
}

// TestGetXDXRInfoMissFails 验证除权除息未命中且行情源不可用时回源失败、
// 二次命中失败标记。
func TestGetXDXRInfoMissFails(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	if _, err := c.GetXDXRInfo(context.Background(), 1, "600000"); err == nil {
		t.Fatal("first miss must fail when source is down")
	}
	if _, err := c.GetXDXRInfo(context.Background(), 1, "600000"); !errors.Is(err, ErrSourceFailed) {
		t.Fatalf("second query must hit fail marker, got %v", err)
	}
}

// TestGetFileMetaCacheHit 验证文件元信息缓存命中。
func TestGetFileMetaCacheHit(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	want := &proto.GetFileMetaReply{Size: 2048, HashValue: [32]byte{1, 2, 3}}
	seedKV(t, c, "file:meta:block_gn.dat", want)

	out, err := c.GetFileMeta(context.Background(), "block_gn.dat")
	if err != nil {
		t.Fatalf("cache hit should succeed, got %v", err)
	}
	if out.Size != 2048 || out.HashValue[0] != 1 {
		t.Fatalf("unexpected cached file meta: %+v", out)
	}
}

// TestGetFileMetaMissFails 验证文件元信息未命中且行情源不可用时回源失败、
// 二次命中失败标记。
func TestGetFileMetaMissFails(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	if _, err := c.GetFileMeta(context.Background(), "block_gn.dat"); err == nil {
		t.Fatal("first miss must fail when source is down")
	}
	if _, err := c.GetFileMeta(context.Background(), "block_gn.dat"); !errors.Is(err, ErrSourceFailed) {
		t.Fatalf("second query must hit fail marker, got %v", err)
	}
}

// TestDownloadFileCacheHit 验证文件分块缓存命中。
func TestDownloadFileCacheHit(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	want := &proto.DownloadFileReply{Size: 3, Data: []byte{0xAA, 0xBB, 0xCC}}
	seedKV(t, c, "file:chunk:block_gn.dat:0:1024", want)

	out, err := c.DownloadFile(context.Background(), "block_gn.dat", 0, 1024)
	if err != nil {
		t.Fatalf("cache hit should succeed, got %v", err)
	}
	if out.Size != 3 || len(out.Data) != 3 || out.Data[2] != 0xCC {
		t.Fatalf("unexpected cached chunk: %+v", out)
	}
}

// TestDownloadFileMissFails 验证文件分块未命中且行情源不可用时回源失败、
// 二次命中失败标记。
func TestDownloadFileMissFails(t *testing.T) {
	c, close := newCompanyCache(t)
	defer close()

	if _, err := c.DownloadFile(context.Background(), "block_gn.dat", 0, 1024); err == nil {
		t.Fatal("first miss must fail when source is down")
	}
	if _, err := c.DownloadFile(context.Background(), "block_gn.dat", 0, 1024); !errors.Is(err, ErrSourceFailed) {
		t.Fatalf("second query must hit fail marker, got %v", err)
	}
}
