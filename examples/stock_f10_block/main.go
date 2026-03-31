package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	f10, err := client.StockF10(gotdx.MarketSZ, "000001")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("sections=%d xdxr=%d finance_nil=%v", len(f10.Sections), len(f10.XDXR), f10.Finance == nil)

	blocks, err := client.StockBlock(gotdx.BlockFileGN)
	if err != nil {
		log.Fatalln(err)
	}

	limit := 5
	if len(blocks) < limit {
		limit = len(blocks)
	}
	for _, item := range blocks[:limit] {
		log.Printf("block=%s type=%d code=%s", item.BlockName, item.BlockType, item.Code)
	}
}
