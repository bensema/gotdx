package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	hello, err := client.Connect()
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("connected info=%s", hello.Info)

	count, err := client.GetSecurityCount(gotdx.MarketSZ)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("security_count=%d", count.Count)

	list, err := client.GetSecurityListRange(gotdx.MarketSZ, 0, 200)
	if err != nil {
		log.Fatalln(err)
	}
	if len(list.List) > 0 {
		log.Printf("security_list first=%s/%s count=%d", list.List[0].Code, list.List[0].Name, len(list.List))
	}

	quotes, err := client.GetSecurityQuotes([]uint8{gotdx.MarketSZ, gotdx.MarketSH}, []string{"000001", "600000"})
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range quotes.List {
		log.Printf("security_quote code=%s price=%.2f open=%.2f high=%.2f low=%.2f", item.Code, item.Price, item.Open, item.High, item.Low)
	}

	minute, err := client.GetMinuteTimeData(gotdx.MarketSZ, "000001")
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range minute.List[:min(10, len(minute.List))] {
		log.Printf("minute price=%.2f avg=%.4f vol=%d", item.Price, item.Avg, item.Vol)
	}

	trans, err := client.GetTransactionData(gotdx.MarketSZ, "000001", 0, 10)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range trans.List {
		log.Printf("transaction time=%s price=%.2f vol=%d num=%d action=%d", item.Time, item.Price, item.Vol, item.Num, item.BuyOrSell)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
