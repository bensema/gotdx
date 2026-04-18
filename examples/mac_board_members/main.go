package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMACClient()
	defer client.Disconnect()

	items, err := client.MACBoardMembersWithSort("880761", 20, 14, 1)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("mac_board_member sort_type=%d sort_order=%d symbol=%s market=%d name=%s",
			14, 1, item.Symbol, item.Market, item.Name)
	}
}
