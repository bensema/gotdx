package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	prices, err := client.ExChartSampling(gotdx.ExCategoryUSStock, "TSLA")
	if err != nil {
		log.Fatalln(err)
	}

	for i, price := range prices[:min(20, len(prices))] {
		log.Printf("sample=%d price=%.2f", i, price)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
