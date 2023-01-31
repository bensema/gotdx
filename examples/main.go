package main

import (
	"github.com/bensema/gotdx"
	"log"
)

func main() {
	var opt = &gotdx.Opt{
		Host: "119.147.212.81",
		Port: 7709,
	}
	api := gotdx.NewClient(opt)
	connectReply, err := api.Connect()
	if err != nil {
		log.Println(err)
	}
	log.Println(connectReply.Info)

	reply, err := api.GetSecurityQuotes([]uint8{gotdx.MarketSh, gotdx.MarketSz}, []string{"000001", "600008"})
	if err != nil {
		log.Println(err)
	}

	for _, obj := range reply.List {
		log.Printf("%+v", obj)
	}

	_ = api.Disconnect()

}
