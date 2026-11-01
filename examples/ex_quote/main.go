// Command ex_quote 演示通过 ExQuote 查询单只扩展证券的行情报价，以美股 `TSLA` 为例打印开高低收与均价等字段。
//
// 运行示例：`go run ./examples/ex_quote`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	// 参数 `category` 为美股类别，`code` 为标的代码 `TSLA`，查询单只标的最新行情报价。
	item, err := client.ExQuote(types.ExCategoryUSStock, "TSLA")
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("code=%s date=%s close=%.2f open=%.2f high=%.2f low=%.2f vol=%d avg=%.2f",
		item.Code, item.Date, item.Close, item.Open, item.High, item.Low, item.Vol, item.Avg)
}
