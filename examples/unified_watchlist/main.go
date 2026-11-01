// Command unified_watchlist 演示统一客户端在同一会话内同时查询 A 股行情、A 股 K 线、港股与美股报价以及扩展市场分时采样。
//
// 运行示例：`go run ./examples/unified_watchlist`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewUnifiedClient()
	defer client.Disconnect()

	// 市场序列与代码序列按下标一一对应：深市 `000001`、沪市 `600000`、深市 `300750`，批量取回详细盘口行情。
	stockItems, err := client.StockQuotesDetail(
		[]uint8{types.MarketSZ.Uint8(), types.MarketSH.Uint8(), types.MarketSZ.Uint8()},
		[]string{"000001", "600000", "300750"},
	)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range stockItems {
		log.Printf("stock code=%s time=%s price=%.2f open=%.2f high=%.2f low=%.2f vol=%d turnover=%.2f%%",
			item.Code, item.ServerTime, item.Price, item.Open, item.High, item.Low, item.Vol, item.Turnover)
	}

	// `category` 取 `types.KLINE_TYPE_DAILY`（日 K）；`start=0` 为起始偏移条数，`count=5` 为请求条数；
	// `times=1` 为周期倍数；`adjust` 取 `types.AdjustNone` 表示不复权。
	bars, err := client.StockKLine(types.KLINE_TYPE_DAILY, types.MarketSZ.Uint8(), "000001", 0, 5, 1, types.AdjustNone)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range bars {
		log.Printf("stock_kline time=%s close=%.2f vol=%.0f turnover=%.2f%%", item.DateTime, item.Close, item.Vol, item.Turnover)
	}

	// 类别序列与代码序列按下标一一对应：`ExCategoryUSStock`（美股）对应 `TSLA`，`ExCategoryHKStock`（港股）对应 `09988`。
	exItems, err := client.ExQuotes(
		[]uint8{types.ExCategoryUSStock, types.ExCategoryHKStock},
		[]string{"TSLA", "09988"},
	)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range exItems {
		log.Printf("ex code=%s date=%s close=%.2f high=%.2f low=%.2f vol=%d",
			item.Code, item.Date, item.Close, item.High, item.Low, item.Vol)
	}

	// 查询美股 `TSLA` 的分时缩略采样价格序列；`min(10, len(...))` 防止切片越界，仅打印前 10 个价格点。
	samples, err := client.ExChartSampling(types.ExCategoryUSStock, "TSLA")
	if err != nil {
		log.Fatalln(err)
	}
	for i, price := range samples[:min(10, len(samples))] {
		log.Printf("ex_sample index=%d price=%.2f", i, price)
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
