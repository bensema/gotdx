package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

const (
	pageSize  = 800
	maxPages  = 3
	startPage = 0
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	total, err := client.StockCount(gotdx.MarketSZ)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("market=SZ total=%d page_size=%d max_pages=%d", total, pageSize, maxPages)

	var fetched int
	for page := startPage; page < maxPages; page++ {
		start := uint32(page * pageSize)
		items, err := client.StockList(gotdx.MarketSZ, start, pageSize)
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
