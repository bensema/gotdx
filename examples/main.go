package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewUnifiedClient()
	defer client.Disconnect()

	reply, err := client.StockQuotesDetail([]uint8{gotdx.MarketSZ, gotdx.MarketSH}, []string{"000001", "600008"})
	if err != nil {
		log.Fatalln(err)
	}

	for _, obj := range reply {
		log.Printf("%+v", obj)
	}

	goods, err := client.GoodsQuotes([]uint8{gotdx.ExCategoryUSStock}, []string{"TSLA"})
	if err != nil {
		log.Fatalln(err)
	}

	for _, obj := range goods {
		log.Printf("%+v", obj)
	}
}
