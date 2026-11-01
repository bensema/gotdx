// Command stock_quotes_encrypt 演示加密行情快照协议批量查询沪深证券报价，并打印最新价、开高低收与买卖盘档位数量。
//
// 运行示例：`go run ./examples/stock_quotes_encrypt`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	// 市场序列与代码序列按下标一一对应：深市 `000001`、沪市 `600000`；返回加密协议编码的报价条目。
	items, err := client.StockQuotesEncrypt(
		[]uint8{types.MarketSZ.Uint8(), types.MarketSH.Uint8()},
		[]string{"000001", "600000"},
	)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("quotes_encrypt code=%s time=%s close=%.2f open=%.2f high=%.2f low=%.2f bids=%d asks=%d",
			item.Code, item.Time, item.Close, item.Open, item.High, item.Low, len(item.BidLevels), len(item.AskLevels))
	}
}
