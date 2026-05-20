package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	bars, err := client.StockKLine(
		types.KLINE_TYPE_DAILY,
		types.MarketSZ.Uint8(),
		"000001",
		5400,
		// 400,
		1,
		1,
		types.AdjustNone,
	)
	if err != nil {
		log.Fatalln(err)
	}

	for _, bar := range bars {
		log.Printf("%s last=%.3f open=%.3f high=%.3f low=%.3f close=%.3f rate=%.2f  price=%.2f vol=%.0f amount=%.0f turnover=%.2f%%",
			bar.DateTime, bar.Last, bar.Open, bar.High, bar.Low, bar.Close, bar.RiseRate, bar.RisePrice, bar.Vol, bar.Amount, bar.Turnover)
	}
}
