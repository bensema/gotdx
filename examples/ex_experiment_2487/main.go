package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	reply, err := client.ExExperiment2487(gotdx.ExCategoryUSStock, "TSLA")
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("ex_2487 code=%s active=%d close=%.2f price=%.2f vol=%d cur_vol=%d amount=%.2f tail=%s",
		reply.Code, reply.Active, reply.Close, reply.Price, reply.Vol, reply.CurVol, reply.Amount, exampleutil.PreviewHex(reply.TailHex, 80))
}
