package main

import (
	"fmt"
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()
	i := 0
	for _, market := range []struct {
		market types.Market
	}{
		{market: types.MarketSZ},
		{market: types.MarketSH},
		{market: types.MarketBJ},
	} {

		if items, err := client.StockAll(market.market.Uint8()); err == nil {
			for _, item := range items {
				code := fmt.Sprintf("%s.%s", item.Code, market.market.String())
				if types.IsStock(code) {
					log.Println(code, item.Name)
					i += 1
				}
			}
		}

	}
	fmt.Println(i)
}
