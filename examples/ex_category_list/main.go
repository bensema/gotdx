// Command ex_category_list 演示通过 ExCategoryList 拉取扩展市场分类（港股、美股、期货等）列表并逐行打印。
//
// 运行示例：`go run ./examples/ex_category_list`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	items, err := client.ExCategoryList()
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("market=%d code=%d name=%s abbr=%s", item.Market, item.Code, item.Name, item.Abbr)
	}
}
