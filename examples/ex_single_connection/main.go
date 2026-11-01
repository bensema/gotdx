// Command ex_single_connection 演示显式单连接模式：连接池大小为 1 并关闭空数据换节点，
// 以美股 `TSLA` 历史成交明细为例，展示单并发场景下接近老库固定单连接的窄尾延迟行为。
//
// 运行示例：`go run ./examples/ex_single_connection`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	// `WithSingleConnectionMode` 把池大小置为 1 并关闭空数据换节点；断连自愈（心跳判死、
	// 后台补建、重平衡）仍然保留，区别只在于不再为一次空应答并行探测其他连接。
	client := exampleutil.NewExSingleConnectionClient()
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
