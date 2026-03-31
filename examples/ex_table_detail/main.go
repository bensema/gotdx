package main

import (
	"log"
	"strings"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	content, err := client.ExTableDetail()
	if err != nil {
		log.Fatalln(err)
	}

	items := strings.Split(content, ",")
	for _, item := range items[:min(20, len(items))] {
		if item == "" {
			continue
		}
		log.Println(item)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
