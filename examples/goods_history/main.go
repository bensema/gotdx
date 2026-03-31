package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	items, err := client.GoodsHistoryTransaction(20260330, gotdx.ExCategoryUSStock, "TSLA")
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items[:min(20, len(items))] {
		log.Printf("time=%s price=%d vol=%d action=%s", item.Time, item.Price, item.Vol, item.Action)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
