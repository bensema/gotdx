package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	items, err := client.GoodsTickChart(gotdx.ExCategoryUSStock, "TSLA", 0)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items[:min(20, len(items))] {
		log.Printf("time=%s price=%.2f avg=%.2f vol=%d", item.Time, item.Price, item.Avg, item.Vol)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
