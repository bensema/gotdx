package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	item, err := client.ExQuote(gotdx.ExCategoryUSStock, "TSLA")
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("code=%s date=%s close=%.2f open=%.2f high=%.2f low=%.2f vol=%d avg=%.2f",
		item.Code, item.Date, item.Close, item.Open, item.High, item.Low, item.Vol, item.Avg)
}
