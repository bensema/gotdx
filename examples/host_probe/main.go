package main

import (
	"log"
	"time"

	"github.com/bensema/gotdx"
)

func main() {
	logProbe("main", gotdx.MainHosts(), 5)
	logProbe("ex", gotdx.ExHosts(), 5)
	logProbe("mac", gotdx.MACHosts(), 5)
	logProbe("mac-ex", gotdx.MACExHosts(), 5)

	client := gotdx.New(gotdx.WithAutoSelectFastest(true), gotdx.WithTimeoutSec(6))
	fastest, err := client.FastestHost()
	if err != nil {
		log.Printf("fastest main host unavailable: %v", err)
		return
	}
	log.Printf("fastest main host: %s %s latency=%s", fastest.Name, fastest.Address, fastest.Latency)
}

func logProbe(label string, hosts []gotdx.HostInfo, limit int) {
	results := gotdx.ProbeHosts(hosts, time.Second)
	log.Printf("%s hosts=%d", label, len(hosts))

	for i, result := range results {
		if i >= limit {
			break
		}
		if result.Reachable {
			log.Printf("%s #%d %s %s latency=%s", label, i+1, result.Name, result.Address, result.Latency)
			continue
		}
		log.Printf("%s #%d %s %s error=%s", label, i+1, result.Name, result.Address, result.Error)
	}
}
