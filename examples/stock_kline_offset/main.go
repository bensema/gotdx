// Command stock_kline_offset 演示通过偏移分页接口 `StockKLineOffset` 查询日 K 线，并逐根打印开高低收、成交量与换手率。
//
// 运行示例：`go run ./examples/stock_kline_offset`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	// `category` 取 `types.KLINE_TYPE_DAILY`（日 K）；`market` 取 `types.MarketSZ`（深市），代码 `000001`；
	// `start=0` 为起始偏移条数，`count=10` 为请求条数；`times=1` 为周期倍数；`adjust` 取 `types.AdjustNone` 表示不复权。
	items, err := client.StockKLineOffset(types.KLINE_TYPE_DAILY, types.MarketSZ.Uint8(), "000001", 0, 10, 1, types.AdjustNone)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("kline_offset date=%s open=%.2f high=%.2f low=%.2f close=%.2f vol=%.2f turnover=%.2f%%",
			item.DateTime, item.Open, item.High, item.Low, item.Close, item.Vol, item.Turnover)
	}
}
