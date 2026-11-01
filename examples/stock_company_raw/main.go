// Command stock_company_raw 演示公司资料分类、章节正文、财务摘要、除权除息记录以及聚合打包接口的原始结果获取与打印。
//
// 运行示例：`go run ./examples/stock_company_raw`
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

	// 市场 `types.MarketSZ`（深市），代码 `000001`；返回该证券 F10 资料的章节目录（名称、文件名、偏移、长度）。
	categories, err := client.GetCompanyCategories(types.MarketSZ.Uint8(), "000001")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("company_categories count=%d", categories.Count)

	if len(categories.Categories) > 0 {
		first := categories.Categories[0]
		// 取首个目录条目；`Filename`、`Start`（文件内起始字节偏移）、`Length`（正文字节数）
		// 必须原样取自目录条目，透传给 `GetCompanyContent` 读取章节正文。
		content, err := client.GetCompanyContent(types.MarketSZ.Uint8(), "000001", first.Filename, first.Start, first.Length)
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf("company_content name=%s filename=%s preview=%s",
			first.Name, first.Filename, preview(content.Content, 80))
	}

	// 查询财务摘要，返回总股本、营业收入、净利润等指标；金额/股本单位以协议原始字段为准。
	finance, err := client.GetFinanceInfo(types.MarketSZ.Uint8(), "000001")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("finance code=%s updated=%d total_shares=%.2f revenue=%.2f profit=%.2f",
		finance.Code, finance.UpdatedDate, finance.TotalShares, finance.OperatingRevenue, finance.NetProfit)

	// 查询除权除息历史记录（按日期倒序）；`min(5, len(...))` 防止切片越界，仅打印前 5 条。
	xdxr, err := client.GetXDXRInfo(types.MarketSZ.Uint8(), "000001")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("xdxr code=%s count=%d", xdxr.Code, xdxr.Count)
	for _, item := range xdxr.List[:min(5, len(xdxr.List))] {
		log.Printf("xdxr date=%s category=%d name=%s", item.Date.Format("2006-01-02"), item.Category, item.Name)
	}

	// 查询聚合后的公司资料打包结果，一次性汇总章节、除权除息与财务信息。
	bundle, err := client.GetCompanyInfo(types.MarketSZ.Uint8(), "000001")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("company_bundle sections=%d xdxr=%d finance_nil=%t",
		len(bundle.Sections), len(bundle.XDXR), bundle.Finance == nil)

	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379", DB: 2})
	cached := cache.NewCacheClient(rdb, client)
	ctx := context.Background()
	flatCache, err := cached.GetFinanceInfo(ctx, types.MarketSZ.Uint8(), "000001")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("finance code=%s updated=%d total_shares=%.2f revenue=%.2f profit=%.2f",
		flatCache.Code, flatCache.UpdatedDate, flatCache.TotalShares, flatCache.OperatingRevenue, flatCache.NetProfit)
	stats := cached.Stats() // 命中/回源/熔断拒绝/空标记命中 各计数
	fmt.Printf("命中=%d 空标记命中=%d 回源=%d 回源失败=%d 熔断拒绝=%d\n",
		stats.Hits, stats.EmptyHits, stats.Fetches, stats.SourceErrors, stats.CircuitRejects)
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
