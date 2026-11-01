// Command stock_batch_quotes 演示批量查询多只沪深证券的行情快照，并打印价格、涨跌额、成交量、涨速与换手率。
//
// 运行示例：`go run ./examples/stock_batch_quotes`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	// 市场序列与代码序列按下标一一对应：深市 `000001`、沪市 `600000`、深市 `300750`；返回快照行情条目，
	// 包装层会先按小数位换算价格、再按流通股本补齐换手率。
	items, err := client.StockQuotes(
		[]uint8{types.MarketSZ.Uint8(), types.MarketSH.Uint8(), types.MarketSZ.Uint8()},
		[]string{"000001", "600000", "300750"},
	)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("market=%d code=%s price=%.2f change=%.2f vol=%d rise_speed=%.2f turnover=%.2f%%",
			item.Market, item.Code, item.Price, item.Price-item.PreClose, item.Vol, item.RiseSpeed, item.Turnover)
	}
}
