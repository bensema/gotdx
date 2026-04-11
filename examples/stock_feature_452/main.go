package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	items, err := client.StockFeature452(0, 10)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("feature_452 market=%d code=%s p1=%.4f p2=%.4f",
			item.Market, item.Code, item.P1, item.P2)
	}
}
