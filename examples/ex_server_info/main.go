package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	hosts := exampleutil.ExHosts()
	client := gotdx.NewEx(
		gotdx.WithExTCPAddress(hosts[0]),
		gotdx.WithExTCPAddressPool(hosts[1:]...),
		gotdx.WithTimeoutSec(6),
	)
	defer client.Disconnect()

	login, err := client.ConnectEx()
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("login time=%s server=%s ip=%s", login.DateTime, login.ServerName, login.IP)

	info, err := client.GetExServerInfo()
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("server_info name=%s version=%s delay=%d now=%s", info.ServerName, info.Version, info.Delay, info.TimeNow)

	tick, err := client.ExGetTickChart(gotdx.ExCategoryUSStock, "TSLA")
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range tick.List[:min(10, len(tick.List))] {
		log.Printf("ex_tick time=%s price=%.2f avg=%.2f vol=%d", item.Time, item.Price, item.Avg, item.Vol)
	}

	historyTick, err := client.ExGetHistoryTickChart(20260330, gotdx.ExCategoryUSStock, "TSLA")
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range historyTick.List[:min(10, len(historyTick.List))] {
		log.Printf("ex_history_tick time=%s price=%.2f avg=%.2f vol=%d", item.Time, item.Price, item.Avg, item.Vol)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
