// Command mac_transactions 演示通过 MAC 协议查询证券逐笔成交明细，并打印时间、价格、数量与买卖方向。
//
// 运行示例：`go run ./examples/mac_transactions`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMACClient()
	defer client.Disconnect()

	// `market` 取 `types.MarketSZ`（深市），代码 `000001`；`start=0` 为起始成交序号（分页偏移，从 0 开始）；
	// `count=20` 为期望成交条数上限，传 0 时由协议使用默认上限；查询最新交易日的逐笔成交明细。
	items, err := client.MACTransactions(types.MarketSZ.Uint8(), "000001", 0, 20)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("mac_transactions time=%s price=%.2f vol=%d trade_count=%d action=%d",
			item.Time, item.Price, item.Vol, item.TradeCount, item.BuyOrSell)
	}
}
