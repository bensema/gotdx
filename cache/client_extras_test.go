package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/proto"
	"github.com/redis/go-redis/v9"
)

// TestGetTableFileCacheHit 验证竖线表格缓存命中路径：预写缓存后调用方法，
// 直接返回缓存数据（未连接的行情客户端不会被触碰，否则会报错）。
func TestGetTableFileCacheHit(t *testing.T) {
	mr := miniredis.RunT(t)
	c := NewCacheClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), gotdx.New())
	defer func() { _ = c.rds.Close() }()

	rows := [][]string{
		{"code", "name"},
		{"000001", "平安银行"},
	}
	seedKV(t, c, "table:pipe:test.dat", rows)

	out, err := c.GetTableFile(context.Background(), "test.dat")
	if err != nil {
		t.Fatalf("cache hit should succeed, got %v", err)
	}
	if len(out) != 2 || out[1][0] != "000001" || out[1][1] != "平安银行" {
		t.Fatalf("unexpected cached rows: %+v", out)
	}
}

// TestGetCSVFileCacheHit 验证 CSV 缓存命中路径与同形参数键隔离：预写 csv 键后
// 命中；同文件名预写 pipe 键不影响 csv 查询（两方法同形参数不串键）。
func TestGetCSVFileCacheHitAndKeyIsolation(t *testing.T) {
	mr := miniredis.RunT(t)
	c := NewCacheClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), gotdx.New())
	defer func() { _ = c.rds.Close() }()

	rows := [][]string{
		{"000001,平安银行,11.69"},
		{"600000,浦发银行,10.16"},
	}
	seedKV(t, c, "table:csv:test.csv", rows)
	seedKV(t, c, "table:pipe:test.csv", [][]string{{"干扰项"}})

	out, err := c.GetCSVFile(context.Background(), "test.csv")
	if err != nil {
		t.Fatalf("cache hit should succeed, got %v", err)
	}
	if len(out) != 2 || out[0][0] != "000001,平安银行,11.69" {
		t.Fatalf("unexpected cached rows: %+v", out)
	}
}

// TestGetTableFileMissFails 验证未命中且行情源不可用（未连接）时：首次回源失败
// 返回底层错误并写入失败空标记，二次查询直接命中标记返回 ErrSourceFailed。
func TestGetTableFileMissFails(t *testing.T) {
	mr := miniredis.RunT(t)
	c := NewCacheClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), gotdx.New())
	defer func() { _ = c.rds.Close() }()

	_, err := c.GetTableFile(context.Background(), "test.dat")
	if err == nil {
		t.Fatal("unconnected source should fail first fetch")
	}
	_, err = c.GetTableFile(context.Background(), "test.dat")
	if !errors.Is(err, ErrSourceFailed) {
		t.Fatalf("want ErrSourceFailed from failure marker, got %v", err)
	}
}

// TestGetCompanyInfoCacheHit 验证聚合公司信息缓存命中路径：预写 bundle 键后命中，
// 返回的聚合 Bundle 与预写内容一致（含章节/除权除息/财务三段）。
func TestGetCompanyInfoCacheHit(t *testing.T) {
	mr := miniredis.RunT(t)
	c := NewCacheClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), gotdx.New())
	defer func() { _ = c.rds.Close() }()

	bundle := &gotdx.CompanyInfoBundle{
		Sections: []gotdx.CompanyInfoSection{
			{Name: "公司概况", Content: "平安银行股份有限公司"},
			{Name: "股本结构", Content: "总股本 194 亿股"},
		},
		XDXR: []proto.XDXRItem{
			{Code: "000001"},
		},
		Finance: &proto.GetFinanceInfoReply{
			Code: "000001",
		},
	}
	seedKV(t, c, "company:bundle:0:000001", bundle)

	out, err := c.GetCompanyInfo(context.Background(), 0, "000001")
	if err != nil {
		t.Fatalf("cache hit should succeed, got %v", err)
	}
	if len(out.Sections) != 2 || out.Sections[0].Name != "公司概况" ||
		out.Sections[0].Content != "平安银行股份有限公司" {
		t.Fatalf("unexpected cached sections: %+v", out.Sections)
	}
	if len(out.XDXR) != 1 || out.XDXR[0].Code != "000001" {
		t.Fatalf("unexpected cached XDXR: %+v", out.XDXR)
	}
	if out.Finance == nil || out.Finance.Code != "000001" {
		t.Fatalf("unexpected cached finance: %+v", out.Finance)
	}
}

// TestGetCompanyInfoKeyIsolation 验证聚合键与底层组件键不冲突：预写底层
// company:cats 键不影响聚合 company:bundle 查询（聚合键带独立操作标签）。
func TestGetCompanyInfoKeyIsolation(t *testing.T) {
	mr := miniredis.RunT(t)
	c := NewCacheClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), gotdx.New())
	defer func() { _ = c.rds.Close() }()

	// 预写底层分类目录键（干扰项），聚合键未写：未命中应回源失败而非命中干扰键。
	seedKV(t, c, "company:cats:0:000001", &proto.GetCompanyCategoryReply{})
	if err := c.rds.Set(context.Background(), c.kvKey("company:cats:0:000001"),
		"", time.Hour).Err(); err != nil {
		t.Fatal(err)
	}

	_, err := c.GetCompanyInfo(context.Background(), 0, "000001")
	if err == nil {
		t.Fatal("aggregate key should miss and fail on unconnected source")
	}
}
