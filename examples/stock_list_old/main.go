package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	items, err := client.StockListOld(gotdx.MarketSZ, 0)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items[:exampleutil.Min(10, len(items))] {
		log.Printf("old_list code=%s name=%s pre_close=%.2f vol_unit=%d",
			item.Code, item.Name, item.PreClose, item.VolUnit)
	}
}
