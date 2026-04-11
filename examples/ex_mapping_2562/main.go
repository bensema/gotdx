package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	items, err := client.ExMapping2562(uint16(gotdx.ExCategoryCFFEXFutures), 0, 10)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("ex_mapping_2562 category=%d name=%s index=%d switch=%d codes=[%.2f %.2f %.2f %d %d]",
			item.Category, item.Name, item.Index, item.Switch, item.Code1, item.Code2, item.Code3, item.Code4, item.Code5)
	}
}
