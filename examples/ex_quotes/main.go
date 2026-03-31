package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	items, err := client.ExQuotes(
		[]uint8{gotdx.ExCategoryUSStock, gotdx.ExCategoryHKMainBoard},
		[]string{"TSLA", "09988"},
	)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("%s date=%s close=%.2f open=%.2f high=%.2f low=%.2f vol=%d",
			item.Code, item.Date, item.Close, item.Open, item.High, item.Low, item.Vol)
	}
}
