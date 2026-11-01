// Command stock_kline 演示通过统一接口 StockKLine 查询沪深市场 K 线并逐根打印字段。
//
// 运行示例：`go run ./examples/stock_kline`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	// `category` 取 `types.KLINE_TYPE_SECONDS`（多秒 K 线，编号 13；下方注释掉的 `types.KLINE_TYPE_1MIN` 为备选周期）；
	// `market` 取 `types.MarketSZ`（深市），代码 `000001`（平安银行）；`start=0` 为起始偏移条数，`count=10`
	// 为请求条数；`times=1` 为周期倍数；`adjust` 取 `types.AdjustNone` 表示不复权。
	bars, err := client.StockKLine(
		types.KLINE_TYPE_SECONDS,
		// types.KLINE_TYPE_1MIN,
		types.MarketSZ.Uint8(),
		"000001",
		0,
		10,
		1,
		types.AdjustNone,
	)
	if err != nil {
		log.Fatalln(err)
	}

	// 逐根打印 K 线的时间、OHLC、涨跌与成交量字段。
	for _, bar := range bars {
		log.Printf("%s preClose=%.3f open=%.3f high=%.3f low=%.3f close=%.3f rate=%.2f  price=%.2f vol=%.0f amount=%.0f turnover=%.2f%%",
			bar.DateTime, bar.PreClose, bar.Open, bar.High, bar.Low, bar.Close, bar.RiseRate, bar.RisePrice, bar.Vol, bar.Amount, bar.Turnover)
	}
}
