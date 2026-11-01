// Command stock_history_transaction_with_trans 演示查询指定历史日期带买卖方向标注的逐笔成交明细，并打印时间、价格、数量与方向。
//
// 运行示例：`go run ./examples/stock_history_transaction_with_trans`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	// 日期 `20260410` 为目标交易日，按 `YYYYMMDD` 打包成整数；市场 `types.MarketSZ`（深市），代码 `000001`。
	// 全量版本自动按固定页大小向前翻页、新页前置保持时间顺序，合并返回该日全部带买卖方向的逐笔成交；
	// 下一行是分页取数版本（`start=0` 起始成交序号、`count=20` 本页笔数），已注释仅作对照。
	// items, err := client.StockHistoryTransactionWithTrans(20260410, types.MarketSZ.Uint8(), "000001", 0, 20)
	items, err := client.StockHistoryFullTransactionWithTrans(20260410, types.MarketSZ.Uint8(), "000001")
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("history_trans_with_dir time=%s price=%.2f vol=%d num=%d action=%s",
			item.Time, item.Price, item.Vol, item.Num, item.Action)
	}
}
