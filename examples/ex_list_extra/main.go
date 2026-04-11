package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	items, err := client.ExListExtra(0, 0, 10)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("ex_list_extra category=%d code=%s flag=%d values=%v",
			item.Category, item.Code, item.Flag, item.Values)
	}
}
