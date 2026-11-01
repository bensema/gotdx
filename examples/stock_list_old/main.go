// Command stock_list_old 演示使用旧版证券列表协议查询深市证券清单，并打印代码、名称、昨收价与成交量单位。
//
// 运行示例：`go run ./examples/stock_list_old`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	// `market` 取 `types.MarketSZ`（深市）；`start=0` 为起始偏移序号（从 0 开始），旧版协议按该偏移返回本页列表。
	items, err := client.StockListOld(types.MarketSZ.Uint8(), 0)
	if err != nil {
		log.Fatalln(err)
	}

	// 列表可能不足 10 条，`exampleutil.Min(10, len(...))` 防止切片越界，仅打印前 10 条。
	for _, item := range items[:exampleutil.Min(10, len(items))] {
		log.Printf("old_list code=%s name=%s pre_close=%.2f vol_unit=%d",
			item.Code, item.Name, item.PreClose, item.VolUnit)
	}
}
