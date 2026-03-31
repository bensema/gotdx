package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	if _, err := client.Connect(); err != nil {
		log.Fatalln(err)
	}

	auction, err := client.GetAuction(gotdx.MarketSZ, "000001", 0, 20)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range auction.List[:min(10, len(auction.List))] {
		log.Printf("auction time=%s price=%.2f matched=%d unmatched=%d", item.Time, item.Price, item.Matched, item.Unmatched)
	}

	top, err := client.GetTopBoard(gotdx.CategoryA, 5)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range top.Increase[:min(5, len(top.Increase))] {
		log.Printf("top_increase code=%s price=%.2f value=%.2f", item.Code, item.Price, item.Value)
	}

	unusual, err := client.GetUnusual(gotdx.MarketSZ, 0, 10)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range unusual.List {
		log.Printf("unusual code=%s time=%s desc=%s value=%s", item.Code, item.Time, item.Desc, item.Value)
	}

	profile, err := client.GetVolumeProfile(gotdx.MarketSZ, "000001")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("volume_profile code=%s close=%.2f count=%d", profile.Code, profile.Close, profile.Count)
	for _, item := range profile.VolProfiles[:min(10, len(profile.VolProfiles))] {
		log.Printf("profile price=%.2f vol=%d buy=%d sell=%d", item.Price, item.Vol, item.Buy, item.Sell)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
