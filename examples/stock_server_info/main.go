package main

import (
	"log"
	"strings"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	if _, err := client.Connect(); err != nil {
		log.Fatalln(err)
	}

	heartbeat, err := client.GetServerHeartbeat()
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("heartbeat date=%d", heartbeat.Date)

	info, err := client.GetServerInfo()
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("server_info now=%s region=%d switch=%d info=%q",
		info.TimeNow, info.Region, info.MaybeSwitch, info.Info)

	announcement, err := client.GetAnnouncement()
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("announcement has_content=%t title=%q preview=%q",
		announcement.HasContent, announcement.Title, preview(announcement.Content, 80))

	exchangeAnnouncement, err := client.GetExchangeAnnouncement()
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("exchange_announcement version=%d preview=%q",
		exchangeAnnouncement.Version, preview(exchangeAnnouncement.Content, 80))
}

func preview(text string, limit int) string {
	text = strings.TrimSpace(text)
	if len(text) <= limit {
		return text
	}
	return text[:limit]
}
