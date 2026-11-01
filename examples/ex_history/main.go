// Command ex_history 演示通过 ExHistoryTransaction 查询扩展市场指定交易日的历史成交明细，以美股 `TSLA` 为例仅打印前 20 条。
//
// 运行示例：`go run ./examples/ex_history`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	// 参数 `date` 为目标交易日，按 `YYYYMMDD` 打包为整数 `20260330`；`category` 为美股类别；
	// `code` 为标的代码 `TSLA`。
	items, err := client.ExHistoryTransaction(20260330, types.ExCategoryUSStock, "TSLA")
	if err != nil {
		log.Fatalln(err)
	}

	// 成交明细可能较长，用 `min(20, len(items))` 兜底截取前 20 条，打印时间、价格、数量与买卖方向。
	for _, item := range items[:min(20, len(items))] {
		log.Printf("time=%s price=%d vol=%d action=%s", item.Time, item.Price, item.Vol, item.Action)
	}
}

// min 返回 `a`、`b` 中较小的整数，用于把截取条数限制在切片实际长度以内，避免越界。
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
