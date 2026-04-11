package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMACClient()
	defer client.Disconnect()

	items, err := client.MACBoardList(gotdx.BoardTypeHY, 20)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("mac_board code=%s name=%s price=%.2f rise_speed=%.2f symbol=%s/%s",
			item.Code, item.Name, item.Price, item.RiseSpeed, item.SymbolCode, item.SymbolName)
	}
}
