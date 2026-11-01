// Command stock_market_watch 演示集合竞价、涨幅榜、异动提示与成交分布等市场监控数据的查询与打印。
//
// 运行示例：`go run ./examples/stock_market_watch`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	// 市场 `types.MarketSZ`（深市），代码 `000001`；`start=0` 为起始竞价序号，`count=20` 为本页请求条数；
	// `min(10, len(...))` 防止切片越界，仅打印前 10 条（含已撮合与未撮合笔数）。
	auction, err := client.StockAuction(types.MarketSZ.Uint8(), "000001", 0, 20)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range auction[:min(10, len(auction))] {
		log.Printf("auction time=%s price=%.2f matched=%d unmatched=%d", item.Time, item.Price, item.Matched, item.Unmatched)
	}

	// `category` 取 `types.CategoryA`（A 股分类），`size=5` 为期望返回的榜单条目数量；
	// `min(5, len(...))` 防止切片越界，仅打印前 5 名。
	top, err := client.StockTopBoard(types.CategoryA, 5)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range top.Increase[:min(5, len(top.Increase))] {
		log.Printf("top_increase code=%s price=%.2f value=%.2f", item.Code, item.Price, item.Value)
	}

	// 查询深市异动提示：`start=0` 为起始异动序号，`count=10` 为本页请求条数。
	unusual, err := client.StockUnusual(types.MarketSZ.Uint8(), 0, 10)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range unusual {
		log.Printf("unusual code=%s time=%s desc=%s value=%s type=%d", item.Code, item.Time, item.Desc, item.Value, item.UnusualType)
	}

	// `market` 取 `types.MarketSZ`（深市），代码 `000001`；返回成交分布（量价分布）汇总与盘口快照。
	profile, err := client.StockVolumeProfile(types.MarketSZ.Uint8(), "000001")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("volume_profile code=%s close=%.2f count=%d turnover=%.2f%%", profile.Code, profile.Close, profile.Count, profile.Turnover)
	// 价格档位可能不足 10 个，`min(10, len(...))` 防止切片越界，仅打印前 10 个档位的买卖量分布。
	for _, item := range profile.VolProfiles[:min(10, len(profile.VolProfiles))] {
		log.Printf("profile price=%.2f vol=%d buy=%d sell=%d", item.Price, item.Vol, item.Buy, item.Sell)
	}
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
