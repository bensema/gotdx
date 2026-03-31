package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	items, err := client.ExQuotes2(
		[]uint8{gotdx.ExCategoryUSStock, gotdx.ExCategoryHKMainBoard},
		[]string{"TSLA", "09988"},
	)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("code=%s date=%s close=%.2f settlement=%.2f raise_speed=%.2f",
			item.Code, item.Date, item.Close, item.Settlement, item.RaiseSpeed)
	}
}
