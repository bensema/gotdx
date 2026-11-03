// Command stock_company_raw_cache 演示公司资料族查询（分类目录、章节正文、财务摘要、
// 除权除息记录、聚合打包）的缓存版调用：首次调用回源行情服务器并写入 `Redis`，
// 二次调用直接命中缓存不再触碰行情源；通过 `Stats` 观察命中/回源计数变化。
//
// 与 `stock_company_raw`（原始接口）对应，本示例走 `cache` 缓存层，除聚合 `GetCompanyInfo`
// 外底层组件各自独立建键，命中时多次 `Redis` 往返；聚合层单键一次往返即得整包。
//
// 前置条件：本地 `Redis`（默认 `127.0.0.1:6379`）与主站网络可达。
// 运行示例：`go run ./examples/stock_company_raw_cache`
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bensema/gotdx/cache"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
	"github.com/redis/go-redis/v9"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	if _, err := client.Connect(); err != nil {
		log.Fatalln(err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379", DB: 2})
	defer rdb.Close()
	cached := cache.NewCacheClient(rdb, client)
	ctx := context.Background()

	market := types.MarketSZ.Uint8()
	code := "000001"

	// 分类目录（章节名、文件名、偏移、长度）：首次回源，二次命中。
	log.Println("== GetCompanyCategories ==")
	categories, err := cached.GetCompanyCategories(ctx, market, code)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("company_categories count=%d", categories.Count)
	if _, err := cached.GetCompanyCategories(ctx, market, code); err != nil {
		log.Fatalln(err)
	}
	log.Printf("categories hit again ok stats=%s", statsText(cached.Stats()))

	// 章节正文：参数（文件名、偏移、长度）必须原样取自目录条目；首次回源，二次命中。
	if len(categories.Categories) > 0 {
		first := categories.Categories[0]
		log.Println("== GetCompanyContent ==")
		content, err := cached.GetCompanyContent(ctx, market, code, first.Filename, first.Start, first.Length)
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf("company_content name=%s filename=%s preview=%s",
			first.Name, first.Filename, preview(content.Content, 80))
		if _, err := cached.GetCompanyContent(ctx, market, code, first.Filename, first.Start, first.Length); err != nil {
			log.Fatalln(err)
		}
		log.Printf("content hit again ok stats=%s", statsText(cached.Stats()))
	}

	// 财务摘要：首次回源，二次命中。
	log.Println("== GetFinanceInfo ==")
	finance, err := cached.GetFinanceInfo(ctx, market, code)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("finance code=%s updated=%d total_shares=%.2f revenue=%.2f profit=%.2f",
		finance.Code, finance.UpdatedDate, finance.TotalShares, finance.OperatingRevenue, finance.NetProfit)
	if _, err := cached.GetFinanceInfo(ctx, market, code); err != nil {
		log.Fatalln(err)
	}
	log.Printf("finance hit again ok stats=%s", statsText(cached.Stats()))

	// 除权除息记录：首次回源，二次命中；仅打印前 5 条。
	log.Println("== GetXDXRInfo ==")
	xdxr, err := cached.GetXDXRInfo(ctx, market, code)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("xdxr code=%s count=%d", xdxr.Code, xdxr.Count)
	for _, item := range xdxr.List[:min(5, len(xdxr.List))] {
		log.Printf("xdxr date=%s category=%d name=%s", item.Date.Format("2006-01-02"), item.Category, item.Name)
	}
	if _, err := cached.GetXDXRInfo(ctx, market, code); err != nil {
		log.Fatalln(err)
	}
	log.Printf("xdxr hit again ok stats=%s", statsText(cached.Stats()))

	// 聚合公司信息：底层组件各有缓存键，聚合层再按市场+代码整包缓存，单键一次往返即得。
	log.Println("== GetCompanyInfo ==")
	bundle, err := cached.GetCompanyInfo(ctx, market, code)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("company_bundle sections=%d xdxr=%d finance_nil=%t",
		len(bundle.Sections), len(bundle.XDXR), bundle.Finance == nil)
	if _, err := cached.GetCompanyInfo(ctx, market, code); err != nil {
		log.Fatalln(err)
	}
	log.Printf("bundle hit again ok stats=%s", statsText(cached.Stats()))

	fmt.Printf("最终统计：命中=%d 空标记命中=%d 回源=%d 回源失败=%d 熔断拒绝=%d\n",
		cached.Stats().Hits, cached.Stats().EmptyHits, cached.Stats().Fetches,
		cached.Stats().SourceErrors, cached.Stats().CircuitRejects)
}

// min 返回整数 a 与 b 中较小的一个。
//
// 参数 a、b 为待比较的两个整数；返回值为二者较小值，用于在切片截断时把长度限制在有效范围内，避免越界。
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// preview 去掉文本首尾空白后截断到指定长度，用于在日志中打印章节正文预览。
//
// 参数 `text` 为待预览的原始文本；参数 `limit` 为预览保留的最大字节数。文本未超长时原样返回；超长时按
// `limit` 字节截断（未做字符边界处理，多字节字符可能被切在中间），返回截断后的预览串。
func preview(text string, limit int) string {
	text = strings.TrimSpace(text)
	if len(text) <= limit {
		return text
	}
	return text[:limit]
}

// statsText 把缓存统计快照压成单行文本，便于日志对比两轮计数。
//
// 参数 `s` 为 `cache.Stats` 快照；返回 `命中/空标记命中/回源/回源失败/熔断拒绝` 五个计数。
func statsText(s cache.Stats) string {
	return fmt.Sprintf("hits=%d empty_hits=%d fetches=%d source_errors=%d circuit_rejects=%d",
		s.Hits, s.EmptyHits, s.Fetches, s.SourceErrors, s.CircuitRejects)
}
