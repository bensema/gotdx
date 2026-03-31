package main

import (
	"log"
	"strings"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	content, err := client.ExTable()
	if err != nil {
		log.Fatalln(err)
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines[:min(10, len(lines))] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		log.Println(line)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
