package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMACClient()
	defer client.Disconnect()

	reply, err := client.MACQuotes(gotdx.MarketSZ, "000001")
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("mac_quotes code=%s name=%s time=%s close=%.2f pre_close=%.2f turnover=%.2f avg=%.2f chart_points=%d",
		reply.Code, reply.Name, reply.DateTime, reply.Close, reply.PreClose, reply.Turnover, reply.Avg, len(reply.ChartData))

	for _, item := range reply.ChartData[:min(5, len(reply.ChartData))] {
		log.Printf("chart time=%s price=%.2f avg=%.2f vol=%d momentum=%.2f",
			item.Time, item.Price, item.Avg, item.Vol, item.Momentum)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
