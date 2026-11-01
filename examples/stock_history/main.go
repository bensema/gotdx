// Command stock_history 演示查询指定历史日期的历史分时图与历史逐笔成交两类数据，并分别打印字段。
//
// 运行示例：`go run ./examples/stock_history`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	// 日期 `20260316` 为目标交易日，按 `YYYYMMDD` 打包成整数；市场 `types.MarketSZ`（深市），代码
	// `000001`。返回该日历史分时条目；`min(10, len(...))` 防止切片越界，仅打印前 10 个时点。
	tickItems, err := client.StockHistoryTickChart(20260316, types.MarketSZ.Uint8(), "000001")
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range tickItems[:min(10, len(tickItems))] {
		log.Printf("history_tick price=%.2f avg=%.4f vol=%d", item.Price, item.Avg, item.Vol)
	}

	// 查询同一日期、同一证券的历史逐笔成交：`start=0` 为起始成交序号（从 0 开始），`count=20` 为本页请求笔数。
	transItems, err := client.StockHistoryTransaction(20260316, types.MarketSZ.Uint8(), "000001", 0, 20)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range transItems {
		log.Printf("history_trans time=%s price=%.2f vol=%d action=%d", item.Time, item.Price, item.Vol, item.BuyOrSell)
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
