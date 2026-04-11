package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	items, err := client.StockQuotesEncrypt(
		[]uint8{gotdx.MarketSZ, gotdx.MarketSH},
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
