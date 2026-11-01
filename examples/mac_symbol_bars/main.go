// Command mac_symbol_bars 演示通过 MAC 协议接口 `MACSymbolBars` 查询证券 K 线并逐根打印行情字段。
//
// 运行示例：`go run ./examples/mac_symbol_bars`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMACClient()
	defer client.Disconnect()

	// `market` 取 `types.MarketSZ`（深市），代码 `000001`；`period` 取 `types.KLINE_TYPE_DAILY`（日 K）；
	// `times=1` 为周期倍数（传 0 时按 1 处理）；`start=0` 为起始偏移条数；`count=10` 为请求条数；
	// `adjust` 取 `types.AdjustNone` 表示不复权。
	items, err := client.MACSymbolBars(types.MarketSZ.Uint8(), "000001", types.KLINE_TYPE_DAILY, 1, 0, 10, types.AdjustNone)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("mac_symbol_bars date=%s open=%.2f high=%.2f low=%.2f close=%.2f vol=%.2f",
			item.DateTime, item.Open, item.High, item.Low, item.Close, item.Vol)
	}
}
