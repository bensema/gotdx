// Command stock_tick 演示查询证券当日分时图序列，并打印每个时点的价格、均价与累计成交量。
//
// 运行示例：`go run ./examples/stock_tick`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	// `market` 取 `types.MarketSZ`（深市），代码 `000001`；`start=0` 为起始分时点序号（从 0 开始），
	// `count=60` 为请求的分时点数量，返回当日分时序列。
	items, err := client.StockTickChart(types.MarketSZ.Uint8(), "000001", 0, 60)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("price=%.2f avg=%.4f vol=%d", item.Price, item.Avg, item.Vol)
	}
}
