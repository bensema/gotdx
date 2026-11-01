// Command mac_board_members 演示 MACBoardMembersWithSort 获取板块 `880761` 的成分股列表，
// 按指定排序类型与顺序取前 20 只并打印代码、市场与名称。
//
// 运行示例：`go run ./examples/mac_board_members`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMACClient()
	defer client.Disconnect()

	// 参数 `boardSymbol` 为板块 `880761`，`count` 为成分股上限 20；`sortType` 传 14 即
	// `MACSortChangePct`（按涨跌幅排序），`sortOrder` 传 1 即 `MACSortOrderDesc`（降序）；示例保持
	// 原始数字透传演示。
	items, err := client.MACBoardMembersWithSort("880761", 20, 14, 1)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("mac_board_member sort_type=%d sort_order=%d symbol=%s market=%d name=%s",
			14, 1, item.Symbol, item.Market, item.Name)
	}
}
