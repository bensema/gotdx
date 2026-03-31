package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	tickItems, err := client.StockHistoryTickChart(20260316, gotdx.MarketSZ, "000001")
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range tickItems[:min(10, len(tickItems))] {
		log.Printf("history_tick price=%.2f avg=%.4f vol=%d", item.Price, item.Avg, item.Vol)
	}

	transItems, err := client.StockHistoryTransaction(20260316, gotdx.MarketSZ, "000001", 0, 20)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range transItems {
		log.Printf("history_trans time=%s price=%.2f vol=%d action=%d", item.Time, item.Price, item.Vol, item.BuyOrSell)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
