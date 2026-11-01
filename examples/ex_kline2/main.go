// Command ex_kline2 演示第二套 K 线协议 ExKLine2 获取扩展市场证券 K 线，以美股 `TSLA` 日线为例打印最近 10 根。
//
// 运行示例：`go run ./examples/ex_kline2`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	// 参数口径与 `ExKLine` 相同：美股 `TSLA` 日线，`start` 为 0，`count` 为 10 根，`times` 为 1；
	// 仅底层使用第二套 K 线协议实现。
	items, err := client.ExKLine2(types.ExCategoryUSStock, "TSLA", types.KLINE_TYPE_DAILY, 0, 10, 1)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("ex_kline2 date=%s open=%.2f high=%.2f low=%.2f close=%.2f vol=%d",
			item.DateTime, item.Open, item.High, item.Low, item.Close, item.Vol)
	}
}
