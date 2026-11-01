// Command ex_kline 演示通过 ExKLine 获取扩展市场证券 K 线，以美股 `TSLA` 日线为例打印最近 10 根 K 线。
//
// 运行示例：`go run ./examples/ex_kline`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	// 参数 `category` 为美股类别，`code` 为 `TSLA`；`period` 为日线周期；`start` 为 0 表示从最新一根
	// 向前取数；`count` 为请求 10 根 K 线；`times` 为周期倍数，取 1 表示不放大周期。
	items, err := client.ExKLine(types.ExCategoryUSStock, "TSLA", types.KLINE_TYPE_DAILY, 0, 10, 1)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("%s open=%.2f high=%.2f low=%.2f close=%.2f vol=%d",
			item.DateTime, item.Open, item.High, item.Low, item.Close, item.Vol)
	}
}
