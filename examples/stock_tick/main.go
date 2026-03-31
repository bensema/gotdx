package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	items, err := client.StockTickChart(gotdx.MarketSZ, "000001", 0, 60)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("price=%.2f avg=%.4f vol=%d", item.Price, item.Avg, item.Vol)
	}
}
