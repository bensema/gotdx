// Command ex_quotes 演示通过 ExQuotes 批量查询扩展市场行情报价，同时取美股 `TSLA` 与港股主板 `09988` 两只标的。
//
// 运行示例：`go run ./examples/ex_quotes`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	// 类别序列与代码序列按下标一一配对：第 1 项美股 `TSLA`，第 2 项港股主板 `09988`；两序列长度必须一致。
	items, err := client.ExQuotes(
		[]uint8{types.ExCategoryUSStock, types.ExCategoryHKMainBoard},
		[]string{"TSLA", "09988"},
	)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("%s date=%s close=%.2f open=%.2f high=%.2f low=%.2f vol=%d",
			item.Code, item.Date, item.Close, item.Open, item.High, item.Low, item.Vol)
	}
}
