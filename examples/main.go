package main

import (
	"github.com/bensema/gotdx"
	"log"
)

func main() {
	// ip地址如果失效，请自行替换
	tdx := gotdx.New(gotdx.WithTCPAddress("124.71.187.122:7709"))
	_, err := tdx.Connect()
	if err != nil {
		log.Fatalln(err)
	}
	defer tdx.Disconnect()

	reply, err := tdx.GetSecurityQuotes([]uint8{gotdx.MarketSh, gotdx.MarketSz}, []string{"000001", "600008"})
	if err != nil {
		log.Println(err)
	}

	for _, obj := range reply.List {
		log.Printf("%+v", obj)
	}
}
