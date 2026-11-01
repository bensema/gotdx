// Command stock_quotes 演示通过统一接口 `StockQuotesDetail` 批量查询沪深证券主行情快照，并打印时间、价格、成交量与换手率。
//
// 运行示例：`go run ./examples/stock_quotes`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	// 市场序列与代码序列按下标一一对应：深市 `000001`、沪市 `600000`；返回带五档买卖盘的详细行情快照。
	items, err := client.StockQuotesDetail(
		[]uint8{types.MarketSZ.Uint8(), types.MarketSH.Uint8()},
		[]string{"000001", "600000"},
	)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("%s %s price=%.2f open=%.2f high=%.2f low=%.2f vol=%d turnover=%.2f%%",
			item.Code, item.ServerTime, item.Price, item.Open, item.High, item.Low, item.Vol, item.Turnover)
	}
}
