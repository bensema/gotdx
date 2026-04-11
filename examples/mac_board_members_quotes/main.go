package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMACClient()
	defer client.Disconnect()

	items, err := client.MACBoardMembersQuotes("880761", 20)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("mac_member_quote symbol=%s name=%s close=%.2f pre_close=%.2f turnover=%.2f pe_static=%.2f",
			item.Symbol, item.Name, item.Close, item.PreClose, item.TurnoverRate, item.PEStatic)
	}
}
