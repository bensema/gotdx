// Command stock_transaction 演示查询证券当日逐笔成交明细，并打印时间、价格、笔数与买卖方向。
//
// 运行示例：`go run ./examples/stock_transaction`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	// `market` 取 `types.MarketSZ`（深市），代码 `000001`；`start=0` 为起始成交序号（从 0 开始），
	// `count=20` 为本页请求笔数，返回当日逐笔成交明细。
	items, err := client.StockTransaction(types.MarketSZ.Uint8(), "000001", 0, 20)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("time=%s price=%.2f vol=%d num=%d action=%d",
			item.Time, item.Price, item.Vol, item.Num, item.BuyOrSell)
	}
}
