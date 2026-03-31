package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	items, err := client.ExCategoryList()
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("market=%d code=%d name=%s abbr=%s", item.Market, item.Code, item.Name, item.Abbr)
	}
}
