package main

import (
	"fmt"
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	info, err := client.StockIndexInfo(types.MarketSH.Uint8(), "000001")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("index_info code=%s time=%s close=%.2f open=%.2f high=%.2f low=%.2f up=%d down=%d",
		info.Code, info.ServerTime, info.Close, info.Open, info.High, info.Low, info.UpCount, info.DownCount)

	momentum, err := client.StockIndexMomentum(types.MarketSH.Uint8(), "000001")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("index_momentum count=%d last=%d", len(momentum), momentum[len(momentum)-1])

	bars, err := client.GetIndexBars(types.KLINE_TYPE_DAILY, types.MarketSH.Uint8(), "000001", 0, 1)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range bars.List {
		log.Printf("index_bar time=%s close=%.2f high=%.2f low=%.2f", item.DateTime, item.Close, item.High, item.Low)
	}

	sampling, err := getSampling()
	if err != nil {
		log.Printf("sampling unavailable: %v", err)
	} else {
		for i, price := range sampling[:min(10, len(sampling))] {
			log.Printf("sampling index=%d price=%.2f", i, price)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getSampling() ([]float64, error) {
	var lastErr error
	for _, host := range exampleutil.MainHosts() {
		client := exampleutil.NewMainClientForHost(host)
		reply, err := client.StockChartSampling(types.MarketSZ.Uint8(), "000001")
		_ = client.Disconnect()
		if err == nil {
			return reply, nil
		}
		lastErr = fmt.Errorf("%s sampling failed: %w", host, err)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no main quote hosts configured")
	}
	return nil, lastErr
}
