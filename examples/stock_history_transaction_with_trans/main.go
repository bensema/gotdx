package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	items, err := client.StockHistoryTransactionWithTrans(20260410, gotdx.MarketSZ, "000001", 0, 20)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("history_trans_with_dir time=%s price=%.2f vol=%d num=%d action=%s",
			item.Time, item.Price, item.Vol, item.Num, item.Action)
	}
}
