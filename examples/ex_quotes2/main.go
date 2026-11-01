// Command ex_quotes2 演示第二套批量报价协议 ExQuotes2，同时取美股 `TSLA` 与港股主板 `09988`，打印结算价与涨速字段。
//
// 运行示例：`go run ./examples/ex_quotes2`
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
	items, err := client.ExQuotes2(
		[]uint8{types.ExCategoryUSStock, types.ExCategoryHKMainBoard},
		[]string{"TSLA", "09988"},
	)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("code=%s date=%s close=%.2f settlement=%.2f raise_speed=%.2f",
			item.Code, item.Date, item.Close, item.Settlement, item.RaiseSpeed)
	}
}
