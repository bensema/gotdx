// Command ex_quotes_list 演示通过 ExQuotesList 分页获取扩展市场排序行情列表，美股类别下按代码排序取前 20 条。
//
// 运行示例：`go run ./examples/ex_quotes_list`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	// 参数 `category` 为美股类别；`start` 为起始偏移 0；`count` 为本页 20 条；`sortType` 按代码排序；
	// `reverse` 为 `false` 表示升序排列。
	items, err := client.ExQuotesList(types.ExCategoryUSStock, 0, 20, types.SortCode, false)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("code=%s close=%.2f open=%.2f high=%.2f low=%.2f vol=%d",
			item.Code, item.Close, item.Open, item.High, item.Low, item.Vol)
	}
}
