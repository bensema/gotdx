// Command mac_symbol_belong_board 演示查询证券所属板块列表，并打印板块类型、代码、名称与价格。
//
// 运行示例：`go run ./examples/mac_symbol_belong_board`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMACClient()
	defer client.Disconnect()

	// 注意该接口入参顺序为代码在前、市场在后：`symbol=000001`，`market` 取 `types.MarketSZ`（深市），返回所属板块列表。
	items, err := client.MACSymbolBelongBoard("000001", types.MarketSZ.Uint8())
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("mac_belong_board type=%s code=%s name=%s price=%.2f pre_close=%.2f",
			item.BoardType, item.BoardCode, item.BoardName, item.Price, item.PreClose)
	}
}
