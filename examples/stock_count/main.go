package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	for _, market := range []struct {
		name   string
		market uint8
	}{
		{name: "SZ", market: gotdx.MarketSZ},
		{name: "SH", market: gotdx.MarketSH},
		{name: "BJ", market: gotdx.MarketBJ},
	} {
		count, err := client.StockCount(market.market)
		if err != nil {
			log.Fatalf("%s count failed: %v", market.name, err)
		}
		log.Printf("market=%s count=%d", market.name, count)
	}
}
