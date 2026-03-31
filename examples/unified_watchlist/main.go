package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewUnifiedClient()
	defer client.Disconnect()

	stockItems, err := client.StockQuotesDetail(
		[]uint8{gotdx.MarketSZ, gotdx.MarketSH, gotdx.MarketSZ},
		[]string{"000001", "600000", "300750"},
	)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range stockItems {
		log.Printf("stock code=%s time=%s price=%.2f open=%.2f high=%.2f low=%.2f vol=%d",
			item.Code, item.ServerTime, item.Price, item.Open, item.High, item.Low, item.Vol)
	}

	bars, err := client.StockKLine(gotdx.KLINE_TYPE_DAILY, gotdx.MarketSZ, "000001", 0, 5, 1, gotdx.AdjustNone)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range bars {
		log.Printf("stock_kline time=%s close=%.2f vol=%.0f", item.DateTime, item.Close, item.Vol)
	}

	goodsItems, err := client.GoodsQuotes(
		[]uint8{gotdx.ExCategoryUSStock, gotdx.ExCategoryHKStock},
		[]string{"TSLA", "09988"},
	)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range goodsItems {
		log.Printf("goods code=%s date=%s close=%.2f high=%.2f low=%.2f vol=%d",
			item.Code, item.Date, item.Close, item.High, item.Low, item.Vol)
	}

	samples, err := client.GoodsChartSampling(gotdx.ExCategoryUSStock, "TSLA")
	if err != nil {
		log.Fatalln(err)
	}
	for i, price := range samples[:min(10, len(samples))] {
		log.Printf("goods_sample index=%d price=%.2f", i, price)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
