// Command mac_auction 演示通过 MACAuction 获取 MAC 竞价数据，以深市 `000001` 为例打印竞价时间、价格与匹配/未匹配量。
//
// 运行示例：`go run ./examples/mac_auction`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMACClient()
	defer client.Disconnect()

	// 参数 `market` 为深市，`code` 为 `000001`；`start` 为起始竞价记录序号 0；`count` 为期望竞价
	// 条数上限 100，接口内部按固定单页大小自动翻页。
	items, err := client.MACAuction(types.MarketSZ.Uint8(), "000001", 0, 100)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("mac_auction time=%s price=%.2f matched=%d unmatched=%d flag=%d",
			item.Time, item.Price, item.Matched, item.Unmatched, item.Flag)
	}
}
