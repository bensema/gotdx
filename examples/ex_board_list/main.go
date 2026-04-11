package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	items, err := client.ExBoardList(gotdx.BoardTypeHY, 0, 20)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("ex_board code=%s name=%s price=%.2f rise_speed=%.2f symbol=%s/%s",
			item.Code, item.Name, item.Price, item.RiseSpeed, item.SymbolCode, item.SymbolName)
	}
}
