package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	count, err := client.ExCount()
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("ex_count=%d", count)
}
