package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

const (
	pageSize = 1000
	maxPages = 3
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	total, err := client.GoodsCount()
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("goods_total=%d page_size=%d max_pages=%d", total, pageSize, maxPages)

	var fetched int
	for page := 0; page < maxPages; page++ {
		start := uint32(page * pageSize)
		items, err := client.GoodsList(start, pageSize)
		if err != nil {
			log.Fatalf("page=%d start=%d failed: %v", page, start, err)
		}
		if len(items) == 0 {
			break
		}

		first := items[0]
		last := items[len(items)-1]
		fetched += len(items)
		log.Printf("page=%d fetched=%d first=%s/%s last=%s/%s",
			page, len(items), first.Code, first.Name, last.Code, last.Name)
	}

	log.Printf("paged_fetch_done fetched=%d of total=%d", fetched, total)
}
