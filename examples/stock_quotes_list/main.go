package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	items, err := client.StockQuotesList(
		gotdx.CategoryA,
		0,
		20,
		gotdx.SortTotalAmount,
		true,
		0,
	)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("code=%s price=%.2f change=%.2f amount=%.0f rise_speed=%.2f turnover=%.2f%%",
			item.Code, item.Price, item.Price-item.PreClose, item.Amount, item.RiseSpeed, item.Turnover)
	}
}
