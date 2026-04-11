package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMACClient()
	defer client.Disconnect()

	items, err := client.MACBoardMembers("880761", 20)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("mac_board_member symbol=%s market=%d name=%s",
			item.Symbol, item.Market, item.Name)
	}
}
