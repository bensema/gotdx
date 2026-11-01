// Command mac_board_members_quotes 演示 MACBoardMembersQuotesWithSort 获取板块 `880761` 的成分股报价，
// 按指定排序类型与顺序取前 20 只并打印行情字段。
//
// 运行示例：`go run ./examples/mac_board_members_quotes`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMACClient()
	defer client.Disconnect()

	// 参数 `boardSymbol` 为板块 `880761`，`count` 为报价条目上限 20；`sortType` 传 14 即
	// `MACSortChangePct`（按涨跌幅排序），`sortOrder` 传 1 即 `MACSortOrderDesc`（降序）；示例保持
	// 原始数字透传演示。
	items, err := client.MACBoardMembersQuotesWithSort("880761", 20, 14, 1)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("mac_member_quote sort_type=%d sort_order=%d symbol=%s name=%s close=%.2f pre_close=%.2f turnover=%.2f pe_static=%.2f",
			14, 1, item.Symbol, item.Name, item.Close, item.PreClose, item.TurnoverRate, item.PEStatic)
	}
}
